/**
 * Public smoke specs + service worker registration assertions.
 *
 * Runs under both ig-ios and ig-android Playwright projects.
 * Max 1-2 buyer page loads total (RateLimitBuyer 20 burst / 2s refill).
 *
 * Note on service worker tests: Playwright's isMobile:true (required for
 * mobile-first layout) disables navigator.serviceWorker in Chromium.
 * We verify the SW code path by checking the DOM for the registration
 * script and the /admin/ guard — this is the strongest assertion possible
 * headless. Runtime SW behavior must be tested on real devices (see the
 * manual checklist at docs/ig-browser-checklist.md).
 */
import { test, expect } from '@playwright/test';

test.describe('Public smoke', () => {
  test('(a) home page renders waffle list region', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('The Waffle Maker');
    await expect(page.locator('a[href="/waffles"]')).toBeVisible();
  });

  test('(b) waffles list page renders', async ({ page }) => {
    await page.goto('/waffles');
    await expect(page.locator('h1')).toContainText('Active Waffles');
  });

  test('(c) buyer card page renders zero-state for fresh handle', async ({ page }) => {
    const freshHandle = 'e2e_fresh_card';
    await page.goto(`/buyer/${freshHandle}/card`);
    await expect(page.locator('h1')).toContainText(`@${freshHandle}`);
    await expect(page.locator('text=No trophies yet')).toBeVisible();
  });
});

test.describe('Service worker registration', () => {
  test('public page includes SW registration script targeting /static/js/sw.js', async ({ page }) => {
    await page.goto('/');
    // base.html:63-77 registers /static/js/sw.js on non-/admin/ pages.
    // navigator.serviceWorker is unavailable under Playwright's isMobile:true,
    // so we assert the registration code path is present in the DOM.
    const hasSwScript = await page.evaluate(() => {
      const scripts = document.querySelectorAll('script');
      for (const script of scripts) {
        if (script.textContent && script.textContent.includes("register('/static/js/sw.js')")) {
          return true;
        }
      }
      return false;
    });
    expect(hasSwScript).toBeTruthy();
  });

  test('SW registration guard skips /admin/ paths', async ({ page }) => {
    await page.goto('/');
    // The guard condition checks !window.location.pathname.startsWith('/admin/')
    // before calling navigator.serviceWorker.register().
    const hasAdminGuard = await page.evaluate(() => {
      const scripts = document.querySelectorAll('script');
      for (const script of scripts) {
        if (script.textContent && script.textContent.includes("register('/static/js/sw.js')")) {
          return script.textContent.includes("startsWith('/admin/')");
        }
      }
      return false;
    });
    expect(hasAdminGuard).toBeTruthy();
  });

  test('admin login page SW guard prevents registration on /admin/ paths', async ({ page }) => {
    // login.html extends base.html (which includes the SW script), but the
    // guard condition `!window.location.pathname.startsWith('/admin/')` prevents
    // registration. We verify the guard is present and the pathname matches.
    await page.goto('/admin/login');
    const result = await page.evaluate(() => {
      const scripts = document.querySelectorAll('script');
      for (const script of scripts) {
        if (script.textContent && script.textContent.includes("register('/static/js/sw.js')")) {
          // The guard: if ('serviceWorker' in navigator && !window.location.pathname.startsWith('/admin/'))
          return {
            hasSwScript: true,
            hasAdminGuard: script.textContent.includes("startsWith('/admin/')"),
            pathIsAdmin: window.location.pathname.startsWith('/admin/'),
          };
        }
      }
      return { hasSwScript: false, hasAdminGuard: false, pathIsAdmin: false };
    });
    expect(result.hasSwScript).toBeTruthy();
    expect(result.hasAdminGuard).toBeTruthy();
    expect(result.pathIsAdmin).toBeTruthy();
  });
});