# App Map — Project Syrup: The Waffle Maker

A quick-reference guide to the codebase for humans and agents. Read this before digging into files.

---

## Repository Layout

```
waffle/
├── backend/                    Go application (module: github.com/syrup/backend)
│   ├── cmd/api/
│   │   ├── main.go             Entry point: router, middleware chain, route registration, graceful shutdown
│   │   ├── static.go           //go:embed directives for static assets and templates
│   │   └── static/
│   │       ├── css/            Tailwind + DaisyUI — input.css → output.css (build step)
│   │       ├── js/             Client-side JS modules (see JS Modules below)
│   │       ├── img/            Logo, payment icons, emoji assets, support logos
│   │       ├── fonts/          Inter TTF (also embedded separately in sharecardassets/)
│   │       └── cache/          share-cards/ — on-disk PNG cache (gitignored, created at runtime)
│   ├── internal/
│   │   ├── db/db.go            pgxpool connection, RunMigrations (iofs source)
│   │   ├── handlers/           HTTP handlers (one file per domain)
│   │   ├── middleware/         Gin middleware
│   │   ├── models/models.go    All struct definitions and JSON tags
│   │   ├── renderer/           Template renderer wrapping html/template
│   │   ├── services/           Business logic (one file per domain)
│   │   ├── sharecardassets/    //go:embed fonts + emoji PNGs for share card generation
│   │   └── websocket/hub.go    WebSocket hub: broadcast, register/unregister, ping/pong
│   ├── migrations/             SQL migration files + fs.go (//go:embed *.sql → migrations.FS)
│   └── templates/
│       ├── layouts/            base.html, admin_base.html
│       ├── partials/           header, footer, admin_nav, head
│       └── pages/
│           ├── admin/          Admin UI pages
│           └── public/         Public-facing pages
├── scripts/                    backup.sh, restore.sh, smoke-test-prod.sh
├── docs/screenshots/           UI screenshots referenced in README
├── docker-compose.yml          Local dev (builds from source, safe defaults)
├── docker-compose.prod.yml     Production (pulls from GHCR, no source mount)
├── Dockerfile                  Multi-stage: node (CSS build) → Go builder → minimal runtime
└── .github/workflows/
    ├── ci.yml                  go test + vet + govulncheck on every push/PR
    └── docker.yml              Tag-driven Docker build + GHCR push + GitHub Release
```

---

## Routes

### Public Pages

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/` | `HomePage` | Active waffles list |
| GET | `/waffles` | `WaffleListPage` | Full waffle list |
| GET | `/waffle/:slug` | `WaffleDetailPage` | Live spot grid via WebSocket |
| GET | `/waffle/:slug/card.png` | `WaffleShareCardPNG` | Generated PNG share card; disk-cached; rate limited |
| GET | `/buyer/:handle` | `BuyerStatsPage` | Buyer win/loss stats; rate limited |
| GET | `/buyer/:handle/card` | `BuyerCardPage` | Shareable buyer card (Instagram Story format); rate limited |
| GET | `/about` | `AboutPage` | Public about page with admin-only extras |
| GET | `/ws/:slug` | WebSocket upgrade | Real-time spot updates |

### Public API

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/waffles` | List active waffles |
| GET | `/api/waffles/:slug` | Waffle detail |
| GET | `/api/waffles/:slug/spots` | Spot grid |
| GET | `/api/waffles/:slug/export` | CSV export |
| POST | `/api/claims` | Claim spots (rate limited) |
| POST | `/api/claims/random` | Random spot claim (rate limited) |
| GET | `/api/buyers/:handle/stats` | Buyer stats JSON (rate limited) |
| GET | `/api/buyers/:handle/history` | Buyer waffle history JSON (rate limited) |
| GET | `/api/buyers/:handle/card` | Buyer card computed data JSON (rate limited) |
| GET | `/api/version` | Current app version string |

### Admin Pages (auth required)

| Path | Page |
|------|------|
| `/admin/login` | Login |
| `/admin/dashboard` | Dashboard with waffle cards |
| `/admin/waffles/new` | Create waffle |
| `/admin/waffles/:id` | Manage waffle (spots, winner, share) |
| `/admin/waffles/:id/edit` | Edit waffle |
| `/admin/reports` | Reports |
| `/admin/settings` | Settings + share templates tab |
| `/admin/payment-methods` | Payment method CRUD |
| `/admin/login-history` | My login history |
| `/admin/audit` | Audit log (admin/super_admin only) |
| `/admin/users` | Users registry |
| `/admin/admins` | Admin management (super_admin only) |

### Admin API (auth required, `/api/admin/...`)

Key groups — see `main.go` for full list:

- **Waffles** — CRUD, archive/unarchive/delete, winner management, share message/template, share card regeneration
- **Spots** — mark paid, release
- **Share Templates** — list, create, update, delete, set default (admin/waffle_manager)
- **Payment Methods** — list, create, update, deactivate
- **Admins** — list, create, update role, reset password, deactivate (super_admin)
- **Reports** — drought, power-buyers, monthly-activity, spot-velocity
- **Settings** — get/update all settings, WHOIS server (super_admin)
- **Auth** — login, forgot-password, reset-password, logout, change-password, me

---

## Services

| File | Responsibility |
|------|---------------|
| `waffle.go` | Waffle CRUD, stats, winner set/clear/change, share template assignment, `RenderWaffleShareMessage` |
| `buyers.go` | Buyer stats, waffle history, `ComputeBuyerCardData` (luck rating, trophies), activity events |
| `share_card.go` | `GenerateShareCard` (PNG via gg), disk cache invalidation, font temp file lifecycle |
| `share_card_assets.go` | Accessors for embedded Inter fonts and emoji PNGs |
| `share_templates.go` | Message template CRUD, default promotion, `RenderShareMessage` (token substitution) |
| `payment_methods.go` | Payment method CRUD, waffle ↔ method linking, `GetPaymentMethodsForWaffle` |
| `payment_url.go` | Build payment deep-link URLs per method type (Venmo/PayPal/CashApp/Zelle) |
| `admins.go` | Admin CRUD, role management, profile updates |
| `auth.go` | JWT generation/validation, token expiry |
| `password_policy.go` | Password strength validation, common-password rejection |
| `login_history.go` | Record logins, query history with role-based filtering |
| `login_throttle.go` | Failed-attempt counter, lockout enforcement |
| `audit_log.go` | Record/query audit events |
| `whois.go` | Async WHOIS IP enrichment for login records |
| `users.go` | users table, GetOrCreateUser, BackfillUsers |
| `reports.go` | Drought list, power buyers, monthly activity, spot velocity |
| `settings.go` | Key/value system settings store |
| `retention.go` | Configurable audit log + login history pruning |

---

## Middleware

| File | Middleware |
|------|-----------|
| `auth.go` | `RequireAuth`, `RequireRole`, `RequireSuperAdmin` — JWT cookie validation |
| `auth_rate_limit.go` | `AuthRateLimit` — login endpoint brute-force protection |
| `rate_limit.go` | `RateLimitClaims`, `RateLimitShareCard`, `RateLimitBuyer` — per-IP token-bucket limiters |
| `security_headers.go` | `SecurityHeaders` — CSP, X-Frame-Options, HSTS, etc. |
| `request_id.go` | `RequestID` — injects a UUID request ID into context and response header |
| `recovery.go` | `Recovery` — panic recovery with structured logging |

---

## Models (`internal/models/models.go`)

Key structs:

| Struct | Purpose |
|--------|---------|
| `Waffle` | Core waffle record; includes `WinningSpotNumbers []int` (multi-winner), `ShareTemplateID`, `ShareMessage` |
| `Spot` | Individual spot with status (`available`/`pending`/`paid`/`winner`) |
| `SpotStatus` | Named string type for spot status constants |
| `Admin` | Admin user with role |
| `PaymentMethod` | Stored payment method (Venmo/PayPal/CashApp/Zelle) |
| `MessageTemplate` | Share message template with `{token}` placeholders |
| `BuyerStats` | Aggregated win/loss/entry stats per handle |
| `BuyerWaffleHistory` | Per-waffle history entry; `IsWinner` from spot status |
| `BuyerCardData` | Computed: luck rating, trophies, history (in services/buyers.go) |
| `ActivityEvent` | Claim/pay/release/winner event log per waffle |
| `AuditLog` | Admin action audit record |

---

## Database Migrations

| # | Name | What it adds |
|---|------|-------------|
| 001 | initial_schema | waffles, spots, buyer_stats, activity_events |
| 002 | add_media_and_archive | media links, archived flag |
| 003 | add_admins | admins table, roles |
| 004 | seed_example_waffle | demo data |
| 005 | add_waffle_manager_role | waffle_manager role |
| 006 | add_admin_timezone | timezone preference |
| 007 | add_login_history_and_settings | login_history, system_settings |
| 008 | add_audit_log_and_login_ip | audit_log, IP on login |
| 009 | add_retention_settings | retention config keys |
| 010 | add_admin_profile_fields | first/last name, social links |
| 011 | create_users_table | users registry |
| 012 | add_multiple_items | item_count, winning_spot_numbers[], winning_instagram_handles[] JSONB |
| 013 | seed_multi_item_waffle | demo data for multi-item |
| 014 | add_payment_methods | payment_methods, waffle_payment_methods junction |
| 015 | add_share_templates | message_templates, waffles.share_template_id, waffles.share_message |

Migrations are embedded in the binary via `migrations.FS` (`//go:embed *.sql`). No filesystem mount needed.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | (dev default in compose) | PostgreSQL connection string |
| `JWT_SECRET` | Yes | (dev default in compose) | JWT signing key — change before production |
| `ADMIN_PASSWORD` | Yes | `syrup` | Default admin password on first boot |
| `APP_HOST` | Yes (prod) | `waffle.social` | Hostname used in persisted share message URLs |
| `PORT` | No | `8383` | HTTP listen port |
| `GIN_MODE` | No | debug | Set to `release` in production |
| `COOKIE_SECURE` | No | auto-detected | Force `true` to require HTTPS for cookies |
| `TRUSTED_PROXIES` | No | RFC1918 ranges | Comma-separated CIDRs for trusted reverse proxies |

---

## JS Modules (`backend/cmd/api/static/js/`)

| File | Purpose |
|------|---------|
| `spot-selection.js` | Public spot grid — single/random selection, claim submission, CSRF, WebSocket integration |
| `websocket-client.js` | WebSocket connect/reconnect with exponential backoff and heartbeat |
| `admin-spot-actions.js` | Admin spot grid — mark paid, release, set winner dropdown, clear winner |
| `share-message.js` | Admin share panel — template select, preview render, save, copy, card download/regenerate |
| `spot-status-classes.js` | Shared status → CSS class mapping used by both public and admin grids |
| `theme-toggle.js` | No-op stub; locks `data-theme="syrup"` (light/dark toggle removed) |
| `footer-clock.js` | Dual UTC + local clock in footer with waffle counter |
| `offline-handler.js` | Offline banner detection and toast notifications |
| `reports.js` | Admin reports page chart rendering |
| `sw.js` | Service worker — PWA offline caching and cache invalidation |

---

## Templates

**Layouts** — `base.html` (public), `admin_base.html` (admin)

**Partials** — `header.html`, `footer.html`, `admin_nav.html`, `head.html`

**Public pages** — `home.html`, `waffles.html`, `waffle_detail.html`, `buyer_stats.html`, `buyer_card.html`, `about.html`

**Admin pages** — `dashboard.html`, `waffle_new.html`, `waffle_manage.html`, `waffle_edit.html`, `payment_methods.html`, `settings.html`, `reports.html`, `audit_log.html`, `login_history.html`, `users.html`, `admins.html`, `login.html`, `reset_password.html`

Custom template functions registered in `renderer.go`: `deref` (dereference `*string`/`*int`), `mulf` (float multiply), `add`, `sub`, `mod`, standard Go time/string helpers.

---

## Share Card System

`GET /waffle/:slug/card.png?format=story|square`

1. Handler checks disk cache at `services.ShareCardCacheDir/{slug}-{format}.png`
2. Cache miss → `services.GenerateShareCard()` renders PNG via `fogleman/gg` + embedded Inter fonts
3. Result written to disk cache and served with `Cache-Control: public, max-age=3600`
4. Cache is invalidated by `InvalidateShareCardCache(slug)` when the waffle's share message or template changes
5. Admin can manually bust via `POST /api/admin/waffles/:id/share-message/regenerate-card`
6. Rate limited to 10 req/IP/min via `RateLimitShareCard`

Fonts are written to OS temp files once at startup (`sync.Once`) and cleaned up on graceful shutdown.

---

## CI/CD

**CI** (`.github/workflows/ci.yml`) — runs `go test ./...`, `go vet ./...`, `govulncheck` on every push and PR.

**Docker** (`.github/workflows/docker.yml`) — triggers on `v*` tags and branch pushes:

| Trigger | Tags published | Release type |
|---------|---------------|-------------|
| `v*.*.*` tag (stable) | `v0.x.x`, `0.x`, `latest` | GitHub Release (stable) |
| `v*.*.*-*` tag (pre-release) | `v0.x.x-rc.1` only | GitHub Pre-Release |
| `dev` branch push | `dev`, detected version tag | No release |
| `main` branch push | `main` | No release |

Platforms: `linux/amd64` + `linux/arm64`. Images pushed to `ghcr.io/notfixingit3/waffle`.
