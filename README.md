<p align="center">
  <img src="backend/cmd/api/static/img/logo.png" alt="Project Syrup logo" width="180" />
</p>

<h1 align="center">Project Syrup - The Waffle Maker</h1>

<p align="center">A dead-simple spot board for Instagram waffle drops. Built for collectors, by collectors.</p>

<p align="center">
  <a href="https://docs.docker.com/compose/"><img src="https://img.shields.io/badge/Docker-Compose-blue?logo=docker" alt="Docker" /></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.23-00ADD8?logo=go" alt="Go" /></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql" alt="PostgreSQL" /></a>
  <a href="https://github.com/notfixingit3/waffle/pkgs/container/waffle"><img src="https://img.shields.io/badge/ghcr.io-notfixingit3%2Fwaffle-blue?logo=docker&label=GHCR" alt="GHCR" /></a>
  <a href="https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API"><img src="https://img.shields.io/badge/WebSockets-Live-green?logo=socket.io" alt="WebSockets" /></a>
</p>

<p align="center"><strong>Live Demo:</strong> <a href="https://dev.waffle.projectsyrup.app">dev.waffle.projectsyrup.app</a> | <strong>Latest Release:</strong> <a href="https://github.com/notfixingit3/waffle/releases/tag/v0.1.23-beta.10">v0.1.23-beta.10</a></p>

---

## What is this?

Project Syrup is a **mobile-first live spot management system** for Instagram waffle sellers. No timers. No randomizers. No payment processing. Just a clean, real-time board where buyers claim spots and admins track payments.

Built to work inside Instagram's in-app browser because that's where your buyers already are.

### The Flow

1. **Admin creates a waffle** — sets title, price per spot, total spots, links to Instagram posts showing what's being waffled
2. **Buyers claim spots** — tap available spots, enter Instagram handle, done (< 10 seconds)
3. **Admin marks paid** — as payments come in via DM/CashApp/Venmo
4. **Winner drawn** — admin enters winning spot number after external drawing
5. **Everyone sees results** — instant WebSocket broadcast to all connected clients

---

## Screenshots

| Screenshot | Screenshot |
|------------|------------|
| **Public Home**<br>![Public home page showing active waffles](docs/screenshots/homepage.png) | **Admin Login**<br>![Admin login page](docs/screenshots/admin-login.png) |
| **Admin Dashboard**<br>![Admin dashboard with waffle cards](docs/screenshots/admin-dashboard.png) | **Waffle Management**<br>![Waffle management spot grid with mixed spot statuses](docs/screenshots/waffle-manage.png) |
| **Admin Management**<br>![Admin management table](docs/screenshots/admins-page.png) | **Reports**<br>![Admin reports page](docs/screenshots/reports-page.png) |
| **About Page**<br>![Public about page with admin extras](docs/screenshots/about-page.png) | |

---

## Features

**Waffle management**
- **Multi-item waffles** — A single drop can have multiple items and award multiple winners
- **Instagram media links** — Link to posts showing what's being waffled
- **Archive + delete controls** — Hide completed waffles by default, or type `DELETE` for permanent removal
- **Real-time spot grid** — WebSocket-powered claim, payment, release, and winner updates
- **Random spot claiming** — Buyers enter a count and the app picks available spots, with partial fulfillment if needed
- **Winner management** — Set, clear, and change winners; buyer stats auto-recalculate

**Payment**
- **Stored payment methods** — Admin-managed Venmo, PayPal, CashApp, and Zelle handles linked per waffle
- **Payment deep links** — Clickable pay buttons with correct app deep-link URLs per payment type

**Sharing**
- **Share card PNG** — Auto-generated 1080×1920 story or 1080×1080 square card at `/waffle/:slug/card.png`
- **Share message templates** — Admin-created hype-drop message templates with `{item}`, `{spots_left}`, `{total_spots}`, `{url}` tokens
- **Copy-ready share panel** — Template selector, editable preview, copy button, and card download on the manage page
- **Shareable buyer cards** — Public `/buyer/:handle/card` page with luck rating, trophy case, and OG meta tags

**Buyer stats**
- **Buyer stats page** — Win/loss history, luck rating, and trophy case per Instagram handle
- **Luck rating** — Statistical delta between actual win rate and expected win rate over completed waffles

**Admin**
- **Multi-admin auth** — Role-based access: `super_admin`, `admin`, `waffle_manager`
- **Admin management** — Create, deactivate, role-change, password-reset (super_admin only)
- **Admin audit log** — Audited state changes with filters and pagination
- **Login history** — Per-admin trail with async WHOIS IP enrichment (org, country, city, ASN)
- **Reports** — Drought list, power buyers, monthly activity, spot velocity
- **CSV exports** — Download a waffle's spot list for external reconciliation

**Platform**
- **Mobile-first public flow** — Built for Instagram's in-app browser; sub-10-second claim
- **Installable app shell** — PWA with Web App Manifest, service worker, offline caching
- **Warm dark theme** — Aligned with projectsyrup.app (amber accent, glass header)
- **Dual clock footer** — Server UTC time + local browser time with waffle counter
- **Security hardening** — Secure cookies, login lockout, password policy, CSRF, rate limiting, destructive-action confirmation

---

## Quick Start

**Prerequisites:** Docker and Docker Compose.

```bash
# Clone the repo
git clone https://github.com/notfixingit3/waffle.git
cd waffle

# Start everything
docker compose up --build
```

Docker Compose starts both the Go app and PostgreSQL, runs database migrations, and injects safe local-development defaults for the database connection, JWT secret, and admin credentials. You do not need to create a `.env` file to run the app locally.

After startup, open the app and admin tools here:

| Service | URL |
|---------|-----|
| Application | http://localhost:8383 |
| Admin Login | http://localhost:8383/admin/login |
| PostgreSQL | localhost:5432 |

Default local admin credentials are `admin` / `syrup`. Change them before any real deployment.

---

## Production Deployment

**Prerequisites:** Docker and Docker Compose v2+.

1. Copy [`docker-compose.prod.yml`](docker-compose.prod.yml) to your server
2. Create a `.env` file (see [`.env.example`](.env.example) for reference):
   ```bash
   WAFFLE_VERSION=v0.1.23-beta.10
   DATABASE_URL=postgres://user:password@postgres:5432/syrup?sslmode=disable
   JWT_SECRET=your-secure-random-secret-here
   ADMIN_PASSWORD=your-secure-admin-password
   APP_HOST=yourdomain.com
   ```
4. Start the services:
   ```bash
   docker compose -f docker-compose.prod.yml up -d
   ```
5. Open `/admin/login` and change the default admin password immediately

Pre-built images are available at [`ghcr.io/notfixingit3/waffle`](https://github.com/notfixingit3/waffle/pkgs/container/waffle) for `linux/amd64` and `linux/arm64`.

### Upgrading to v0.1.19+

Beginning in `v0.1.19`, database migrations are embedded directly in the Go application binary.

If you are upgrading an existing deployment from a version prior to `v0.1.19`:
1. **Update `docker-compose.prod.yml`:** The volume mount for migrations (`- ./backend/migrations:/app/migrations:ro`) has been removed and is no longer needed. You can safely delete it from your local compose file.
2. **Clean up files (Optional):** You no longer need to copy the `backend/migrations/` directory to your server. Any existing `migrations/` directory on your server can be safely deleted.
3. **Pull and redeploy:**
   ```bash
   docker compose -f docker-compose.prod.yml pull
   docker compose -f docker-compose.prod.yml up -d
   ```

## Release Channels

The following channels are available for the Docker image:

| Channel | Tag | Description |
|---------|-----|-------------|
| Stable | `latest` | Tracks the latest stable release from the `main` branch. Recommended for production. |
| Dev | `dev` | Tracks the latest build from the `dev` branch. For testing and staging only. May be unstable. || Pinned | `v0.1.22` | A specific stable version. Recommended for reproducible deployments. |

The stable channel is currently in production testing. Pin to specific versions for critical deployments.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Go server-side templates + Tailwind CSS + DaisyUI |
| Backend | Go (Gin), WebSocket hub |
| Database | PostgreSQL with migrations |
| DevOps | Docker Compose, multi-stage builds |
| PWA | Web App Manifest, app icons, standalone display metadata |

---

## Project Status

| Phase | Status | Description |
|-------|--------|-------------|
| 1 | ✅ Complete | Docker Compose + Postgres + Go health check + server-rendered UI |
| 2 | ✅ Complete | Waffle schema + create endpoint + public page + spot grid + Instagram media links |
| 3 | ✅ Complete | Spot claims + pending status + admin dashboard + mark paid/release + archive + delete |
| 4 | ✅ Complete | WebSocket live updates + activity feed + buyer stats + admin reporting |
| 5 | ✅ Complete | Manual winner entry + winner/loser marking + buyer stats + history |
| 6 | ✅ Complete | Mobile polish + production Dockerfiles + deployment docs |
| 7 | ✅ Complete | Multi-admin auth + role-based access + password reset + admin management + archive/delete |
| 8 | ✅ Complete | Offline/service worker support with installable app shell, offline page caching, and update notifications |
| 9 | ✅ Complete | DaisyUI migration (Halloween/syrup theme + amber primary) |
| 10 | ✅ Complete | Production hardening (structured logging, health probes, graceful shutdown, rate limiting, Docker hardening) |
| 11 | ✅ Complete | Admin audit/security polish (audit log, password policy, lockout, destructive confirmations) |
| 12 | ✅ Complete | Bugfix/polish release (archived filters, buyer stats recalculation, password reset response, accessibility polish) |

---

## Architecture

```
┌──────────────────┐     ┌─────────────┐
│  Go (Gin)        │────▶│ PostgreSQL  │
│  Server Templates │◄────│   (pgx)     │
│  + WebSocket Hub │     └─────────────┘
└──────────────────┘
        │
        ▼
  Tailwind CSS
  (server-rendered)
```

**Design principles:**
- Keep it simple and boring
- No microservices
- WebSocket server stays inside Go backend
- Avoid premature abstractions
- Readable names over clever ones

---

## API Overview

**Public Endpoints**
- `GET /api/waffles` — List public waffles
- `GET /api/waffles/:slug` — Waffle details
- `GET /api/waffles/:slug/spots` — Spot grid
- `GET /api/waffles/:slug/export` — Export spots as CSV
- `POST /api/claims` — Claim specific spots
- `POST /api/claims/random` — Claim a random count of available spots
- `GET /api/buyers/:handle/stats` — Buyer win/loss stats
- `GET /api/buyers/:handle/history` — Buyer waffle history
- `GET /api/buyers/:handle/card` — Buyer card computed data (luck rating, trophies)
- `GET /api/version` — Current app version

**Public Pages**
- `GET /` — Home page
- `GET /waffles` — Waffle list
- `GET /waffle/:slug` — Waffle detail + live spot grid
- `GET /waffle/:slug/card.png?format=story|square` — Share card PNG
- `GET /buyer/:handle` — Buyer stats page
- `GET /buyer/:handle/card` — Shareable buyer card (Instagram Story layout)

**Admin Auth Endpoints**
- `POST /api/admin/login` — Username/password login
- `POST /api/admin/forgot-password` — Request password reset
- `POST /api/admin/reset-password` — Reset password with token

**Admin Endpoints** (auth required)
- `GET /api/admin/me` — Current admin info
- `PATCH /api/admin/me/timezone` — Update timezone preference
- `POST /api/admin/change-password` — Change password
- `GET /api/admin/waffles?archived=true|false` — List waffles
- `POST /api/admin/waffles` — Create waffle
- `PATCH /api/admin/waffles/:id` — Update waffle
- `POST /api/admin/waffles/:id/archive` — Archive waffle
- `POST /api/admin/waffles/:id/unarchive` — Unarchive waffle
- `DELETE /api/admin/waffles/:id` — Permanently delete waffle
- `POST /api/admin/waffles/:id/winner` — Set winner
- `POST /api/admin/waffles/:id/clear-winner` — Clear winner
- `POST /api/admin/waffles/:id/change-winner` — Change winner
- `GET /api/admin/waffles/:id/share-message` — Get share message + template list
- `PATCH /api/admin/waffles/:id/share-message` — Update share template/message
- `POST /api/admin/waffles/:id/share-message/render` — Preview rendered message
- `POST /api/admin/waffles/:id/share-message/regenerate-card` — Bust share card cache
- `POST /api/admin/spots/:id/pay` — Mark spot paid
- `POST /api/admin/spots/:id/release` — Release pending spot
- `GET /api/admin/payment-methods` — List active payment methods
- `POST /api/admin/payment-methods` — Create payment method
- `PATCH /api/admin/payment-methods/:id` — Update payment method
- `DELETE /api/admin/payment-methods/:id` — Deactivate payment method
- `GET /api/admin/share-templates` — List message templates
- `POST /api/admin/share-templates` — Create message template
- `PATCH /api/admin/share-templates/:id` — Update message template
- `DELETE /api/admin/share-templates/:id` — Delete message template
- `POST /api/admin/share-templates/:id/default` — Set default template
- `GET /api/admin/admins` — List all admins (super_admin only)
- `POST /api/admin/admins` — Create admin (super_admin only)
- `PATCH /api/admin/admins/:id` — Update admin role (super_admin only)
- `PATCH /api/admin/admins/:id/password` — Reset admin password (super_admin only)
- `DELETE /api/admin/admins/:id` — Deactivate admin (super_admin only)
- `GET /api/admin/reports/drought` — Drought list
- `GET /api/admin/reports/power-buyers` — Power buyers
- `GET /api/admin/reports/monthly-activity` — Monthly activity
- `GET /api/admin/reports/spot-velocity` — Spot velocity

---

## Contributing

This is a personal project, but issues and PRs are welcome. The codebase prioritizes:

1. **Correctness** — Server-side validation for every state change
2. **Performance** — Sub-10-second claim flow on mobile
3. **Simplicity** — No over-engineering, clear service boundaries

### Community Contributors

- **OrangeSoJuicy** ([Instagram](https://www.instagram.com/mxkxng/)) — UI feature idea for random spot claiming / "Pick Random Spots" ([feature commit](https://github.com/notfixingit3/waffle/commit/cd9edec20f0991ed59158886282b51ac55d83786)).

---

## Special Thanks

Project Syrup exists because two glass artists kept running great waffles the hard way.

<table>
  <tr>
    <td align="center" width="50%">
      <a href="https://www.instagram.com/dani_boo_glass/">
        <img src="backend/cmd/api/static/img/dani.png" alt="Dani Boo Glass Logo" width="100" />
      </a>
      <h3>Dani Boo Glass</h3>
      <a href="https://www.instagram.com/dani_boo_glass/">
        <img src="https://img.shields.io/badge/Instagram-dani__boo__glass-E4405F?style=for-the-badge&logo=instagram&logoColor=white" alt="Dani Boo Glass on Instagram" />
      </a>
    </td>
    <td align="center" width="50%">
      <a href="https://www.instagram.com/crysis_designs/">
        <img src="backend/cmd/api/static/img/crysis.png" alt="Crysis Designs Logo" width="100" />
      </a>
      <h3>Crysis Designs</h3>
      <a href="https://www.instagram.com/crysis_designs/">
        <img src="https://img.shields.io/badge/Instagram-crysis__designs-E4405F?style=for-the-badge&logo=instagram&logoColor=white" alt="Crysis Designs on Instagram" />
      </a>
    </td>
  </tr>
</table>

Special shout out to [Dani Boo Glass](https://www.instagram.com/dani_boo_glass/) and [Crysis Designs](https://www.instagram.com/crysis_designs/) for creating the original Waffle and for driving me nuts watching them copy/paste spot lists over and over again in chat.

---

## Support

Project Syrup is built out of passion for the glass art community. If this app helps you run smoother waffles or more exciting races, here is how you can support the project:

<table>
  <tr>
    <td align="center" width="33%">
      <h3>🎨 Sponsor a Glass Piece</h3>
      <p>Sponsor Tom's next <strong>Wubble</strong>, <strong>Jelli</strong>, or <strong>Pocket Monstor</strong>.</p>
      <a href="https://www.instagram.com/crysis_designs/">
        <img src="https://img.shields.io/badge/Instagram-PM%20Crysis%20Designs-E4405F?style=for-the-badge&logo=instagram&logoColor=white" alt="PM Crysis Designs on Instagram" />
      </a>
    </td>
    <td align="center" width="33%">
      <h3>💖 GitHub Sponsors</h3>
      <p>Support development directly through GitHub Sponsors program.</p>
      <a href="https://github.com/sponsors/notfixingit3">
        <img src="https://img.shields.io/badge/Sponsor-GitHub%20Sponsors-EA4AAA?style=for-the-badge&logo=github-sponsors&logoColor=white" alt="Sponsor on GitHub" />
      </a>
    </td>
    <td align="center" width="33%">
      <h3>☕ Support Development</h3>
      <p>Help cover hosting costs and directly support the development of this application.</p>
      <a href="https://www.buymeacoffee.com/notfixingit">
        <img src="https://img.shields.io/badge/Buy%20Me%20A%20Coffee-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black" alt="Buy Me A Coffee" />
      </a>
    </td>
  </tr>
</table>

---

## Third-Party Libraries & Licenses

Project Syrup utilizes several excellent open-source third-party libraries:

### Backend (Go)
- **[Gin](https://github.com/gin-gonic/gin)** — MIT License
- **[Gin CORS middleware](https://github.com/gin-contrib/cors)** — MIT License
- **[Golang JWT](https://github.com/golang-jwt/jwt)** — MIT License
- **[Golang Migrate](https://github.com/golang-migrate/migrate)** — MIT License
- **[Google UUID](https://github.com/google/uuid)** — BSD 3-Clause License
- **[Gorilla WebSocket](https://github.com/gorilla/websocket)** — BSD 2-Clause License
- **[pgx (PostgreSQL Driver)](https://github.com/jackc/pgx)** — MIT License
- **[Go Crypto Subrepository](https://golang.org/x/crypto)** — BSD 3-Clause License
- **[Go Rate Limit Subrepository](https://golang.org/x/time)** — BSD 3-Clause License

### Frontend & Styling
- **[Tailwind CSS](https://tailwindcss.com/)** — MIT License
- **[DaisyUI](https://daisyui.com/)** — MIT License
- **[Inter Font](https://rsms.me/inter/)** — SIL Open Font License 1.1

### Graphics
- **[Twemoji](https://github.com/twitter/twemoji)** — CC-BY 4.0 License (trophy emoji used in share card assets)

---

## License

MIT — do whatever you want, just don't blame me when your waffle fills up in 30 seconds.

---

*Built with 🧇 and questionable sleep habits.*
