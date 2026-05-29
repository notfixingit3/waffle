# `backend/cmd/api/static/`

## OVERVIEW

Embedded static assets served at `/static` via `http.FS`. All files compiled into the Go binary.

## STRUCTURE

| File | Purpose |
|------|---------|
| `css/input.css` | Tailwind v4 + DaisyUI config with custom `syrup` theme (dark, amber primary) |
| `css/output.css` | Compiled Tailwind output — generated, do not hand-edit |
| `js/reports.js` | Client-side report rendering (drought, power buyers, monthly activity, spot velocity) |
| `js/admin-spot-actions.js` | Admin spot management (mark-paid modals, release confirmations, bulk select, select-all) |
| `js/spot-selection.js` | Public spot grid interactivity (claim selection, localStorage cache) |
| `js/spot-status-classes.js` | Single source of truth for spot status CSS class mappings |
| `js/websocket-client.js` | WebSocket connection manager with auto-reconnect |
| `js/sw.js` | Service worker for PWA offline caching |
| `js/offline-handler.js` | localStorage-based offline data cache |
| `manifest.json` | PWA manifest (standalone, amber theme) |
| `offline.html` | Offline fallback page |
| `img/` | App icons, logos, screenshots |
| `fonts/` | Inter variable font (woff2) |

## WHERE TO LOOK

| Task | File |
|------|------|
| Change spot status colors | `js/spot-status-classes.js` + `css/input.css` |
| Change Tailwind/DaisyUI theme | `css/input.css` |
| Add a new report tab | `js/reports.js` |
| Add bulk spot action UI | `js/admin-spot-actions.js` |
| Change WebSocket behavior | `js/websocket-client.js` |
| Update PWA cache version | `js/sw.js` (bump `CACHE_VERSION`) |
| Add new static asset | File here + update `static.go` embed directive |

## CONVENTIONS

- **Vanilla JavaScript only.** No build step for JS. IIFE modules with `var`, not `let`/`const`.
- **Tailwind v4 + DaisyUI v5.** Custom `syrup` theme defined in `input.css`.
- **Mobile-first touch targets.** 44px minimum. `touch-action: manipulation` globally.
- **Spot status colors:** `available=success(green)`, `pending=warning(yellow)`, `paid=error(red)`, `winner=secondary(purple)`, `loser=base-300(gray)`.
- **All new static files must be added to `static.go` embed directive.**

## ANTI-PATTERNS

- **Do NOT edit `css/output.css` directly.** Regenerate via Tailwind CLI from `input.css`.
- **Do NOT add JS frameworks or bundlers.** Keep it vanilla.
- **Do NOT forget to bump `CACHE_VERSION` in `sw.js`** when changing cached assets.
- **Do NOT use `let`/`const` in JS.** Stick to `var` for consistency with existing files.
- **Do NOT add assets without updating `static.go`.** Missing embeds = 404 in production.
