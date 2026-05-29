# backend/migrations/

## OVERVIEW

Database schema migrations for Project Syrup, managed by golang-migrate.

## STRUCTURE

| File | Purpose |
|------|---------|
| `001_initial_schema.up.sql` | Core tables: `waffles`, `spots` (with `status CHECK`), `activity_events`, `buyer_stats` |
| `001_initial_schema.down.sql` | Drop core tables |
| `002_add_media_and_archive.up.sql` | Add `instagram_media_links`, `archived_at`, `archived_by` columns |
| `002_add_media_and_archive.down.sql` | Remove added columns |
| `003_add_admins.up.sql` | `admins` table, `password_reset_tokens`, seed super_admin user |
| `003_add_admins.down.sql` | Drop admin tables |
| `004_seed_example_waffle.up.sql` | Seed example waffle data |
| `004_seed_example_waffle.down.sql` | Remove seed data |

## WHERE TO LOOK

| Task | File |
|------|------|
| Add new table or column | Create `00X_description.up.sql` + matching `.down.sql` |
| Check spot status constraint | `001_initial_schema.up.sql` |
| See admin schema | `003_add_admins.up.sql` |
| Understand archive columns | `002_add_media_and_archive.up.sql` |
| Verify migration ran | Check `schema_migrations` table in Postgres |

## CONVENTIONS

- Sequential numeric versioning: `001`, `002`, `003`, `004`
- Every `.up.sql` must have a matching `.down.sql`
- Migrations run automatically on startup via `db.RunMigrations()`
- Embedded FS path: `file:///app/migrations`
- Use `IF NOT EXISTS` / `IF EXISTS` guards where practical

## ANTI-PATTERNS

- **Never edit an existing migration file after it has been deployed.** Create a new migration instead.
- **Never skip `.down.sql` files.** They are required for rollback safety.
- **Never use non-sequential numbering.** Gaps or out-of-order numbers break golang-migrate.
- **Never put business logic in migrations.** Schema changes only, no Go code or triggers.
- **Never manually modify `schema_migrations` table.** Let the tool manage version tracking.
