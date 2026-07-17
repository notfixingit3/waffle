/**
 * WebSocket live-update spec under Instagram UA profiles.
 *
 * Seeds a waffle via API, opens /waffle/<slug>, then:
 *   1. Assert #ws-status span:last-child reaches text "live"
 *   2. While the page stays open, claim a DIFFERENT available spot
 *      via the API from the test runner (page.request / fetch), and
 *      assert the page's spot element flips data-spot-status to
 *      "pending" WITHOUT reload — proving WS broadcast drives live UI.
 *
 * The client closes idle sockets after 90s (websocket-client.js:47-53),
 * so keep specs well under that threshold or expect a reconnect cycle.
 */

import { test, expect } from '@playwright/test';
import { loginAPI, createWaffleAPI } from '../helpers/seed';

declare const process: { env: Record<string, string | undefined> };

const BASE_URL = process.env.BASE_URL ?? 'http://app:8383';

test.describe('WebSocket live update', () => {
  let waffleSlug: string;
  let waffleId: string;
  let authToken: string;

  test.beforeEach(async () => {
    authToken = await loginAPI(BASE_URL);
    const waffle = await createWaffleAPI(BASE_URL, authToken, {
      title: 'WS Test Waffle',
      total_spots: 20,
      spot_price: 5,
    });
    waffleSlug = waffle.slug;
    waffleId = waffle.id;
  });

  test('ws-status reaches live and spot flips to pending via broadcast', async ({ page }) => {
    // 1. Open the waffle page
    await page.goto(`/waffle/${waffleSlug}`);

    // Assert #ws-status label reaches "live" (auto-retrying).
    // The WS client connects on DOMContentLoaded and sets label to "live"
    // on open (websocket-client.js:114-153).
    const wsLabel = page.locator('#ws-status span:last-child');
    await expect(wsLabel).toHaveText('live', { timeout: 10000 });

    // 2. Claim a different available spot via API from the test runner,
    //    NOT through the page UI. This proves the WS broadcast path.
    //    Spot #3 is available (fresh waffle, no UI interaction claimed it).
    const claimResponse = await page.request.post(`${BASE_URL}/api/claims`, {
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({
        waffle_id: waffleId,
        spots: [3],
        instagram_handle: 'e2e_ws_broadcast',
      }),
    });
    expect(claimResponse.ok()).toBeTruthy();

    // 3. Assert the spot #3 element flips to data-spot-status="pending"
    //    WITHOUT page reload (WS broadcast drives the UI update).
    const spot3 = page.locator('#spot-grid button.spot-item[data-spot-number="3"]');
    await expect(spot3).toHaveAttribute('data-spot-status', 'pending', { timeout: 10000 });
  });
});