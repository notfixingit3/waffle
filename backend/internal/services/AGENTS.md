# backend/internal/services/

## OVERVIEW

Core business logic for Project Syrup. All service functions access `db.Pool` directly as a global variable. No dependency injection, no ORM, no query builder.

## STRUCTURE

| File | Lines | Purpose |
|------|-------|---------|
| `waffle.go` | 591 | Waffle CRUD, spot lifecycle (claim, pay, release, winner), buyer stats updates, slug generation, Instagram handle normalization |
| `waffle_test.go` | 359 | Tests for waffle CRUD, spot lifecycle, buyer stats, slug generation |
| `admins.go` | 393 | Admin CRUD, authentication, password reset tokens, login recording, bcrypt hashing, constant-time comparison, login history, destructive action password confirmation |
| `buyers.go` | 164 | Buyer stats lookup, per-buyer waffle history with spot aggregation, activity event recording/retrieval |
| `reports.go` | 229 | Drought list, power buyers, monthly activity, payment lag, spot velocity (large SQL CTEs and window functions) |
| `auth.go` | 42 | JWT secret loading from env, generic token generation, password hash/check helpers |
| `login_throttle.go` | 124 | Per-admin login attempt tracking with configurable threshold and lockout duration |
| `login_throttle_test.go` | 294 | Tests for login throttling: threshold, lockout, reset, concurrent safety |
| `login_history.go` | 352 | Login history recording, retrieval with pagination, WHOIS enrichment integration |
| `login_history_test.go` | 704 | Tests for login history: recording, pagination, enrichment, filtering |
| `password_policy.go` | 58 | Password strength validation: minimum length, complexity requirements, common password rejection |
| `password_policy_test.go` | 33 | Tests for password policy: valid/invalid passwords, edge cases |
| `retention.go` | 52 | Data retention policy enforcement: scheduled cleanup of old login history, audit logs, activity events |
| `settings.go` | 115 | Admin settings management: timezone preference, WHOIS server configuration |
| `settings_test.go` | 241 | Tests for settings: get/update timezone, WHOIS config, validation |
| `whois.go` | 143 | WHOIS IP lookup client: configurable server, response parsing, caching |
| `whois_test.go` | 424 | Tests for WHOIS: lookup, parsing, caching, server config |
| `audit_log.go` | 127 | Audit log recording and retrieval with filters (action, admin, target type, time range) and pagination |
| `users.go` | 142 | User registry: idempotent upsert (`GetOrCreateUser`), paginated search (`ListUsers`), handle backfill from spots (`BackfillUsers`) |
| `users_test.go` | 239 | Tests for user creation, idempotent upsert, handle normalization, empty handle rejection, search, pagination |

## WHERE TO LOOK

| Task | File |
|------|------|
| Create waffle + generate spots transactionally | `waffle.go` — `CreateWaffle` |
| Claim spots with race-condition protection | `waffle.go` — `ClaimSpots` (uses `FOR UPDATE`) |
| Mark winner, update buyer stats, complete waffle | `waffle.go` — `SetWinner` |
| Admin login, password reset flow | `admins.go` — `AuthenticateAdmin`, `CreatePasswordResetToken`, `ValidatePasswordResetToken` |
| Login attempt throttling | `login_throttle.go` — `CheckLoginThrottle`, `RecordLoginAttempt` |
| Login history with WHOIS enrichment | `login_history.go` — `RecordLoginHistory`, `GetLoginHistory` |
| Buyer win/loss history per Instagram handle | `buyers.go` — `GetBuyerWaffleHistory` |
| Activity feed for a waffle | `buyers.go` — `GetActivityEvents`, `RecordActivityEvent` |
| Drought/power buyer/velocity reports | `reports.go` — `GetDroughtList`, `GetPowerBuyers`, `GetSpotVelocity` |
| Password strength validation | `password_policy.go` — `ValidatePassword` |
| Data retention / cleanup | `retention.go` — `CleanupOldRecords` |
| Admin settings (timezone, WHOIS) | `settings.go` — `GetAdminSettings`, `UpdateTimezone` |
| WHOIS IP enrichment | `whois.go` — `WhoisLookup` |
| Audit log queries | `audit_log.go` — `RecordAuditLog`, `GetAuditLogs` |
| User registry (auto-create on claim, backfill, search) | `users.go` — `GetOrCreateUser`, `ListUsers`, `BackfillUsers` |
| JWT secret fallback for local dev | `auth.go` — `GetJWTSecret` |

## CONVENTIONS

- **Global db access:** `db.Pool.QueryRow`, `db.Pool.Exec`, `db.Pool.Begin` — no interfaces, no constructors
- **Raw SQL only:** Positional parameters `$1`, `$2`, ... — never string interpolation
- **Transactions:** `Begin` → `defer tx.Rollback` → `Commit` for multi-step ops (CreateWaffle, ClaimSpots, SetWinner, DeleteWaffle)
- **Error wrapping:** `fmt.Errorf("context: %w", err)` on every boundary
- **Instagram handle normalization:** `NormalizeInstagramHandle` strips `@` and lowercases; called inside `ClaimSpots`
- **Buyer stats:** `UpdateBuyerStats` uses `ON CONFLICT` upsert; called from `SetWinner` after commit
- **Password tokens:** Stored as bcrypt hashes; plaintext tokens returned to caller but never persisted
- **Login throttling:** Configurable threshold (default 5 attempts) and lockout duration (default 15 min); per-admin tracking
- **WHOIS caching:** Results cached in-memory with configurable TTL to avoid redundant lookups
- **Audit logging:** Structured action logging with admin, target type, target ID, and metadata JSON
- **Data retention:** Scheduled cleanup goroutine for old records (login history, audit logs, activity events)

## ANTI-PATTERNS

- **No ORM or query builder** — raw SQL only
- **No dependency injection** — do not pass `*pgxpool.Pool` as a parameter or struct field
- **No service structs** — package-level functions only
- **No business logic in SQL** — keep calculations in Go where possible (reports are the exception)
- **No naked returns** — always name return values explicitly
- **No `sql.Null*` types** — scan into pointer fields directly (`*string`, `*time.Time`)
- **No logging in transactions** — log only after commit (see `SetWinner` buyer stats loop)
- **Do NOT embed business logic in test files** — test files validate behavior, they should not contain production helpers
