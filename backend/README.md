# Nuchi Go Backend

This directory contains the separate Go API that will replace the current
Hono/Drizzle/Neon backend.

The scaffold currently exposes a health endpoint, password auth with JWT
sessions (`/api/auth/register`, `/api/auth/login`, `/api/auth/refresh`,
`/api/auth/logout`), and the database connection pool. Email verification,
password reset, resource endpoint handlers, and RLS session binding are
intentionally left to their own issues (#42, #43+).

## Local Run

The API connects to PostgreSQL at startup and exits if it cannot, and it
requires `AUTH_JWT_SECRET` to be set (also fail-fast — see Auth below), so
start the database and export a secret first:

```bash
docker compose up -d postgres
cd backend
export AUTH_JWT_SECRET="$(openssl rand -base64 48)"
go run ./cmd/api
```

The API listens on `0.0.0.0:8080` by default.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `BACKEND_HOST` | `0.0.0.0` | HTTP listen host |
| `BACKEND_PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | `postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable` | PostgreSQL connection string |
| `AUTH_JWT_SECRET` | *(none — required)* | HMAC key for signing/verifying access tokens. Startup exits with a clear error if unset or shorter than 32 bytes. Generate one with `openssl rand -base64 48`. |
| `AUTH_ACCESS_TOKEN_TTL` | `30m` | Access-token lifetime, as a Go duration (e.g. `30m`, `1h`) |
| `AUTH_REFRESH_TOKEN_TTL` | `720h` (30 days) | Refresh-token lifetime, as a Go duration |
| `AUTH_COOKIE_SECURE` | `false` | Sets the refresh cookie's `Secure` attribute. Must be `true` in any deployed environment — `false` only works over plain HTTP, which is fine for local dev but never for a real deployment. |

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

## Auth

Owned password auth replaces Clerk. Implemented in `internal/auth`
(Argon2id password hashing, opaque token generation, JWT issue/verify) and
wired into HTTP handlers in `internal/http/auth.go`. Email verification and
password reset (`/api/auth/verify-email`, `/api/auth/password-reset/*`) and
resource-route auth middleware/RLS session binding are separate,
not-yet-implemented issues (#42, #43).

### Password hashing

Passwords are hashed with Argon2id (`golang.org/x/crypto/argon2`):
time=3, memory=64 MiB, parallelism=2, a random 16-byte salt, a 32-byte key,
encoded as a standard PHC string
(`$argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>`). Verification re-derives
the key using the parameters embedded in the stored hash, not the package's
current constants, so a future parameter change never breaks existing
hashes. Comparison is constant-time. A login attempt against an email that
does not exist still runs a full Argon2id verification against a fixed
dummy hash, so response timing cannot be used to enumerate registered
emails.

### JWT access tokens

Access tokens are signed HS256 JWTs (`github.com/golang-jwt/jwt/v5`) with
exactly three claims: `sub` (user UUID), `iat`, and `exp`. Nothing else —
email and verification status travel in the response body
(`AuthSessionResponse.user`), not the token. `AUTH_JWT_SECRET` is the HMAC
key; there is no default, and startup exits with a clear error if it is
unset or shorter than 32 bytes:

```bash
export AUTH_JWT_SECRET="$(openssl rand -base64 48)"
```

### Refresh tokens and the session cookie

A refresh token is 32 bytes of `crypto/rand`, base64url-encoded (no
padding) as the value handed to the client; only its SHA-256 hex digest is
ever stored (`refresh_tokens.token_hash`), so a database read alone can
never produce a usable token.

The refresh token travels as an HttpOnly cookie, never in a JSON body:

| Attribute | Value |
| --- | --- |
| Name | `nuchi_refresh_token` |
| Path | `/api/auth` |
| HttpOnly | always |
| Secure | from `AUTH_COOKIE_SECURE` |
| SameSite | `Lax` |
| Max-Age | `AUTH_REFRESH_TOKEN_TTL` on login/refresh; `0` (cleared) on logout or an invalid/expired refresh attempt |

Refresh rotates the token on every use: `POST /auth/refresh` atomically
consumes the presented cookie (`ConsumeRefreshToken` — a single `UPDATE`
that only one concurrent caller can win) and, on success, creates a
successor token and issues a new access token inside the same database
transaction. Replaying an already-consumed (or otherwise invalid) cookie
always returns `401 INVALID_REFRESH_TOKEN` and clears the cookie.

`POST /auth/logout` revokes the session identified by the cookie and clears
it. Per the OpenAPI contract, a missing, unknown, expired, or
already-revoked cookie is a `401 INVALID_REFRESH_TOKEN` on logout too — it
is not a silent no-op.

### Registration and login

`POST /auth/register` creates an unverified user (Argon2id hash only) and
returns `201 { "message": ... }`. It does **not** send a verification email
or create a verification token yet — that lands in #42. Until then, mark a
user verified directly:

```sql
UPDATE users SET email_verified_at = now() WHERE email = 'someone@example.com';
```

`POST /auth/login` requires a verified email: unverified users get
`403 EMAIL_NOT_VERIFIED`. A wrong password and an unknown email both return
the identical `401 UNAUTHORIZED` body (enumeration safety) — there is no
way to distinguish the two from the response.

### curl examples

Against a locally running API (`go run ./cmd/api`, `BACKEND_PORT=8080`):

```bash
# Register
curl -i -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"correct-horse-battery"}'

# Mark verified (stand-in for #42's email flow)
# psql "$DATABASE_URL" -c "UPDATE users SET email_verified_at = now() WHERE email = 'you@example.com';"

# Login — save the refresh cookie to a jar
curl -i -c cookies.txt -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"correct-horse-battery"}'

# Refresh — reads the cookie from the jar, writes the rotated cookie back
curl -i -b cookies.txt -c cookies.txt -X POST http://localhost:8080/api/auth/refresh

# Logout — revokes the session and clears the cookie
curl -i -b cookies.txt -X POST http://localhost:8080/api/auth/logout
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

Every owned-resource **read, update, and delete** carries an explicit
ownership predicate (`user_id = $N`, or a join/EXISTS through the owning
account for transactions) even though RLS also enforces it — RLS is the
backstop, not the mechanism (see RLS above). The transaction **INSERTs**
(`CreateTransaction`, `BulkCreateTransactions`) are the deliberate
exception: handlers validate `account_id`/`category_id` ownership first
(legacy behavior — friendly 400/404 before insert) and RLS `WITH CHECK`
hard-rejects anything that slips through. An ownership join inside the bulk
INSERT would *silently drop* unowned rows — a partial import — whereas the
`WITH CHECK` failure is loud and atomic, which is the safer failure mode
for financial data.

`token_hash`-based one-time consume queries (`Consume*Token`) are a single
`UPDATE ... WHERE used_at IS NULL AND expires_at > now() RETURNING ...`
statement so two concurrent submissions of the same token cannot both
succeed; a failed consume (already used, expired, or unknown) surfaces as
`pgx.ErrNoRows`.

`BulkCreateTransactions` inserts every row in one round trip from a single
structured `jsonb` parameter (`jsonb_to_recordset`): the caller marshals the
rows as one JSON array (fields `id`, `amount`, `payee`, `notes`, `date`,
`account_id`, `category_id`, `currency`; dates in UTC — the `timestamp`
cast drops any zone suffix). One parameter makes per-row integrity
structural — there are no parallel arrays whose lengths can drift, JSON
nulls land as SQL NULLs, and a batch containing an invalid row fails
atomically (single INSERT statement). Refresh-token rotation must use the
atomic `ConsumeRefreshToken` (same one-winner semantics as `Consume*Token`);
`GetRefreshTokenByHash` is a read-only validity check only.

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

Most tests run without a database and without `AUTH_JWT_SECRET` set (the
config package's own fail-fast test sets it per-test). Running the API
requires PostgreSQL to be up (see Database above) and `AUTH_JWT_SECRET` set
(see Auth above): startup pings the database and validates configuration,
exiting with status 1 if either fails.

```bash
cd backend
go vet ./...
go test ./...
export AUTH_JWT_SECRET="$(openssl rand -base64 48)"
go run ./cmd/api
```

Some tests are live-database-gated (skipped unless `TEST_DATABASE_URL` is
set), including the full auth HTTP lifecycle in
`internal/http/auth_live_test.go` (register, login, refresh rotation,
logout, cross-user isolation) and the sqlc round-trip / RLS tests in
`internal/db`:

```bash
cd backend
TEST_DATABASE_URL="postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable" go test ./...
```
