# Instagram In-App Browser Device Checklist

## Purpose

This checklist verifies that the public claim flow and core admin actions work inside the actual Instagram app on real iOS and Android devices. A separate Playwright test suite in `e2e/` already catches regressions by simulating Instagram user-agent strings and mobile viewports. This document covers the quirks that automation cannot reproduce: Meta's injected scripts, service-worker limitations, storage wipes, keyboard viewport shifting, memory-pressure WebSocket drops, and Instagram's dynamic chrome.

**Two-layer strategy:**

1. Automated simulation in `e2e/` catches layout, flow, and WebSocket regressions cheaply.
2. This manual checklist catches the device-level Instagram behaviors that no headless browser can fake.

## Getting the URL into Instagram

Instagram's in-app browser does not let you type a URL. Use one of these methods instead:

- **DM the link to yourself.** Open Instagram, start a direct message to yourself, paste the URL, send it, then tap the link in the chat.
- **Use a link-in-bio.** Put the URL in an Instagram bio or link-sticker, then tap it from the same account or a second test account.
- **To inspect what Meta injects into pages,** open https://inappdebugger.com inside Instagram using either method above. This shows you the available JavaScript APIs and any injected scripts (for example, Meta's `pcm.js`).

## Environment

Pick one test environment and use it for the whole checklist. Do not mix environments within a single finding.

- **Option A: Local LAN stack.** Run `docker compose up --build` on your computer, then use `http://<your-computer-LAN-IP>:8383` on each phone. Default local credentials `admin` / `syrup` work here. The phones must be on the same Wi-Fi network as the computer.
  - Trade-off: Fast, free, and resets cleanly; requires the phones and laptop to share a network, and the URL changes with your LAN IP.

- **Option B: Dev demo URL.** Use `https://dev.waffle.projectsyrup.app`. Use your **real** admin credentials. Never use `admin` / `syrup` there.
  - Trade-off: Matches real user geography and SSL; requires a working admin account and any claim changes persist until cleared.

For each track, record which environment you used in the findings doc.

## What you need

- One iPhone with the Instagram app.
- One Android phone with the Instagram app.
- The test environment URL loaded into both phones via DM or link-in-bio.
- For multi-user steps, a second Instagram account or a friend with a phone.
- A way to take screenshots of failures.

## How to fail forward

- If a step fails, stop the track, record it in `docs/ig-browser-findings.md`, and include the track name, environment, exact step, expected result, actual result, and a screenshot.
- Do not attempt workarounds unless the checklist specifically asks for one.
- A failure in one track does not block other tracks.

---

## Track 1: Public Flow on iOS

Run every step on an iPhone inside the Instagram in-app browser.

### 1. Open the waffle page

- [ ] Tap the DM or link-in-bio URL to open the app.
- **Expected:** The page loads within 5 seconds and shows the waffle title, item image, price per spot, and the spot grid.
- **FAIL:** Record in findings doc. Include loading time and whether the grid appeared.

### 2. Compare the grid to the desktop view

- [ ] Open the same waffle page on a desktop browser (or ask a teammate) and compare the spot grid layout.
- **Expected:** Grid is readable, spot numbers are visible, 5 columns on the small screen, and buttons are tall enough to tap without zooming.
- **FAIL:** Record in findings doc. Note whether spots are cut off, overlapping, or too small to tap accurately.

### 3. Select available spots

- [ ] Tap two available spots. They should highlight or change appearance.
- **Expected:** Selected spots are clearly marked, and the Claim button updates to show the count and total price.
- **FAIL:** Record in findings doc. Note whether taps register, duplicate selections occur, or the button text does not update.

### 4. Open the keyboard on the Instagram handle input

- [ ] Tap the Instagram handle input field. Type a valid handle (letters, numbers, underscores, periods only; no leading `@`).
- **Expected:** Keyboard opens smoothly, the input accepts characters, and the leading `@` is removed if you type one by accident.
- **FAIL:** Record in findings doc. Note whether the keyboard covers the input or the field rejects valid characters.

### 5. Close the keyboard and tap Claim

- [ ] Dismiss the keyboard (tap the Done key or swipe down), then tap the Claim button.
- **Expected:** The button responds to the tap exactly where it appears. A success message appears with your handle, and the two spots turn yellow pending.
- **FAIL:** Record in findings doc. This is the main iOS click-misalignment test. Note if you had to tap twice, aim above the button, or if the page jumped after the keyboard closed.

### 6. Check the live indicator

- [ ] Look at the live indicator near the top of the spot grid.
- **Expected:** The indicator shows a green dot and the text "live".
- **FAIL:** Record in findings doc. Note if it stays on "connecting" or "offline".

### 7. Cross-device live update

- [ ] On a second phone or a friend's phone, open the same waffle page and claim a different spot.
- **Expected:** The first phone's grid updates the new spot to yellow pending without pulling to refresh or leaving the page.
- **FAIL:** Record in findings doc. Include whether the update happened after a delay, only after refresh, or not at all.

### 8. Background and return

- [ ] Send Instagram to the background for 60 seconds, then return to the waffle page. Do not close the page.
- **Expected:** The live indicator reconnects and shows "live" again. The spot grid still shows the correct state.
- **FAIL:** Record in findings doc. Note if the indicator says "connection lost", if the grid is blank, or if the page requires a refresh to recover.

### 9. Scroll the whole page

- [ ] Scroll from the top of the page to the bottom and back up.
- **Expected:** The header and footer stay in place, nothing is hidden by the bottom Instagram chrome, and the page height feels stable.
- **FAIL:** Record in findings doc. Note any jumpiness, elements hidden by the Instagram toolbar, or footer floating above the bottom of the screen.

### 10. Navigate away and back

- [ ] Tap a link in the page (for example, the home link or a buyer card link), then use the Instagram back button to return to the waffle page.
- **Expected:** The waffle page reloads from the network and the spot grid reappears. The claim form still works.
- **FAIL:** Record in findings doc. This checks localStorage wipe. Note any offline error banners, blank grids, or broken claim flow after returning.

### 11. Service worker absence on iOS

- [ ] Confirm that the page works even though iOS Instagram does not expose service workers.
- **Expected:** No error banner appears. You can still select spots, enter a handle, and submit a claim. The page loads from the network each time.
- **FAIL:** Record in findings doc. Note any "offline" or "service worker failed" messages that block the claim flow.

### 12. Back-button behavior

- [ ] Tap the Instagram back arrow from the waffle page.
- **Expected:** You return to the previous page (home or the DM conversation) without the app freezing or reloading the current page in a loop.
- **FAIL:** Record in findings doc. Note any infinite back-button loops, blank screens, or unexpected exits.

---

## Track 2: Public Flow on Android

Run every step on an Android phone inside the Instagram in-app browser.

### 1. Open the waffle page

- [ ] Tap the DM or link-in-bio URL to open the app.
- **Expected:** The page loads within 5 seconds and shows the waffle title, item image, price per spot, and the spot grid.
- **FAIL:** Record in findings doc.

### 2. Compare the grid to the desktop view

- [ ] Open the same waffle page on a desktop browser and compare the spot grid layout.
- **Expected:** Grid is readable, spot numbers are visible, 5 columns on the small screen, and buttons are tall enough to tap without zooming.
- **FAIL:** Record in findings doc.

### 3. Select available spots

- [ ] Tap two available spots. They should highlight or change appearance.
- **Expected:** Selected spots are clearly marked, and the Claim button updates to show the count and total price.
- **FAIL:** Record in findings doc.

### 4. Open the keyboard on the Instagram handle input

- [ ] Tap the Instagram handle input field. Type a valid handle (letters, numbers, underscores, periods only; no leading `@`).
- **Expected:** Keyboard opens smoothly, the input accepts characters, and the leading `@` is removed if you type one by accident.
- **FAIL:** Record in findings doc.

### 5. Close the keyboard and tap Claim

- [ ] Dismiss the keyboard, then tap the Claim button.
- **Expected:** The button responds to the tap exactly where it appears. A success message appears with your handle, and the two spots turn yellow pending.
- **FAIL:** Record in findings doc. Note any Android-specific keyboard-close misalignment here.

### 6. Check the live indicator

- [ ] Look at the live indicator near the top of the spot grid.
- **Expected:** The indicator shows a green dot and the text "live".
- **FAIL:** Record in findings doc.

### 7. Cross-device live update

- [ ] On a second phone or a friend's phone, open the same waffle page and claim a different spot.
- **Expected:** The first phone's grid updates the new spot to yellow pending without pulling to refresh or leaving the page.
- **FAIL:** Record in findings doc.

### 8. Background and return

- [ ] Send Instagram to the background for 60 seconds, then return to the waffle page. Do not close the page.
- **Expected:** The live indicator reconnects and shows "live" again. The spot grid still shows the correct state.
- **FAIL:** Record in findings doc.

### 9. Scroll the whole page

- [ ] Scroll from the top of the page to the bottom and back up.
- **Expected:** The header and footer stay in place, nothing is hidden by the bottom Instagram chrome, and the page height feels stable.
- **FAIL:** Record in findings doc.

### 10. Navigate away and back

- [ ] Tap a link in the page, then use the Instagram back button to return to the waffle page.
- **Expected:** The waffle page reloads from the network and the spot grid reappears. The claim form still works.
- **FAIL:** Record in findings doc. This checks localStorage wipe on Android.

### 11. Service worker behavior on Android

- [ ] Confirm the page works after navigation. Android Instagram WebView may support service workers but is unreliable.
- **Expected:** No error banner appears. The page loads from the network if needed, and claims still succeed.
- **FAIL:** Record in findings doc. Note any stale offline banner or broken claim after returning to the page.

### 12. Back-button behavior

- [ ] Tap the Instagram back arrow from the waffle page.
- **Expected:** You return to the previous page without the app freezing or reloading the current page in a loop.
- **FAIL:** Record in findings doc.

---

## Track 3: Admin Flow on iOS

Run every step while logged in as an admin on an iPhone inside the Instagram in-app browser.

### 1. Admin login

- [ ] Tap the DM or link-in-bio URL to `/admin/login`. Enter your admin username and password, then tap Log In.
- **Expected:** You land on `/admin/dashboard` and the dashboard loads with waffle cards and navigation.
- **FAIL:** Record in findings doc. Include login credentials source (local `admin`/`syrup` or dev demo credentials).

### 2. Login persists across navigation

- [ ] Tap the dashboard logo or home link, then navigate back to `/admin/dashboard` without logging in again.
- **Expected:** You are still logged in and the dashboard loads.
- **FAIL:** Record in findings doc. Note if you were redirected to the login page unexpectedly.

### 3. Open a waffle manage page

- [ ] Tap any active waffle card, or navigate directly to `/admin/waffles/<slug>`.
- **Expected:** The manage page loads and shows the spot grid, pending claims list, and action buttons.
- **FAIL:** Record in findings doc.

### 4. Mark a pending spot as paid

- [ ] First, claim a spot using the public flow so there is a pending spot. Then, on the admin manage page, tap that pending spot in the grid.
- **Expected:** A confirmation modal appears. Tap OK in the modal. The spot turns red paid.
- **FAIL:** Record in findings doc. Note if the modal does not open, the spot color does not change, or the tap does nothing.

### 5. Release a pending spot

- [ ] Claim another spot using the public flow so it is pending. On the manage page, find the pending spot in the pending claims list and tap the Release button.
- **Expected:** A confirmation appears, then the spot returns to green available.
- **FAIL:** Record in findings doc.

### 6. Live indicator on admin page

- [ ] Look at the live indicator on the manage page.
- **Expected:** It shows "live" and reflects public claim activity without refresh.
- **FAIL:** Record in findings doc.

### 7. Background and return

- [ ] Send Instagram to the background for 60 seconds, then return to the admin manage page.
- **Expected:** The page reconnects and the admin actions still work.
- **FAIL:** Record in findings doc.

### 8. Navigate away and back

- [ ] Tap the admin dashboard link, then return to the same manage page.
- **Expected:** The page reloads from the network and the admin controls still work. No service worker is registered on admin paths, so navigation must work without cached scripts.
- **FAIL:** Record in findings doc.

### 9. Back-button behavior

- [ ] Tap the Instagram back arrow from the manage page.
- **Expected:** You return to the dashboard without a login prompt or blank screen.
- **FAIL:** Record in findings doc.

---

## Track 4: Admin Flow on Android

Run every step while logged in as an admin on an Android phone inside the Instagram in-app browser.

### 1. Admin login

- [ ] Tap the DM or link-in-bio URL to `/admin/login`. Enter your admin username and password, then tap Log In.
- **Expected:** You land on `/admin/dashboard` and the dashboard loads with waffle cards and navigation.
- **FAIL:** Record in findings doc. Include login credentials source.

### 2. Login persists across navigation

- [ ] Tap the dashboard logo or home link, then navigate back to `/admin/dashboard` without logging in again.
- **Expected:** You are still logged in and the dashboard loads.
- **FAIL:** Record in findings doc.

### 3. Open a waffle manage page

- [ ] Tap any active waffle card, or navigate directly to `/admin/waffles/<slug>`.
- **Expected:** The manage page loads and shows the spot grid, pending claims list, and action buttons.
- **FAIL:** Record in findings doc.

### 4. Mark a pending spot as paid

- [ ] First, claim a spot using the public flow so there is a pending spot. Then, on the admin manage page, tap that pending spot in the grid.
- **Expected:** A confirmation modal appears. Tap OK in the modal. The spot turns red paid.
- **FAIL:** Record in findings doc.

### 5. Release a pending spot

- [ ] Claim another spot using the public flow so it is pending. On the manage page, find the pending spot in the pending claims list and tap the Release button.
- **Expected:** A confirmation appears, then the spot returns to green available.
- **FAIL:** Record in findings doc.

### 6. Live indicator on admin page

- [ ] Look at the live indicator on the manage page.
- **Expected:** It shows "live" and reflects public claim activity without refresh.
- **FAIL:** Record in findings doc.

### 7. Background and return

- [ ] Send Instagram to the background for 60 seconds, then return to the admin manage page.
- **Expected:** The page reconnects and the admin actions still work.
- **FAIL:** Record in findings doc.

### 8. Navigate away and back

- [ ] Tap the admin dashboard link, then return to the same manage page.
- **Expected:** The page reloads from the network and the admin controls still work.
- **FAIL:** Record in findings doc.

### 9. Back-button behavior

- [ ] Tap the Instagram back arrow from the manage page.
- **Expected:** You return to the dashboard without a login prompt or blank screen.
- **FAIL:** Record in findings doc.

---

## Coverage Checklist

Use this table to confirm every required area was tested on at least one track. Place a checkmark in each cell once a step covering that area has passed or failed on that track.

| Area | Public iOS | Public Android | Admin iOS | Admin Android |
|------|------------|----------------|-----------|---------------|
| URL delivery (DM/link-in-bio) | | | | |
| Meta injection awareness (inappdebugger.com) | | | | |
| Spot grid render | | | | |
| Select spots | | | | |
| Keyboard open/close + Claim tap | | | | |
| Claim success feedback | | | | |
| Live indicator | | | | |
| Cross-device live update | | | | |
| Background-and-return (memory/WS drop) | | | | |
| 100vh / safe-area scroll | | | | |
| Navigate away and back (localStorage wipe) | | | | |
| Service worker absence on iOS | | | | |
| Admin login persists | | | | |
| Admin mark paid + release modal | | | | |
| Back-button behavior | | | | |

## Record Findings Here

When a step fails, copy the relevant lines into `docs/ig-browser-findings.md` and fill in every field. Do not edit this checklist to describe the failure; the findings doc is the single source of truth for defects.

**FAIL → record in findings doc** means: open `docs/ig-browser-findings.md`, add a row with the date, platform, track, reproduction steps, expected result, actual result, and a screenshot. If you are unsure of the severity, default to Medium and the team can change it during review.
