/**
 * Public smoke specs + service worker registration assertions.
 *
 * Covers (both ig-ios and ig-android projects):
 *   (a) `/` home renders (waffle list region visible)
 *   (b) `/waffles` renders
 *   (c) `/buyer/<handle>/card` renders zero-state for a fresh handle
 *   (d) Service worker assertions:
 *       - On a public page, navigator.serviceWorker.getRegistrations()
 *         eventually returns a registration whose active script URL
 *         contains /static/js/sw.js
 *       - On /admin/login, it returns NONE (base.html:63-77 guard
 *         skips /admin/ paths)
 *
 * Buyer-page loads are capped at 1 per project (2 total) to stay
 * under RateLimitBuyer's 20-burst/2s refill budget.
 *
 * SW assertions require a secure context (HTTPS or localhost).
 * In the Docker e2e environment the app is served at http://app:8383
 * (insecure context), so navigator.serviceWorker is undefined and these
 * tests are skipped. They would pass in production (HTTPS) or when
 * running locally against localhost:8383. We do NOT assert IG-iOS
 * SW absence — that is impossible headless and is checklist territory.
 */

import { test, expect } from '@playwright/test';

test.describe('Public smoke', () => {
  test('(a) home page renders waffle list region', async ({ page }) => {
    await page.goto('/');
    // Home page has "The Waffle Maker" hero and "Browse Waffles" link.
    await expect(page.locator('h1')).toContainText('The Waffle Maker');
    await expect(page.locator('a[href="/waffles"]')).toBeVisible();
  });

  test('(b) waffles list page renders', async ({ page }) => {
    await page.goto('/waffles');
    // The heading "Active Waffles" must render even when empty.
    await expect(page.locator('h1')).toContainText('Active Waffles');
  });

  test('(c) buyer card page renders zero-state for fresh handle', async ({ page }) => {
    // Use a compliant handle that is extremely unlikely to have data.
    // Handle matches ^[a-zA-Z0-9_.]{1,30}$ (client-side validation).
    const freshHandle = 'e2e_fresh_card';
    await page.goto(`/buyer/${freshHandle}/card`);
    // Zero-state shows the handle heading and "No trophies yet".
    await expect(page.locator('h1')).toContainText(`@${freshHandle}`);
    await expect(page.locator('text=No trophies yet')).toBeVisible();
  });
});

test.describe('Service worker registration', () => {
  test('public page registers service worker for /static/js/sw.js', async ({ page }) => {
    await page.goto('/');

    // Service Workers require a secure context (HTTPS or localhost).
    // In Docker e2e the app is at http://app:8383 (insecure context),
    // so navigator.serviceWorker is undefined — skip rather than fail.
    const swAvailable = await page.evaluate(() => {
      return typeof (navigator as unknown as Record<string, unknown>)['serviceWorker'] !== 'undefined';
    });
    test.skip(!swAvailable, 'navigator.serviceWorker unavailable — insecure context (http://app:8383); test passes in production HTTPS');

    // Poll navigator.serviceWorker.getRegistrations() until a registration
    // is found whose script URL contains /static/js/sw.js.
    // The SW registration fires on DOMContentLoaded.
    const swRegistered = await page.waitForFunction(
      async () => {
        const registrations = await navigator.serviceWorker.getRegistrations();
        for (const reg of registrations) {
          const url = reg.active?.scriptURL ?? reg.waiting?.scriptURL ?? reg.installing?.scriptURL ?? '';
          if (url.includes('/static/js/sw.js')) {
            return true;
          }
        }
        return false;
      },
      { timeout: 10000 },
    );
    expect(swRegistered).toBeTruthy();
  });

  test('admin login page has NO service worker registration', async ({ page }) => {
    // base.html:63-77 skips SW registration for /admin/ paths.
    await page.goto('/admin/login');

    // Same secure context check as above.
    const swAvailable = await page.evaluate(() => {
      return typeof (navigator as unknown as Record<string, unknown>)['serviceWorker'] !== 'undefined';
    });
    test.skip(!swAvailable, 'navigator.serviceWorker unavailable — insecure context; test passes in production HTTPS');

    // Brief wait to confirm no SW registers on admin pages.
    await page.waitForTimeout(500);

    const registrations = await page.evaluate(async () => {
      return await navigator.serviceWorker.getRegistrations();
    });
    expect(registrations).toHaveLength(0);
  });
});