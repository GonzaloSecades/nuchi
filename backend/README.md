# Nuchi Go Backend

This directory contains the separate Go API that will replace the current
Hono/Drizzle/Neon backend.

The scaffold currently exposes only a health endpoint and the database
connection pool. Auth, OpenAPI-bound handlers, migrations, and frontend
integration are intentionally left to their own issues.

## Local Run

The API connects to PostgreSQL at startup and exits if it cannot, so start
the database first:

```bash
docker compose up -d postgres
cd backend
go run ./cmd/api
```

The API listens on `0.0.0.0:8080` by default.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `BACKEND_HOST` | `0.0.0.0` | HTTP listen host |
| `BACKEND_PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | `postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable` | PostgreSQL connection string |

## Health Check

```bash
curl http://localhost:8080/api/health
```

Expected response:

```json
{
  "service": "nuchi-api",
  "status": "ok",
  "time": "2026-06-29T00:00:00Z"
}
```

## Database

The API owns a single `pgxpool.Pool` opened at startup from `DATABASE_URL`
(`internal/db`). The pool is verified with a 5-second-bounded `Ping` before
the HTTP server starts; if the ping fails, the process logs the failure and
exits with status 1 — the database is foundational infrastructure from this
point forward, not an optional dependency. Only the host portion of
`DATABASE_URL` is ever logged, never credentials. The pool is closed on
shutdown.

No request handlers use the pool yet; that lands in a later issue once
resource endpoints exist.

### Migrations (goose)

Migrations live in `backend/migrations/` and are applied with
[goose](https://github.com/pressly/goose).

| Migration | Purpose |
| --- | --- |
| `00001_auth_base.sql` | `citext` extension; `users`, `email_verification_tokens`, `password_reset_tokens`, `refresh_tokens` tables |
| `00002_finance_base.sql` | `accounts`, `categories`, `transactions` tables and their indexes |
| `00003_finance_rls.sql` | Row level security enable/force + ownership policies on `accounts`, `categories`, `transactions` |

Install the pinned CLI version:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.27.2
```

Run migrations from `backend/`:

```bash
cd backend
goose -dir migrations postgres "$DATABASE_URL" up
```

Resetting the local database is destructive and must be explicit — never
run `goose down`/`reset` as part of routine startup or automation.

### RLS

`accounts`, `categories`, and `transactions` have row level security
**enabled and forced** (`ALTER TABLE ... FORCE ROW LEVEL SECURITY`). FORCE
matters because the API connects as the `nuchi` role, which owns these
tables — Postgres exempts table owners from RLS unless it is explicitly
forced.

Every policy reads the current app user from the `app.user_id` session
setting:

- Each request handler (wired in #43) runs its queries inside a single
  transaction and binds the setting as the first statement with
  `SELECT set_config('app.user_id', $1, true)` — the parameterized
  equivalent of `SET LOCAL app.user_id = '...'` (`SET LOCAL` does not accept
  bind parameters; `set_config(..., true)` has the same transaction-local
  scope). The RLS tests use the same call.
- Policies read it as
  `NULLIF(current_setting('app.user_id', true), '')::uuid` —
  `current_setting(..., true)` returns `NULL` instead of raising when the
  setting was never set, and `NULLIF` turns an empty string into `NULL` too.
- A `NULL` app user can never equal a row's `user_id`, so an unset or empty
  `app.user_id` **fails closed**: zero rows are visible or writable, rather
  than erroring or exposing every user's data.
- `accounts` and `categories` are matched directly on `user_id`.
  `transactions` are matched through their required `account_id` (there is
  no direct `user_id` column on transactions).

`users` and the token tables have no RLS — they are auth-layer tables never
touched by user-scoped queries.

### Typed queries (sqlc)

Typed query code is generated with [sqlc](https://sqlc.dev) from
`backend/sqlc.yaml`: hand-written SQL in `internal/db/queries/` plus the
schema in `migrations/` generate Go code into `internal/db/gen/` (package
`dbgen`). Generation targets `sql_package: "pgx/v5"` so generated code binds
directly to `pgxpool.Pool` / `pgx.Tx` (the `DBTX` interface in
`internal/db/gen/db.go`) instead of `database/sql`. `emit_json_tags` is
disabled because JSON response shapes come from the OpenAPI layer, not the
DB layer.

`internal/db/gen/` is **generated code and is never hand-edited** — change
the `.sql` files in `internal/db/queries/` and re-run `sqlc generate`
instead. The generated package is committed to the repo (not gitignored) so
`go build`/`go test` work without requiring sqlc to be installed.

Query files, one per domain:

| File | Covers |
| --- | --- |
| `users.sql` | `users` CRUD, email verification marking, password update |
| `auth_tokens.sql` | `email_verification_tokens`, `password_reset_tokens` (atomic one-time consume, invalidate-prior, rate-limit counts), `refresh_tokens` (create, get-valid, revoke, revoke-all) |
| `accounts.sql` | `accounts` CRUD + bulk delete |
| `categories.sql` | `categories` CRUD + bulk delete (mirrors `accounts.sql`) |
| `transactions.sql` | `transactions` CRUD, list (joined account/category names), bulk create, bulk delete — every query is scoped through the owning account |
| `summary.sql` | period totals, category spending, daily totals aggregates |

Every owned-resource query (accounts/categories/transactions) carries an
explicit ownership predicate (`user_id = $N`, or a join/EXISTS through the
owning account for transactions) even though RLS also enforces it — RLS is
the backstop, not the mechanism (see RLS above).

`token_hash`-based one-time consume queries (`Consume*Token`) are a single
`UPDATE ... WHERE used_at IS NULL AND expires_at > now() RETURNING ...`
statement so two concurrent submissions of the same token cannot both
succeed; a failed consume (already used, expired, or unknown) surfaces as
`pgx.ErrNoRows`.

`BulkCreateTransactions` inserts every row in one round trip. Because sqlc's
static analyzer (no live database is configured for generation) cannot
resolve Postgres's multi-argument `unnest(a, b, c, ...)` form used directly
in a `FROM` clause, the query instead unnests each parameter array
separately with `WITH ORDINALITY` and re-joins them on their shared ordinal
position — equivalent to the multi-arg form, built entirely from the
single-arg `unnest(anyarray)` overload sqlc already knows.

Install the pinned CLI version:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
```

Regenerate from `backend/` after changing any file in
`internal/db/queries/` or the migrations:

```bash
cd backend
sqlc generate
```

Commit the resulting changes under `internal/db/gen/` alongside the query
changes that produced them.

## Verification

Tests run without a database. Running the API requires PostgreSQL to be up
(see Database above): startup pings the database and exits with status 1 if
it is unreachable.

```bash
cd backend
go test ./...
go run ./cmd/api
```
