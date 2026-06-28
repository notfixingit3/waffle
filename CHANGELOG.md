# Changelog

All notable changes to Project Syrup will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
-

### Changed
-

### Fixed
-

## [v0.1.23-beta.5] - 2026-06-27

### Added
- **In-app toast notifications** — New `toast.js` utility (`AdminToast.error/success/warning/info`) provides DaisyUI-styled ephemeral toasts. Replaces all native browser `alert()` calls in admin pages.
- **In-app confirm dialog** — `AdminConfirm.show()` added to `admin_base.html` using a DaisyUI modal `<dialog>`. Replaces native browser `confirm()` for mark-paid and release-spot flows.
- **Bulk pay feedback** — Bulk pay now shows a success toast with the count of spots marked, an error toast for any individual failures, and a network-error toast on catch. Previously silent.

### Changed
- **Spot class consolidation** — `SPOT_BASE_CLASSES` and `SPOT_GRID_CLASSES` constants added to `spot-status-classes.js`. The WebSocket spot-update handler and `admin-spot-actions.js` now reference these instead of duplicating the full Tailwind class string.
- **Spot color consistency fix** — Bulk pay and mark-paid were incorrectly setting `bg-error` on updated grid spots; now correctly uses `bg-error/10` to match the server-rendered template.
- **CSRF header on admin API calls** — All `fetch()` calls in `admin-spot-actions.js` (pay, release, set-winner, change-winner, clear-winner) now send `X-CSRF-Token` read from the page `<meta>` tag, consistent with the existing CSRF middleware.
- **Service worker asset list** — `share-message.js`, `spot-status-classes.js`, and `toast.js` added to `STATIC_ASSETS`; cache version bumped to `v6`.

### Fixed
- **Test DB isolation** — `TEST_DATABASE_URL` env var used in CI and local test runs to prevent tests from touching dev/prod databases. `DATABASE_URL` is cleared in `TestMain` when `TEST_DATABASE_URL` is unset.
- **`pool.Close()` hang** — Removed `db.Pool.Close()` call from both `TestMain` functions; the `golang-migrate` stdlib adapter holds connections asynchronously and caused the test binary to hang until the `go test` timeout fired.
- **Share message URL in test** — `TestRenderWaffleShareMessageAPI_Success` now uses `strings.Contains` instead of an exact URL match, so it passes regardless of `APP_HOST` configuration in CI.
- **Nil pool panics in handler tests** — Added `skipWithoutDB(t)` guard to all handler tests that call DB-backed service functions, preventing panics when Postgres is unavailable.

## [v0.1.23-beta.4] - 2026-06-20

### Fixed
- **Inactive payment methods on public pages** — `GetPaymentMethodsForWaffle` now
  filters `WHERE pm.is_active = true`; deactivated payment methods no longer appear
  on public waffle pages.
- **CSRF header support** — `validateCSRF` now accepts the `X-CSRF-Token` request
  header in addition to the form field, so JSON API endpoints (share message, share
  templates) are properly CSRF-protected.
- **Share message host fallback** — `APP_HOST` added to `docker-compose.prod.yml`;
  persisted share message URLs now use the real deployment hostname instead of the
  `waffle.social` fallback.
- **Buyer history stale fields** — `GetBuyerWaffleHistory` no longer selects the
  deprecated single-winner columns; `IsWinner` is derived from spot status only.
  Removed `WinningSpotNumber`/`WinningInstagramHandle` from `BuyerWaffleHistory` model.

### Changed
- **Rate limiter refactor** — `rate_limit.go` replaces two duplicated `sync.Map` +
  `init()` goroutine patterns with a shared `ipRateLimiter` struct. No behavior change.
- **Buyer endpoint rate limiting** — All `/buyer/:handle` page and API routes now use
  the `RateLimitBuyer` middleware (20-request burst / 2-second window) to prevent
  handle enumeration.
- **Share card cache dir** — `handlers.ShareCardCacheDir` removed; all code now reads
  `services.ShareCardCacheDir` as the single source of truth.
- **Font temp file cleanup** — Share card embedded fonts are now removed from the OS
  temp directory on graceful shutdown via `services.CleanupShareCardFonts()`.

### Security
- **Production DB backups removed from git** — Two committed `.sql.gz` dump files
  deleted from version history. `backups/` added to `.gitignore`.

## [v0.1.23-beta.3] - 2026-06-14

### Added
- **Share Card Rate Limiting** — The `/waffle/:slug/card.png` endpoint is now rate-limited
  to 10 requests per IP per minute with a separate limiter isolated from the claims
  endpoint, preventing share card abuse without affecting spot claims.
- **Test Isolation Fixes** — `TestDeleteMessageTemplate_RejectsLastTemplate` and
  `TestShareTemplatesAPI_DeleteLastTemplate_Returns400` now properly clear FK
  references and all template rows before testing, eliminating flakiness from
  shared DB state. `createCompletedWaffleForBuyer` auto-seeds a default template
  if none exists, preventing FK violations in buyer card tests.

### Changed
- **Admin Waffle Manage UI** — Added "Share to Instagram" section with template
  selector, editable message preview, public claim URL with copy button, story/square
  card format toggle, download card link, regenerate card button, copy message
  button, and save message button.

## [v0.1.23-beta.2] - 2026-06-12

### Added
- **Public `/api/version` endpoint** — Exposes the current application version for
  remote version checks. GET `/api/version` returns `{"version":"v0.1.23-beta.2"}`.

## [v0.1.23-beta.1] - 2026-06-12

### Fixed
- **Archive/Unarchive/Delete Form 404** — Rendered admin dashboard forms posting to
  `/admin/waffles/{id}/{archive,unarchive,delete}` now route correctly. The handlers
  existed but were not registered; they are now wired with admin/super_admin role
  checks and CSRF validation.
- **Gin Route Parameter Conflict** — Renamed existing `/admin/waffles/:slug*` route
  parameters to `:id` so Gin no longer panics on startup from conflicting sibling
  parameter names.

### Added
- **Archive Tests** — Route-level RBAC tests, service-level `ArchiveWaffle` tests,
  and handler-level CSRF/redirect tests for archive, unarchive, and delete forms.

## [v0.1.23-beta.0] - 2026-06-11

### Added
- **Shareable Buyer Cards** — New chromeless `/buyer/:handle/card` page optimized for
  Instagram Story screenshots, with luck rating, trophy case, and OG meta tags.
- **Buyer Card API** — New `GET /api/buyers/:handle/card` endpoint returning
  `BuyerCardData` with computed luck rating and parsed trophy items.
- **Luck Rating** — Statistical comparison of actual win rate vs expected win rate
  (total wins / total completed waffles entered) as a percentage delta.
- **Trophy Parsing** — Automated extraction of double-quoted item names from
  completed waffle titles displayed in a trophy case grid.
- **Buyer Card Tests** — Full Go test coverage for luck rating computation, trophy
  parsing with edge cases, and `ComputeBuyerCardData` against real DB state.

### Changed
- **Buyer Stats Redesign** — Buyer stats page now uses a card-based layout with
  big stat tiles, conditionally displayed luck rating (lucky/due badges), trophy
  case section, and a prominent CTA linking to the shareable card page.
- **Math Template Helpers** — Added `mulf` function to `renderer.go` for float
  multiplication, fixing truncated template rendering when computing luck
  rating percentages.

## [v0.1.22] - 2026-06-10

### Added
- **Stored Payment Methods** — Structured pool (Venmo, PayPal, CashApp, Zelle)
- **Payment Method Admin Page** — /admin/payment-methods with CRUD and soft-delete
- **Payment Method Public Display** — Grouped by type with icons and clickable links
- **Payment Method Tests** — Full test coverage
- **Docker Version Auto-Detect** — Build process extracts version from main.go when VERSION=dev
- **Versioned Dev Docker Tags** — Dev branch Docker builds now publish both the
  floating `dev` tag and the detected app version tag.

### Changed
- **Waffle Create/Edit Forms** — Replaced payment_info textarea with multi-select checkboxes
- **Admin Navigation** — Renamed "Admin Tools" to "Management", moved Payment Methods into dropdown
- **Docker Image Metadata** — OCI `org.opencontainers.image.version` now uses the
  detected app version instead of the branch name for clearer package metadata.
- **Agent Guidance** — Expanded Scooby-Doo commit message guidance and added local
  Docker pre-flight testing reminders for future release bumps.

## [v0.1.21] - 2026-06-10

### Added
- **Random Spot Selection** — Public waffle pages include a "Pick Random Spots" flow;
  buyers enter a count and the app claims available spots automatically.
- **Random Claim API** — `POST /api/claims/random` with rate limiting, handle
  normalization, transactional row locking, and WebSocket broadcast.
- **Partial Fulfillment** — Random claims claim as many spots as available when the
  requested count exceeds remaining availability.
- **Live Stats Updates** — Available/Pending/Paid counts and progress bar update in
  real-time on all spot state changes.

## [v0.1.21-beta.0] - 2026-06-10

### Added
- **Random Spot Selection** — Public waffle pages now include a "Pick Random Spots"
  flow so buyers can enter a spot count and let the app claim available spots for
  them automatically.
- **Random Claim API** — New `POST /api/claims/random` endpoint reuses existing
  claim rate limiting, Instagram handle normalization, transactional row locking,
  and WebSocket spot updates.
- **Partial Fulfillment** — Random claims now claim as many available spots as
  possible when the requested count is higher than remaining availability, then
  return the claimed spot numbers and requested/claimed counts.

### Changed
- **Live Stats Updates** — Available/Pending/Paid counts and progress bar now
  update in real-time when spots are claimed, paid, or released.
- **Disable Random on Manual Selection** — The "Pick Random Spots" button is
  disabled when manual spots are selected to prevent conflicting claim methods.
- **Public Claim UI** — Random claims share the same Instagram handle input as
  manual spot selection while keeping manual and random claim actions independent.
- **Release Attribution** — Random spot selection is credited as a community UI
  feature idea from OrangeSoJuicy on Instagram.

### Fixed
- **No Hard Random Count Limit** — Removed the client-side maximum count so the
  server-side partial fulfillment behavior remains authoritative.

## [v0.1.20] - 2026-06-04

### Added
- **Multiple Items and Winners** — Waffles now support `item_count` and parallel
  winner arrays (`winning_spot_numbers`, `winning_instagram_handles`) so a single
  drop can award multiple prizes.

## [v0.1.19] - 2026-06-04

### ⚠️ Upgrade Notes

**Migrations are now embedded in the binary.**
The app no longer reads migration files from the filesystem at runtime. If your
`docker-compose.prod.yml` has the following volume mount you can safely remove it —
it is no longer needed and the line is not present in the updated compose file:

```yaml
# Remove this — no longer required
volumes:
  - ./backend/migrations:/app/migrations:ro
```

You also no longer need to `rsync` or copy the `backend/migrations/` directory to
your server before deploying. Updating is now a single step:

```bash
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

The old volume mount is harmless if left in place — the app just won't read from it.
But removing it keeps your compose file clean and avoids confusion.

### Added
- **Embedded Migrations** — SQL migration files are compiled directly into the binary
  via `//go:embed`. No filesystem mount or file-sync step required at deploy time.
- **Winner Dropdown** — The Set Winner field on the waffle manage page is now a
  dropdown listing all paid spots with their Instagram handles instead of a free-text
  spot number input.

### Changed
- **CI/CD Workflow** — Tag naming convention now drives release type: tags matching
  `v*.*.*-*` (e.g. `v0.1.19-rc.1`) produce a GitHub pre-release and image with the
  versioned tag only; plain `v*.*.*` tags produce a stable release and also move the
  `latest` and `major.minor` Docker image tags. Old releases are never deleted.
- **Version in Footer** — The Docker build now correctly stamps the image with the
  git tag so the footer shows the real version instead of always displaying `dev`.
- **Dark-only Theme** — Light/dark toggle removed. The app now runs the warm dark
  theme exclusively, aligned with the projectsyrup.app design language.
- **Re-theme** — Base colors updated from purple-grey to warm dark brown to match
  projectsyrup.app (amber accent, warm cream text, glass header).

## [v0.1.18] - 2026-05-31

### Added
- **Users Registry** — New `users` table with `GetOrCreateUser`, `ListUsers`, and `BackfillUsers` service functions, admin users list page and JSON API endpoint
- **User Backfill** — Automatic backfill of existing `claimed_by_handle` values from `spots` table into `users` table on application startup

### Fixed
- **Duplicate Lockout Removal** — Resolved duplicate Instagram handle lockout preventing claim submissions for handles with existing pending/paid spots

### Changed
- **AGENTS.md Updates** — Expanded API conventions with all admin endpoints, added Users Registry implementation notes

## [v0.1.16] - 2026-05-31

### Changed
- **Dev/Stable Release Channels** — Docker workflow dev branch trigger, version bump, CHANGELOG backfill, README channels section

## [v0.1.15] - 2026-05-30

### Added
- **Admin Profile Expansion** — First name, last name, email, and social links fields on admin profile

## [v0.1.14] - 2026-05-30

### Changed
- **Admin UI Polish** — Password change UI, tooltips, responsive layout fix

## [v0.1.13] - 2026-05-29

### Changed
- **Admin Nav Grouping** — Grouped admin navigation under Admin Tools dropdown

## [v0.1.12] - 2026-05-29

### Added
- **Audit Log Nav Prominence** — Moved audit log to top nav, added admin filter, and added server settings tab
- **Role-Permissions Guide** — Inline role-permissions guide on admin users page

## [v0.1.11] - 2026-05-29

### Fixed
- **Archived Waffle Filter** — Admin active and archived waffle lists now show the correct records.
- **Buyer Stats Recalculation** — Clear/change winner actions now refresh buyer win/loss stats.
- **Password Reset API** — Forgot-password JSON response no longer exposes reset tokens.
- **Drought Report Dates** — Missing last-entry dates now render cleanly instead of showing `Invalid Date`.

### Changed
- **Admin Code Cleanup** — Removed unused spot/winner handlers and consolidated shared audit/password confirmation helpers.
- **Accessibility Polish** — Claim success/error feedback now announces via `aria-live`.
- **Audit Export Link** — CSV export filters are URL-encoded.

## [v0.1.10] - 2026-05-29

### Fixed
- **Admin Login Redirect** — Already-authenticated admins are redirected from login to the dashboard.
- **Public Header/Footer Spacing** — Public header and footer vertical spacing now better matches admin layout density.

## [v0.1.9] - 2026-05-29

### Added
- **CI Pipeline** — `dev` branch added to GitHub Actions trigger
- **Audit Export UI** — CSV export button on admin audit log page with date filter support
- **WebSocket Heartbeat** — Server-side ping/pong with per-connection mutex, client-side stale detection
- **Smoke Tests** — Shell script for end-to-end Docker Compose validation
- **Data Retention** — Configurable audit_log and login_history retention (default 90 days)
- **Release Automation** — GitHub Release auto-created on tag push with CHANGELOG excerpt

## [v0.1.8] - 2026-05-28

### Added
- **CI Pipeline** — GitHub Actions workflow with essential checks (go test, vet, govulncheck, Docker build)
- **Audit Log CSV Export** — New API endpoint to export audit log entries as CSV
- **WebSocket Reconnect Logic** — Exponential backoff jitter and max retry cap for resilient client reconnection
- **gosec G104 Triage** — All unhandled error returns reviewed and explicitly handled across the codebase
- **gitignore Update** — `*.sarif` files added to `.gitignore`

### Changed
- Release version bumped to v0.1.8

## [v0.1.7] - 2026-05-27

### Added
- **Admin Audit Log** — Full audit trail with `audit_log` table, service layer, JSON API (list + single entry), and dedicated admin UI page at `/admin/audit`
- **Last Login IP Tracking** — Login history records and displays the IP address of each admin session
- **Brute-Force Lockout** — Rate-limited login endpoints with configurable failed attempt threshold and lockout duration
- **Configurable JWT Expiration** — System setting to control JWT token lifetime
- **Password Policy Enforcement** — Server-side validation enforcing minimum length and rejecting common/weak passwords
- **Destructive Action Confirmation** — Delete, deactivate, and role-demotion operations require current password confirmation before proceeding

## [v0.1.5] - 2026-05-27

### Added
- **Login History** — Audit trail tracking admin logins with IP, browser, OS, and device type parsing
- **WHOIS Enrichment** — Async WHOIS lookups on login to capture org, country, city, and ASN (10s timeout, non-blocking)
- **Private IP Detection** — Skip WHOIS lookups for RFC1918 addresses (10.x, 172.16-31.x, 192.168.x)
- **System Settings** — Configurable WHOIS server (default: whois.pwhois.org), super_admin only
- **Winner Management** — Admin-only endpoints to clear winner (reset to active) and change winner (reassign winning spot)
- **Buyer Stats Recalculation** — Automatic win/loss stat updates when winners are changed or cleared
- **Settings Dropdown** — Consolidated admin nav menu under username with Settings, My Login History, About, Theme, Logout
- **About Page** — Public about page with admin-only system extras section
- **Login History Pages** — My Login History tab on settings page + full admin login history page with role-based filtering
- **Role-Based History Visibility** — waffle_manager sees self only, admin sees self + waffle_managers, super_admin sees all

### Changed
- Updated admin navigation to use dropdown menu instead of separate nav items
- Screenshots updated to show v0.1.5 features

## [v0.1.0] - 2026-05-26

### Added
- **Multi-Admin Auth** — Role-based access control with super_admin, admin, and waffle_manager roles
- **Admin Management** — Create admins, change roles, deactivate accounts, reset passwords (super_admin only)
- **waffle_manager Role** — Create and manage waffles + view reports, without archive/delete/user-management access
- **Timezone Settings** — Per-admin timezone preference with IANA timezone dropdown
- **Password Reset** — Self-service reset tokens plus authenticated password changes
- **Instagram Media Links** — Link to posts showing what's being waffled (supports multiple items)
- **Archive + Delete Controls** — Hide completed waffles by default, or type DELETE for permanent removal
- **Version Footer** — App version injected at build time via ldflags
- **Theme Toggle** — Light/dark mode with persisted preference

### Changed
- Migrated all UI to DaisyUI components with syrup theme
- Redesigned navigation with Inter font and unified brand colors
- Updated admin auth pages to DaisyUI card/input/btn/alert components
- Converted public pages to DaisyUI card/btn/badge components
- Added spot status class map for consistent styling

### Fixed
- Corrected default admin password from admin123 to syrup
- Fixed duplicate IDs for theme toggle icons (now uses classes)
- Removed redundant version text in dev mode footer

## [v0.0.9] - 2026-05-26

### Added
- **DaisyUI Migration** — Complete UI overhaul with DaisyUI component library and syrup color theme
- **Production Deployment** — docker-compose.prod.yml, .env.example, and GHCR image workflow
- **PWA Service Worker** — Offline caching with service worker registration
- **Offline Handling** — Offline banners and toast notifications for public pages
- **Rate Limiting** — Request rate limiting for public endpoints
- **Commit Message Guidelines** — Scooby-Doo quote convention documented

### Changed
- Migrated 5 admin templates to DaisyUI with syrup theme
- Converted layout templates and partials to DaisyUI components
- Updated JavaScript files to use DaisyUI theme tokens and shared status class map
- Rebuilt output.css via Docker Tailwind stage

### Fixed
- Added spot-status-classes.js to go:embed directive (was causing 404s)
- Added spot-status-classes script tag to waffle manage template

## [v0.0.8] - 2026-05-25

### Added
- **Seed Data** — Demo waffle and admin data for fresh installs
- **Navigation Redesign** — Cleaner admin nav with role-based visibility
- **Inter Font** — Modern typography across all pages
- **Brand Color Unification** — Consistent amber/brown color scheme

### Fixed
- Renamed Admin Management to User Management for clarity
- Fixed template bug in admin management page

## [v0.0.5] - 2026-05-24

### Added
- **PWA Support** — Web App Manifest, app icons, standalone display metadata
- **Offline Page** — Cached offline.html for when network is unavailable
- **Phase 8 Complete** — Offline/service worker support with installable app shell

## [v0.0.1] - 2026-05-20

### Added
- **Initial Public Release** — Project Syrup v0.1.0 foundation
- Docker Compose setup with PostgreSQL
- Go backend with Gin framework
- Server-rendered templates with Tailwind CSS
- WebSocket hub for real-time updates
- Basic waffle CRUD operations
- Spot claim and payment tracking
- Admin authentication system

---

[v0.1.23-beta.5]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.5
[v0.1.23-beta.4]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.4
[v0.1.23-beta.3]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.3
[v0.1.23-beta.2]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.2
[v0.1.23-beta.1]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.1
[v0.1.23-beta.0]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.0
[v0.1.22]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.22
[v0.1.21]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.21
[v0.1.21-beta.0]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.21-beta.0
[v0.1.20]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.20
[v0.1.19]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.19
[v0.1.16]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.16
[v0.1.15]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.15
[v0.1.14]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.14
[v0.1.13]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.13
[v0.1.12]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.12
[v0.1.11]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.11
[v0.1.10]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.10
[v0.1.9]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.9
[v0.1.8]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.8
[v0.1.7]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.7
[v0.1.5]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.5
[v0.1.0]: https://github.com/notfixingit3/waffle/releases/tag/v0.1.0
[v0.0.9]: https://github.com/notfixingit3/waffle/releases/tag/v0.0.9
[v0.0.8]: https://github.com/notfixingit3/waffle/releases/tag/v0.0.8
[v0.0.5]: https://github.com/notfixingit3/waffle/releases/tag/v0.0.5
[v0.0.1]: https://github.com/notfixingit3/waffle/releases/tag/v0.0.1
