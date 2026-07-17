/**
 * Admin smoke specs under Instagram UA profiles.
 *
 * (a) UI login: fill the real login form (admin/syrup), submit, assert
 *     redirect to /admin/dashboard. This spec deliberately does NOT use
 *     the API login — validating the actual IG-browser form path (with
 *     the double-submit CSRF pair handled by the browser) is the point.
 *
 * (b) Manage flow: seed waffle + claim spots via API, then reuse the API
 *     login token as the browser's storage state (admin_token cookie) so
 *     no second UI form login hammers the login throttle. On the manage
 *     page (route param is the SLUG, admin.go:112-115): mark spot #5 paid
 *     via grid click + AdminConfirm #admin-confirm-ok, release spot #7
 *     via #pending-list .release-btn + the same confirm modal.
 *
 * (c) Negative login: a deliberately NONEXISTENT username only. The
 *     throttle keys on IP+username and locks for 15 min after 5 failures
 *     (login_throttle.go:9-11), so a failed login as the seeded super
 *     user would poison the suite and immediate reruns — never do it.
 *     After the error renders, a real login proves there is no lockout
 *     contamination (the failure lands on a different throttle key,
 *     login_throttle.go:39-41).
 */

import { test, expect } from '@playwright/test';
import { loginAPI, createWaffleAPI } from '../helpers/seed';

// Minimal Node env typing — @types/node is intentionally not an e2e dep.
declare const process: { env: Record<string, string | undefined> };

const BASE_URL = process.env.BASE_URL ?? 'http://app:8383';

test.describe('Admin login', () => {
  test('UI login with default credentials redirects to dashboard', async ({ page }) => {
    await page.goto('/admin/login');

    await page.locator('#username').fill('admin');
    await page.locator('#password').fill('syrup');
    // Scoped to #login-form-wrapper: the page renders a second submit
    // (Send Reset Link) inside the hidden forgot-password wrapper.
    await page.locator('#login-form-wrapper button[type="submit"]').click();

    await expect(page).toHaveURL(/\/admin\/dashboard/);
    await expect(page.locator('h1')).toContainText('Dashboard');
  });
});

test.describe('Admin manage flow', () => {
  let waffleSlug: string;
  let waffleId: string;

  test.beforeEach(async ({ context }) => {
    const token = await loginAPI(BASE_URL);
    const waffle = await createWaffleAPI(BASE_URL, token, {
      title: 'Admin Manage Waffle',
      total_spots: 20,
      spot_price: 5,
    });
    waffleSlug = waffle.slug;
    waffleId = waffle.id;

    // Reuse the API login as browser storage state — the same JWT the
    // server sets in the admin_token cookie on form login (auth.go:162-170).
    // Avoids a second UI login per spec (login throttle hygiene).
    await context.addCookies([
      {
        name: 'admin_token',
        value: token,
        url: BASE_URL,
        httpOnly: true,
        secure: false,
        sameSite: 'Strict',
      },
    ]);
  });

  test('mark pending spot paid and release another via admin UI', async ({ page, request }) => {
    // Claim spots 5 + 7 via the public API (payload shape per main.go createClaim).
    const claimRes = await request.post(`${BASE_URL}/api/claims`, {
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({
        waffle_id: waffleId,
        spots: [5, 7],
        instagram_handle: 'e2e_admin_manage',
      }),
    });
    expect(claimRes.ok()).toBeTruthy();

    await page.goto(`/admin/waffles/${waffleSlug}`);
    await expect(page.locator('#spot-grid')).toBeVisible();

    // Grid click on a pending spot only OPENS the AdminConfirm modal
    // (admin-spot-actions.js:58-61) — nothing happens without #admin-confirm-ok.
    const spot5 = page.locator('#spot-grid .admin-spot-item[data-spot-number="5"]');
    await expect(spot5).toHaveAttribute('data-spot-status', 'pending');
    await spot5.click();

    const confirmOk = page.locator('#admin-confirm-ok');
    await expect(confirmOk).toBeVisible();
    await confirmOk.click();

    await expect(spot5).toHaveAttribute('data-spot-status', 'paid');

    // Release spot #7 from the pending list (+ its confirm modal).
    const releaseBtn = page.locator('#pending-list .release-btn[data-spot-number="7"]');
    await expect(releaseBtn).toBeVisible();
    await releaseBtn.click();
    await expect(confirmOk).toBeVisible();
    await confirmOk.click();

    // The fetch handler only removes the pending-list row; the grid flip to
    // "available" arrives via the WS broadcast (admin.go:357), so this stays
    // an auto-retrying assertion with the same budget as the WS spec.
    const spot7 = page.locator('#spot-grid .admin-spot-item[data-spot-number="7"]');
    await expect(spot7).toHaveAttribute('data-spot-status', 'available', { timeout: 10000 });
  });
});

test.describe('Negative login (lockout safety)', () => {
  test('unknown username renders error and real login still succeeds', async ({ page }) => {
    // Every test gets a fresh browser context, so no admin_token cookie
    // exists and LoginPage cannot redirect us away (auth.go:80-105).
    await page.goto('/admin/login');

    await page.locator('#username').fill('e2e_no_such_user');
    await page.locator('#password').fill('e2e_not_the_password');
    await page.locator('#login-form-wrapper button[type="submit"]').click();

    // Failed login re-renders the form with the error (auth.go:122-131).
    const errorAlert = page.locator('.alert-error');
    await expect(errorAlert).toBeVisible();
    await expect(errorAlert).toContainText('Invalid username or password');

    // A real login immediately after proves the failure above did not
    // lock anything out for the seeded super user.
    await page.locator('#username').fill('admin');
    await page.locator('#password').fill('syrup');
    await page.locator('#login-form-wrapper button[type="submit"]').click();

    await expect(page).toHaveURL(/\/admin\/dashboard/);
  });
});
