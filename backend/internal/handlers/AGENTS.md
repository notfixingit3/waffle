# backend/internal/handlers/

## OVERVIEW

HTTP request handlers. Each file serves both rendered HTML pages and JSON API endpoints for a domain.

## STRUCTURE

| File | Lines | Purpose |
|------|-------|---------|
| `admin.go` | 556 | Admin pages (dashboard, waffle manage, new waffle, reports, admin mgmt) + JSON API (mark-paid, release-spot, set-winner, create-admin, reset-password) |
| `auth.go` | 262 | Auth pages (login, logout, forgot-password, reset-password) + auth API handlers + CSRF helpers |
| `public.go` | 122 | Public pages (home, waffle list, waffle detail, buyer stats) + buyer JSON API |
| `admin_test.go` | 54 | Tests `validateCreateAdminForm` |
| `public_test.go` | 131 | Tests template parsing + rendering for public pages |

## WHERE TO LOOK

| Task | File |
|------|------|
| Add a new admin page or admin API endpoint | `admin.go` |
| Change login/logout flow or CSRF logic | `auth.go` |
| Add a new public page | `public.go` |
| Fix WebSocket broadcast on spot state change | `admin.go` (only file that calls `ws.BroadcastSpotUpdate` / `ws.BroadcastWaffleCompleted`) |
| Add form validation for admin creation | `admin.go` + `admin_test.go` |
| Test template rendering | `public_test.go` |

## CONVENTIONS

- **Dual handlers in one file**: rendered page handlers (e.g., `AdminDashboard`) and JSON API handlers (e.g., `MarkSpotPaidAPI`) coexist. No separate `api/` subpackage.
- **Page handlers** use `renderers["template.html"].Render(c, ...)`. The `renderers` map is initialized via `InitRenderers()` in `public.go`.
- **API handlers** return `c.JSON()` with `gin.H{"error": ...}` or `gin.H{"message": ...}`.
- **Form POST handlers** call `validateCSRF(c)` first. On failure, re-render the form with an error message.
- **Error responses**: pages use `c.String(http.StatusNotFound, "...")` or re-render with errors; APIs use `c.JSON(http.StatusBadRequest, gin.H{"error": "..."})`.
- **ID parsing**: always `uuid.Parse(c.Param("id"))` with BadRequest on failure.
- **Redirects**: form success handlers redirect with `c.Redirect(http.StatusFound, "/admin/...")`; referer fallback used for archive/unarchive/delete.

## ANTI-PATTERNS

- **Do not call WebSocket broadcasts from new files**. Only `admin.go` should import `internal/websocket`. If another domain needs broadcasts, refactor to a service-level hook first.
- **Do not add business logic here**. Handlers validate input, call `services.*`, and format responses. No DB queries, no transactions.
- **Do not skip CSRF validation on form POSTs**. Every form handler must call `validateCSRF(c)` before reading `c.PostForm`.
- **Do not return raw errors to users in page handlers**. Re-render the form with a friendly message. APIs may return `err.Error()` in the JSON error field.
- **Do not create separate handler files for small API additions**. Add to the existing domain file (`admin.go`, `auth.go`, or `public.go`).
