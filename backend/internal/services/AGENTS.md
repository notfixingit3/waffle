# backend/internal/services/

## OVERVIEW

Core business logic for Project Syrup. All service functions access `db.Pool` directly as a global variable. No dependency injection, no ORM, no query builder.

## STRUCTURE

| File | Lines | Purpose |
|------|-------|---------|
| `waffle.go` | 449 | Waffle CRUD, spot lifecycle (claim, pay, release, winner), buyer stats updates, slug generation, Instagram handle normalization |
| `admins.go` | 255 | Admin CRUD, authentication, password reset tokens, login recording, bcrypt hashing, constant-time comparison |
| `buyers.go` | 164 | Buyer stats lookup, per-buyer waffle history with spot aggregation, activity event recording/retrieval |
| `reports.go` | 229 | Drought list, power buyers, monthly activity, payment lag, spot velocity (large SQL CTEs and window functions) |
| `auth.go` | 42 | JWT secret loading from env, generic token generation, password hash/check helpers |

## WHERE TO LOOK

| Task | File |
|------|------|
| Create waffle + generate spots transactionally | `waffle.go` — `CreateWaffle` |
| Claim spots with race-condition protection | `waffle.go` — `ClaimSpots` (uses `FOR UPDATE`) |
| Mark winner, update buyer stats, complete waffle | `waffle.go` — `SetWinner` |
| Admin login, password reset flow | `admins.go` — `AuthenticateAdmin`, `CreatePasswordResetToken`, `ValidatePasswordResetToken` |
| Buyer win/loss history per Instagram handle | `buyers.go` — `GetBuyerWaffleHistory` |
| Activity feed for a waffle | `buyers.go` — `GetActivityEvents`, `RecordActivityEvent` |
| Drought/power buyer/velocity reports | `reports.go` — `GetDroughtList`, `GetPowerBuyers`, `GetSpotVelocity` |
| JWT secret fallback for local dev | `auth.go` — `GetJWTSecret` |

## CONVENTIONS

- **Global db access:** `db.Pool.QueryRow`, `db.Pool.Exec`, `db.Pool.Begin` — no interfaces, no constructors
- **Raw SQL only:** Positional parameters `$1`, `$2`, ... — never string interpolation
- **Transactions:** `Begin` → `defer tx.Rollback` → `Commit` for multi-step ops (CreateWaffle, ClaimSpots, SetWinner, DeleteWaffle)
- **Error wrapping:** `fmt.Errorf("context: %w", err)` on every boundary
- **Instagram handle normalization:** `NormalizeInstagramHandle` strips `@` and lowercases; called inside `ClaimSpots`
- **Buyer stats:** `UpdateBuyerStats` uses `ON CONFLICT` upsert; called from `SetWinner` after commit
- **Password tokens:** Stored as bcrypt hashes; plaintext tokens returned to caller but never persisted

## ANTI-PATTERNS

- **No ORM or query builder** — raw SQL only
- **No dependency injection** — do not pass `*pgxpool.Pool` as a parameter or struct field
- **No service structs** — package-level functions only
- **No business logic in SQL** — keep calculations in Go where possible (reports are the exception)
- **No naked returns** — always name return values explicitly
- **No `sql.Null*` types** — scan into pointer fields directly (`*string`, `*time.Time`)
- **No logging in transactions** — log only after commit (see `SetWinner` buyer stats loop)
