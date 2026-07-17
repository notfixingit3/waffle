/**
 * Claim flow spec under Instagram UA profiles.
 *
 * beforeEach seeds a fresh waffle via createWaffleAPI so reruns never
 * collide with previously claimed spots. The full flow is:
 *   1. Navigate to /waffle/<slug>
 *   2. Assert grid rendered
 *   3. Tap 2 available spots
 *   4. Fill #instagram-handle with a compliant handle
 *   5. Tap #claim-btn
 *   6. Assert #claim-success visible containing "spots claimed by @"
 *   7. Assert the 2 spot buttons now have data-spot-status="pending"
 *
 * #claim-success auto-hides after 5s (spot-selection.js:188-190), so we
 * use Playwright's auto-retrying assertions (toBeVisible / toContainText)
 * instead of sleep-then-assert.
 *
 * Rate limit note: RateLimitClaims is a per-IP token bucket (10 burst +
 * 1/6s refill, rate_limit.go:70). Each spec run makes exactly 1 claim
 * (2 spots). With 2 projects x 1 spec = 2 claims, well under the budget.
 * If a 429 appears (error text "rate limit exceeded", Retry-After: 6),
 * treat it as an infra flake: wait 6s and retry once. Never as an IG
 * regression. (Documented here, not coded — single-project retry config
 * in playwright.config.ts handles infra flake.)
 *
 * Test handles must match ^[a-zA-Z0-9_.]{1,30}$ (client-side validation
 * at spot-selection.js:117).
 */

import { test, expect } from '@playwright/test';
import { loginAPI, createWaffleAPI } from '../helpers/seed';

// Minimal Node env typing — @types/node is intentionally not an e2e dep.
declare const process: { env: Record<string, string | undefined> };

// Fresh waffle per spec — no retries needed since beforeEach re-seeds.
test.describe.configure({ retries: 0 });

const BASE_URL = process.env.BASE_URL ?? 'http://app:8383';

test.describe('Claim flow', () => {
  let waffleSlug: string;
  let waffleId: string;

  test.beforeEach(async () => {
    const token = await loginAPI(BASE_URL);
    const waffle = await createWaffleAPI(BASE_URL, token, {
      title: 'Claim Test Waffle',
      total_spots: 20,
      spot_price: 5,
    });
    waffleSlug = waffle.slug;
    waffleId = waffle.id;
  });

  test('claim 2 available spots and verify pending status', async ({ page }) => {
    // 1. Navigate to waffle detail page
    await page.goto(`/waffle/${waffleSlug}`);

    // 2. Assert grid rendered — at least one spot button exists
    const grid = page.locator('#spot-grid');
    await expect(grid).toBeVisible();

    // 3. Tap 2 available spots
    const availableSpots = grid.locator('button.spot-item[data-spot-status="available"]');
    await expect(availableSpots).toHaveCount(20); // fresh waffle = 20 available
    await availableSpots.nth(0).tap();
    await availableSpots.nth(1).tap();

    // 4. Fill #instagram-handle with a compliant handle
    const handleInput = page.locator('#instagram-handle');
    await handleInput.fill('e2e_claim_test');

    // 5. Tap #claim-btn
    const claimBtn = page.locator('#claim-btn');
    await claimBtn.tap();

    // 6. Assert #claim-success visible containing "spots claimed by @"
    //    (#claim-success auto-hides after 5s — auto-retrying assertion
    //    catches it before it fades)
    const successMsg = page.locator('#claim-success');
    await expect(successMsg).toBeVisible();
    await expect(successMsg).toContainText('claimed by @e2e_claim_test');

    // 7. Assert the 2 claimed spots now have data-spot-status="pending"
    const spot1 = grid.locator('button.spot-item[data-spot-number="1"]');
    const spot2 = grid.locator('button.spot-item[data-spot-number="2"]');
    await expect(spot1).toHaveAttribute('data-spot-status', 'pending');
    await expect(spot2).toHaveAttribute('data-spot-status', 'pending');
  });
});