# AGENTS.md

## Project Overview

Project Syrup — Instagram Waffle Spot Management Web App

A mobile-first live spot board and admin tracking system. Not a payment processor, not a randomizer, not Instagram automation.

## Repository State

**This project is fully built and operational.** All phases (1-7) are complete. Phase 8 (PWA support) is complete.

## Tech Stack

- Docker + Docker Compose
- Go backend (Gin) with server-rendered templates + Tailwind CSS
- PostgreSQL
- WebSockets for live updates

## Local Development

```bash
docker compose up --build
```

Services and ports:
- Application: http://localhost:8383
- Postgres: localhost:5432

**No .env file required for local development.** Docker Compose injects all needed environment variables.

## Target Structure

```
project-syrup/
  docker-compose.yml
  Dockerfile
  backend/
    go.mod
    cmd/api/main.go
    cmd/api/static/
      js/
      css/
      img/
      manifest.json
    internal/
      api/
      db/
      handlers/
      middleware/
      models/
      renderer/
      services/
      websocket/
    migrations/
    templates/
      layouts/
      pages/
        admin/
        public/
      partials/
```

## Implementation Phases

1. ✅ Docker Compose + Postgres + Go health check + server-rendered UI + migrations
2. ✅ Waffle schema + create endpoint + public page + static spot grid
3. ✅ Spot claims + pending status + admin dashboard + mark paid/release
4. ✅ WebSocket live updates + activity feed
5. ✅ Manual winner entry + winner/loser marking + buyer stats + history
6. ✅ Mobile polish + production Dockerfiles + deployment docs
7. ✅ Multi-admin auth + role-based access + password reset + admin management + archive/delete
8. ✅ PWA support + service worker + offline capabilities

## Critical Business Rules

**Spot states:** `available -> pending -> paid`, then `paid -> winner|loser`
- Admin can release pending spots back to available
- Paid spots should not be released without explicit admin handling

**Waffle completion:** All spots paid = filled. No timer, no scheduled end.

**Winner entry (manual only):**
1. Admin enters winning spot number after external drawing
2. Verify spot exists and is paid
3. Mark as winner, all other paid spots as loser
4. Set waffle status to completed
5. Update buyer stats

**Instagram handle validation:**
- Strip leading `@`, lowercase
- Reject empty
- Note: Client-side HTML maxlength="30" enforcement; server-side normalization via `NormalizeInstagramHandle()` strips `@` and lowercases only

**Spot claims:**
- Validate waffle is active
- Validate all requested spots exist and are available
- Claim transactionally — prevent race conditions

## API Conventions

Public JSON API:
- `GET /api/waffles` — List public waffles
- `GET /api/waffles/:slug` — Get waffle details + stats
- `GET /api/waffles/:slug/spots` — Get spot grid
- `GET /api/waffles/:slug/export` — Export spots as CSV
- `POST /api/claims` — Claim spots
- `GET /api/buyers/:handle/stats` — Buyer win/loss stats
- `GET /api/buyers/:handle/history` — Buyer claim history

Public pages:
- `GET /health` — Health check (JSON)
- `GET /` — Home page
- `GET /waffles` — Waffle list
- `GET /waffle/:slug` — Waffle detail + spot grid
- `GET /buyer/:handle` — Buyer stats page

Admin auth pages (no auth required):
- `GET /admin/login` — Login page
- `POST /admin/login` — Login (form POST with CSRF)
- `POST /admin/logout` — Logout (clears HttpOnly cookie)
- `GET /admin/forgot-password` — Password reset request page
- `POST /admin/forgot-password` — Submit reset request

Admin JSON API:
- `POST /api/admin/login` — JSON login (returns JWT)
- `POST /api/admin/forgot-password` — Request password reset
- `POST /api/admin/reset-password` — Reset password with token

Admin JSON API (auth required via HttpOnly cookie or Authorization header):
- `GET /api/admin/me` — Get current admin info
- `POST /api/admin/change-password` — Change password
- `POST /api/admin/waffles` — Create waffle
- `GET /api/admin/waffles?archived=true|false` — List waffles
- `PATCH /api/admin/waffles/:id` — Update waffle
- `POST /api/admin/waffles/:id/winner` — Set winner
- `POST /api/admin/waffles/:id/archive` — Archive waffle
- `POST /api/admin/waffles/:id/unarchive` — Unarchive waffle
- `DELETE /api/admin/waffles/:id` — Permanently delete waffle
- `POST /api/admin/spots/:id/pay` — Mark spot paid
- `POST /api/admin/spots/:id/release` — Release pending spot
- `GET /api/admin/reports/drought` — Drought list report
- `GET /api/admin/reports/power-buyers` — Power buyers report
- `GET /api/admin/reports/monthly-activity` — Monthly activity
- `GET /api/admin/reports/spot-velocity` — Spot velocity

Admin JSON API (super_admin only):
- `GET /api/admin/admins` — List all admins
- `POST /api/admin/admins` — Create new admin
- `PATCH /api/admin/admins/:id` — Update admin role
- `DELETE /api/admin/admins/:id` — Deactivate admin

Admin rendered pages (auth required):
- `GET /admin/dashboard` — Admin dashboard
- `GET /admin/waffles/:slug` — Manage individual waffle
- `GET /admin/waffles/new` — New waffle form
- `POST /admin/waffles/new` — Create waffle (form, CSRF-protected)
- `POST /admin/waffles/:id/archive` — Archive (form, CSRF-protected)
- `POST /admin/waffles/:id/unarchive` — Unarchive (form, CSRF-protected)
- `POST /admin/waffles/:id/delete` — Delete (form, CSRF-protected)
- `GET /admin/reports` — Reports page

Admin rendered pages (super_admin only):
- `GET /admin/admins` — Admin management page
- `POST /admin/admins` — Create admin (form, CSRF-protected)

## WebSocket

Route: `GET /ws/:slug` — Upgrade to WebSocket connection for a specific waffle

## WebSocket Events

Clients subscribe to `waffle:{slug}`. Broadcast on:
- spot claimed, released, marked paid
- waffle completed, winner entered
- activity event created

Example: `{"type": "SPOT_UPDATED", "payload": {"spot_number": 12, "status": "paid"}}`

## Design Notes

- Mobile-first, target <10 second claim flow
- Must work in Instagram's in-app browser
- Spot grid is the main feature
- Colors: available=green, pending=yellow, paid=red, winner=purple (bg-purple-500, #a855f7), loser=gray
- Winner spots use purple styling with 🏆 emoji overlay
- Feel: fast, simple, collectible-focused, community-driven
- Avoid: corporate, casino-themed, cluttered, overbuilt

## Security Requirements

- Admin auth via JWT stored in HttpOnly, Secure, SameSite=Strict cookie (`admin_token`)
- Bearer token in Authorization header also accepted for API clients
- CSRF protection on all form POST endpoints via cookie/form token with constant-time comparison
- Multi-admin auth with role-based access control (super_admin, admin roles)
- Rate limit public claim endpoint (implemented via `RateLimitClaims` middleware — token-bucket per IP with 5-min stale cleanup)
- Sanitize all user input (Instagram handles normalized via NormalizeInstagramHandle)
- Validate all state changes server-side
- Never trust frontend state
- Use DB transactions for multi-step changes
- Per-page cloned template renderers (renderer/) — base layouts + partials cloned per page

## Coding Guidelines

- Keep it simple and boring
- No microservices
- WebSocket server stays inside Go backend
- Avoid premature abstractions
- Use clear service boundaries
- Readable names over clever ones
- Comments only where logic is non-obvious

## Commit Message Guidelines (a.k.a. The Mystery Machine Rules)

> **Velma says:** Every commit needs a Scooby-Doo quote hidden in the message body — like finding a clue in a haunted mansion. It's our thing. Don't skip it.

**The Gang's Commit Checklist:**
- ✅ Explain what you changed and (more importantly) *why*
- ✅ Reference issue numbers if they exist
- ✅ Keep the technical details accurate — Fred gets mad when PRs don't match the description
- ✅ **Drop a Scooby quote somewhere in there** — make it feel like the gang wrote it, not a robot

**Approved Clues (pick one, don't overthink it):**
- "Ruh-roh!" — something went wrong / you fixed a bug
- "Zoinks!" — big surprise / major change / you can't believe this worked
- "Jinkies!" — you figured something out / clever solution
- "Would you do it for a Scooby Snack?" — tedious chore that had to be done
- "Puppy Power!" — new feature / something exciting
- "And I would have gotten away with it too, if it weren't for you meddling kids!" — refactoring / removing bad code
- "Scooby-Dooby-Doo!" — general purpose, we did it!
- "Like, no way, man!" — when you disagree with your past self
- "Creepers!" — something scary / edge case / security fix
- "Let's split up, gang!" — parallelizing work / decomposing a big task into smaller PRs
- "It was Old Man Jenkins all along!" — finally found the root cause of a sneaky bug
- "Time for a Scooby Snack!" — debugging marathons / documentation / taking a breather
- "That was close!" — narrowly avoided disaster (data loss, production bug, race condition)

> **Shaggy adds:** Don't just paste the quote at the bottom like an afterthought. Work it into the message like you're telling the gang about the case. Example: "Fixed race condition in spot claims — Zoinks! Almost missed that one!"

> **Scooby says:** *Raggy's right. Make it natural, not ro-ro-robotic.* 🐕

## Local Testing & Release Workflow

> **Daphne's Pre-Flight Checklist:** Before you bump that version string and push to origin, make sure the Mystery Machine actually *runs*, okay?

**When making changes that affect the application:**
1. Rebuild Docker containers locally: `docker compose up --build -d`
2. Verify the app boots and passes health check (http://localhost:8383/health)
3. Spot-check affected pages / endpoints
4. THEN bump version, update docs, and push

**Never do this:**
- ❌ Bump version → push → *then* test
- ❌ Commit README/CHANGELOG changes without running the build first
- ❌ Skip local verification because "it's just a docs change" — docs can lie, containers don't

> **Fred says:** *Measure twice, cut once. Test locally, tag after.* 🔧

## Future Todo / Roadmap

1. **Shareable "Buyer Cards" (Gamification):**
   - Public sharing layout optimized for Instagram Stories screenshotting (`/buyer/:handle`).
   - "Luck Rating" comparing actual wins vs mathematical probability.
   - Trophy case showing items won parsed from completed waffle titles.

2. **Multi-Tier Drops ("Minis & Mains"):**
   - Relational linkage between waffles (marking one as a "Mini" for a "Main").
   - Automatically claiming a designated reserved spot on the "Main" board for the winner of the "Mini" upon drawing/completion.

3. **Hyped Drop Protection ("Spot Queue & Ghosting Scores"):**
   - Allow waitlisting on already-claimed spots, auto-claiming them if the current pending timer expires.
   - Track ghosting rates (expired/unpaid spots vs claimed spots) per Instagram handle and warn admins.
