# `backend/cmd/api/`

## OVERVIEW

Application entry point and composition root. Contains `main.go` (route registration + inline JSON API handlers), static asset embedding, and the `static/` directory.

## STRUCTURE

| File | Purpose |
|------|---------|
| `main.go` | Composition root: DB connect, migrations, WebSocket hub init, Gin setup, route registration, ~40 inline JSON API handler functions |
| `static.go` | `//go:embed` directive for static assets (CSS, JS, images, fonts, manifest, offline.html) |
| `static/` | Embedded static files served at `/static` via `http.FS` |

## WHERE TO LOOK

| Task | File |
|------|------|
| Add/modify a route | `main.go` |
| Add/modify a JSON API endpoint | `main.go` (handler functions are inline, not in `handlers/` package) |
| Add/modify a rendered page handler | `internal/handlers/` package |
| Add static assets (JS, CSS, images) | `static/` + update `static.go` embed directive |
| Change CORS/trusted proxies/middleware | `main.go` |
| Change template loading | `main.go` (per-page renderer cloning) |
| Change DB/migration setup | `main.go` |

## CONVENTIONS

- **JSON API handlers live inline in `main.go`**. Page handlers live in `internal/handlers/`.
- **Per-page template renderers**: base layout + partials cloned per page in `main.go`, then injected via `handlers.InitRenderers()`.
- **Static files embedded** via `embed.FS` and served with `r.StaticFS("/static", http.FS(staticSub))`.
- **CORS allows all origins** (`AllowOrigins: []string{"*"}`).
- **Trusted proxies** set to private Docker networks (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`).

## ANTI-PATTERNS

- **Do NOT extract JSON API handlers to `internal/handlers/`**. Keep them inline in `main.go`.
- **Do NOT add business logic in `main.go`**. Handlers should delegate to `internal/services/`.
- **Do NOT forget to update `static.go` embed directive** when adding new static files.
- **Do NOT use `gin.Default()`** (which includes Recovery + Logger). Use `gin.New()` and add middleware explicitly.
