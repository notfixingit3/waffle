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

<p align="center"><strong>Live Demo:</strong> [Coming Soon] | <strong>Latest Release:</strong> <a href="https://github.com/notfixingit3/waffle/releases/tag/v0.1.21">v0.1.21</a></p>

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

- **Multi-admin auth** — Role-based access control with `super_admin`, `admin`, and `waffle_manager` roles
- **Admin management** — Create admins, change roles, deactivate accounts, and reset another admin's password (super_admin only)
- **waffle_manager role** — Create and manage waffles + view reports, without archive/delete/user-management access
- **Timezone settings** — Per-admin timezone preference with IANA timezone dropdown in settings page
- **Password reset** — Self-service reset tokens plus authenticated password changes
- **Instagram media links** — Link to posts showing what's being waffled (supports multiple items)
- **Archive + delete controls** — Hide completed waffles by default, or type `DELETE` for permanent removal
- **Real-time spot grid** — WebSocket-powered claim, payment, release, and winner updates
- **Random spot claiming** — Buyers can enter a count and let the app pick available spots for them while preserving the normal pending/payment workflow
- **Mobile-first public flow** — Built for fast spot claims inside Instagram's in-app browser
- **Installable app shell** — Web App Manifest, app icons, and standalone display metadata are wired in
- **Admin dashboard** — Create waffles, manage spots, track payments, enter winners
- **Admin reports** — Drought list, power buyers, monthly activity, and spot velocity reports
- **Buyer stats** — Track win/loss history per Instagram handle
- **Activity history** — Record claim, payment, release, and winner events per waffle
- **CSV exports** — Download a waffle's spot list for external reconciliation
- **Transactional safety** — No double-claims, ever
- **Light/dark mode** — Manual theme toggle with persisted preference
- **Dual clock footer** — Server UTC time + local browser time with waffle counter
- **Winner management** — Admin-only clear/change winner with buyer stats recalculation
- **Login history** — Audit trail with async WHOIS IP enrichment (org, country, city, ASN)
- **Settings dropdown** — Consolidated admin menu under username (Settings, History, About, Theme, Logout)
- **About page** — Public about page with admin-only system extras
- **Configurable WHOIS** — Super_admin can configure WHOIS server for IP lookups
- **Admin audit log** — Admin and super_admin users can review audited state changes with filters and pagination
- **Security hardening** — Structured logging, health/readiness probes, secure cookies, login lockout, password policy, and destructive action password confirmation

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
   WAFFLE_VERSION=v0.1.21
   DATABASE_URL=postgres://user:password@postgres:5432/syrup?sslmode=disable
   JWT_SECRET=your-secure-random-secret-here
   ADMIN_PASSWORD=your-secure-admin-password
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
| Dev | `dev` | Tracks the latest build from the `dev` branch. For testing and staging only. May be unstable. |
| Pinned | `v0.1.21` | A specific stable version. Recommended for reproducible deployments. |

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
- `GET /api/waffles/:slug` — Get waffle details
- `GET /api/waffles/:slug/spots` — Get spot grid
- `GET /api/waffles/:slug/export` — Export spots as CSV
- `POST /api/claims` — Claim spots
- `POST /api/claims/random` — Claim a requested count of random available spots
- `GET /api/buyers/:handle/stats` — Buyer win/loss stats
- `GET /api/buyers/:handle/history` — Buyer claim history

**Public Pages**
- `GET /` — Home page
- `GET /waffles` — Waffle list
- `GET /waffle/:slug` — Waffle detail + live spot grid
- `GET /buyer/:handle` — Buyer stats page

**Admin Auth Endpoints**
- `POST /api/admin/login` — Username/password login
- `POST /api/admin/forgot-password` — Request password reset
- `POST /api/admin/reset-password` — Reset password with token

**Admin Endpoints** (auth required)
- `GET /api/admin/me` — Get current admin info
- `PATCH /api/admin/me/timezone` — Update timezone preference
- `POST /api/admin/change-password` — Change password
- `GET /api/admin/waffles?archived=true|false` — List waffles
- `POST /api/admin/waffles` — Create waffle
- `PATCH /api/admin/waffles/:id` — Update waffle
- `POST /api/admin/waffles/:id/archive` — Archive waffle
- `POST /api/admin/waffles/:id/unarchive` — Unarchive waffle
- `DELETE /api/admin/waffles/:id` — Permanently delete waffle
- `POST /api/admin/spots/:id/pay` — Mark spot paid
- `POST /api/admin/spots/:id/release` — Release pending spot
- `POST /api/admin/waffles/:id/winner` — Enter winner
- `GET /api/admin/admins` — List all admins (super_admin only)
- `POST /api/admin/admins` — Create new admin (super_admin only)
- `PATCH /api/admin/admins/:id` — Update admin role (super_admin only)
- `PATCH /api/admin/admins/:id/password` — Reset another admin's password (super_admin only)
- `DELETE /api/admin/admins/:id` — Deactivate admin (super_admin only)
- `GET /api/admin/reports/drought` — Drought list report
- `GET /api/admin/reports/power-buyers` — Power buyers report
- `GET /api/admin/reports/monthly-activity` — Monthly activity report
- `GET /api/admin/reports/spot-velocity` — Spot velocity report

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
    <td align="center" width="50%">
      <h3>🎨 Sponsor a Glass Piece</h3>
      <p>Sponsor Tom's next <strong>Wubble</strong>, <strong>Jelli</strong>, or <strong>Pocket Monstor</strong>.</p>
      <a href="https://www.instagram.com/crysis_designs/">
        <img src="https://img.shields.io/badge/Instagram-PM%20Crysis%20Designs-E4405F?style=for-the-badge&logo=instagram&logoColor=white" alt="PM Crysis Designs on Instagram" />
      </a>
    </td>
    <td align="center" width="50%">
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

---

## License

MIT — do whatever you want, just don't blame me when your waffle fills up in 30 seconds.

---

*Built with 🧇 and questionable sleep habits.*
