# backend/internal/middleware/

## OVERVIEW

Gin middleware for auth, role gating, rate limiting, security hardening, and observability.

## STRUCTURE

| File | Purpose |
|------|---------|
| `auth.go` | `RequireAuth`, `RequireSuperAdmin`. JWT from cookie or Bearer header. Sets `admin_role` / `admin_id` on context. |
| `auth_test.go` | Tests for auth middleware: valid/invalid tokens, expired, missing auth, role checks, API vs page responses. |
| `auth_rate_limit.go` | Per-admin rate limiter for auth endpoints (login, password reset). Token-bucket via `sync.Map` with 3-min cleanup. |
| `auth_rate_limit_test.go` | Tests for auth rate limiter: burst, block, per-admin isolation, cleanup. |
| `rate_limit.go` | Token-bucket rate limiter per IP (`sync.Map`). Cleanup goroutine every 3 min. Applied to claims only. |
| `rate_limit_test.go` | Burst, block, per-IP isolation, OPTIONS bypass, stale cleanup tests. |
| `recovery.go` | Custom panic recovery middleware that logs structured error details and returns 500 JSON/HTML. |
| `recovery_test.go` | Tests for recovery middleware: panic capture, CORS header preservation, response format. |
| `request_id.go` | Injects `X-Request-ID` header (from incoming or generated UUID) into context and response. |
| `request_id_test.go` | Tests for request ID propagation: incoming passthrough, generation, context access. |
| `security_headers.go` | Sets security headers: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, `Content-Security-Policy`. |
| `security_headers_test.go` | Tests for security headers: presence, values, override behavior. |

## WHERE TO LOOK

| Task | File |
|------|------|
| Add new protected route | `auth.go` — middleware already applied in `main.go` |
| Adjust claim throttling | `rate_limit.go` — edit `rate.NewLimiter` burst/refill |
| Adjust login throttling | `auth_rate_limit.go` — separate limiter for auth endpoints |
| Change auth failure behavior (JSON vs redirect) | `auth.go` — `isAPIRequest` controls response type |
| Add super_admin-only endpoint | `auth.go` — chain `RequireAuth` then `RequireSuperAdmin` |
| Debug rate limit hits | `rate_limit.go` — `Retry-After` header + 429 JSON |
| Add/modify security headers | `security_headers.go` — `SecurityHeaders()` handler |
| Add request tracing | `request_id.go` — ID set on `c.Request.Context()` |
| Change panic recovery behavior | `recovery.go` — `RecoveryMiddleware()` handler |

## CONVENTIONS

- **Dual auth delivery**: cookie (`admin_token`) preferred, fallback to `Authorization: Bearer <token>`.
- **API vs page detection**: `isAPIRequest` checks `Accept`, `Content-Type`, and `/api/` prefix. Determines JSON 401/403 vs redirect/plain text.
- **JWT secret**: read from `JWT_SECRET` env var at request time (no init-time caching).
- **Separate rate limiters**: `RateLimitClaims` for claims (by IP), `AuthRateLimit` for auth endpoints (by admin ID/IP).
- **Rate limiter state**: global `sync.Map`, stale entries purged after 5 min (claims) or 3 min (auth) inactivity.
- **OPTIONS bypass**: `RateLimitClaims` skips limiting on `OPTIONS` to avoid preflight failures.
- **Request ID propagation**: extracted from incoming `X-Request-ID` header or auto-generated UUIDv4; accessible via `GetRequestID(ctx)`.
- **Security headers**: applied globally via Gin's `Use()` — `nosniff`, `deny` framing, strict referrer, no permission delegation, permissive CSP (dev-friendly).

## ANTI-PATTERNS

- **Do NOT** store JWT secret at package init or in a global var. Read from env per validation.
- **Do NOT** use `c.GetString("admin_role")` before `RequireAuth` has run. Role is unset otherwise.
- **Do NOT** apply `RateLimitClaims` to non-claim endpoints without explicit discussion. It is scoped to claims by design.
- **Do NOT** mutate rate limiter maps directly outside the dedicated get/cleanup functions. Maps are shared across goroutines.
- **Do NOT** return HTML error pages from API routes. `isAPIRequest` must stay accurate.
- **Do NOT** bypass the auth rate limiter for login/password-reset endpoints. Brute-force protection is essential.
- **Do NOT** set `Content-Security-Policy` too restrictively in development — it must allow inline styles (Tailwind).
