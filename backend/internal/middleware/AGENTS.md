# backend/internal/middleware/

## OVERVIEW
Gin middleware for auth, role gating, and rate limiting.

## STRUCTURE

| File | Purpose |
|------|---------|
| `auth.go` | `RequireAuth`, `RequireSuperAdmin`. JWT from cookie or Bearer header. Sets `admin_role` / `admin_id` on context. |
| `rate_limit.go` | Token-bucket rate limiter per IP (`sync.Map`). Cleanup goroutine every 3 min. Applied to claims only. |
| `rate_limit_test.go` | Burst, block, per-IP isolation, OPTIONS bypass, stale cleanup tests. |

## WHERE TO LOOK

| Task | File |
|------|------|
| Add new protected route | `auth.go` — middleware already applied in `main.go` |
| Adjust claim throttling | `rate_limit.go` — edit `rate.NewLimiter` burst/refill |
| Change auth failure behavior (JSON vs redirect) | `auth.go` — `isAPIRequest` controls response type |
| Add super_admin-only endpoint | `auth.go` — chain `RequireAuth` then `RequireSuperAdmin` |
| Debug rate limit hits | `rate_limit.go` — `Retry-After: 6` header + 429 JSON |

## CONVENTIONS

- **Dual auth delivery**: cookie (`admin_token`) preferred, fallback to `Authorization: Bearer <token>`.
- **API vs page detection**: `isAPIRequest` checks `Accept`, `Content-Type`, and `/api/` prefix. Determines JSON 401/403 vs redirect/plain text.
- **JWT secret**: read from `JWT_SECRET` env var at request time (no init-time caching).
- **Rate limiter state**: global `sync.Map` (`rateLimitClients`), not per-request. Stale entries purged after 5 min inactivity.
- **OPTIONS bypass**: `RateLimitClaims` skips limiting on `OPTIONS` to avoid preflight failures.

## ANTI-PATTERNS

- **Do NOT** store JWT secret at package init or in a global var. Read from env per validation.
- **Do NOT** use `c.GetString("admin_role")` before `RequireAuth` has run. Role is unset otherwise.
- **Do NOT** apply `RateLimitClaims` to non-claim endpoints without explicit discussion. It is scoped to claims by design.
- **Do NOT** mutate `rateLimitClients` directly outside `getRateLimitLimiter` or `cleanupStaleRateLimiters`. The map is shared across goroutines.
- **Do NOT** return HTML error pages from API routes. `isAPIRequest` must stay accurate.
