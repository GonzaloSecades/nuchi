# Nuchi Go Backend

This directory contains the separate Go API that will replace the current
Hono/Drizzle/Neon backend.

The scaffold currently exposes a health endpoint, password auth with JWT
sessions (`/api/auth/register`, `/api/auth/login`, `/api/auth/refresh`,
`/api/auth/logout`), email verification and password reset
(`/api/auth/verify-email`, `/api/auth/password-reset/request`,
`/api/auth/password-reset/confirm`), and the database connection pool.
Resource endpoint handlers and RLS session binding (auth middleware) are
intentionally left to their own issue (#43+).

## Local Run

The API connects to PostgreSQL at startup and exits if it cannot, and it
requires `AUTH_JWT_SECRET` to be set (also fail-fast — see Auth below), so
start the database (and Mailpit, for the email flows) and export a secret
first:

```bash
docker compose up -d postgres mailpit
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
| `SMTP_ADDR` | `localhost:1025` | Outbound SMTP server `host:port`. Points at Mailpit in dev; unauthenticated. Validated at startup (both parts present, port numeric and in range). |
| `MAIL_FROM` | `nuchi@localhost` | From address on outgoing verification/reset mail. Parsed as an address at startup. |
| `APP_BASE_URL` | `http://localhost:3000` | Origin used to build verification/reset links. Must be origin-only — scheme and host, optional trailing `/`, no path, query, or fragment. Parsed and validated at startup, so a malformed or non-origin value fails fast rather than producing a broken link inside an email. Serving the app under a subpath would need `internal/mail` to join paths instead of replacing them. |
| `AUTH_VERIFICATION_TOKEN_TTL` | `48h` | Email verification token lifetime, as a Go duration. |
| `AUTH_RESET_TOKEN_TTL` | `30m` | Password reset token lifetime, as a Go duration. |

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
password reset send mail through `internal/mail` (see Email verification
and password reset below). Resource-route auth middleware/RLS session
binding is a separate, not-yet-implemented issue (#43).

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

Refresh rotates the token on every use: `POST /api/auth/refresh` atomically
consumes the presented cookie (`ConsumeRefreshToken` — a single `UPDATE`
that only one concurrent caller can win) and, on success, creates a
successor token and issues a new access token inside the same database
transaction. Replaying an already-consumed (or otherwise invalid) cookie
always returns `401 INVALID_REFRESH_TOKEN` and clears the cookie.

`POST /api/auth/logout` revokes the session identified by the cookie and clears
it. Per the OpenAPI contract, a missing, unknown, expired, or
already-revoked cookie is a `401 INVALID_REFRESH_TOKEN` on logout too — it
is not a silent no-op.

### Registration and login

`POST /api/auth/register` creates an unverified user (Argon2id hash only),
returns `201 { "message": ... }`, and — inside the same database
transaction that creates the user — creates an email verification token
(id `uuid.NewV7`, expiry `AUTH_VERIFICATION_TOKEN_TTL`). Only after that
transaction commits does the verification email send, asynchronously (see
Email verification and password reset below). Registration always 201s
regardless of whether the send succeeds.

`POST /api/auth/login` requires a verified email: unverified users get
`403 EMAIL_NOT_VERIFIED`. A wrong password and an unknown email both return
the identical `401 UNAUTHORIZED` body (enumeration safety) — there is no
way to distinguish the two from the response.

### Email verification and password reset

New package `internal/mail` defines the `Mailer` interface
(`SendVerificationEmail`, `SendPasswordResetEmail`) plus two
implementations: `SMTPMailer` (production/dev, talks to Mailpit) and
`CapturingMailer` (tests — records sends instead of talking to SMTP).
`AuthServer` takes a `Mailer` dependency; `cmd/api` wires `SMTPMailer` from
`SMTP_ADDR` / `MAIL_FROM` / `APP_BASE_URL`.

`SMTPMailer` sends plain-text mail, unauthenticated, over `net/smtp`.
`net/smtp` is a frozen standard-library package and `smtp.SendMail` applies
no timeout of its own, so `SMTPMailer` dials with a `net.Dialer`, sets an
explicit `conn.SetDeadline`, and drives `smtp.NewClient` over that
connection directly rather than calling `smtp.SendMail`. Each email body
contains a link built from the parsed, validated `APP_BASE_URL` with the
token in an escaped query parameter (`/verify-email?token=...` or
`/reset-password?token=...` — the paths the not-yet-built #51 frontend
pages will read), plus the raw token again on its own line for curl-based
testing. HTML templates are out of scope; bodies are plain text.

Both tokens are 256-bit (`crypto/rand`), stored only as their SHA-256 hash
(`auth.GenerateToken`/`auth.HashToken` — the same primitive refresh tokens
use), and are one-time: `Consume*Token` is a single atomic `UPDATE ...
WHERE used_at IS NULL AND expires_at > now()`, so a replayed, expired, or
unknown token always yields `401 INVALID_TOKEN`.

Sends are **asynchronous, best-effort, and strictly after commit**: once
the transaction that created the token row commits, the send runs in a
goroutine with its own 10-second-bounded context. A send failure is logged
(a message and the user id — **never the token or email body**) and can
never fail or delay the HTTP response. This decouples response timing from
SMTP delivery time, which is the OWASP-recommended mitigation for the
password-reset timing oracle described next.

`POST /api/auth/verify-email` takes `{ "token": "..." }`. Consuming the
token and marking the user verified share one database transaction
(`ConsumeEmailVerificationToken` → `MarkUserEmailVerified` → commit) so a
valid token is never burned by a later failure. If the user was already
verified, the response is still `200` — the token was validly consumed
either way (idempotent outcome).

`POST /api/auth/password-reset/request` takes `{ "email": "..." }` and
**always returns the same `200` message**, whether or not the account
exists — the contract declares no `401`/`404` here, deliberately, so the
response can never be used to enumerate registered emails. (The response
timing difference between a known and unknown account — a known account
does a small extra database transaction — is accepted residual risk,
recorded in `post-migration-improvements/`; async send removes the SMTP
component of the original timing oracle but not this smaller one.) For a
known account, issuance is serialized per user: `BEGIN` → `LockUser` (a row
lock, `FOR UPDATE`) → check the per-hour cap (max 3 tokens/hour,
`CountRecentPasswordResetTokens`) → invalidate the user's prior unused
reset tokens → create the new token → `COMMIT`. The row lock is what makes
several concurrent reset requests for the same user resolve one at a time
instead of racing past the cap check or each other's invalidate step. Over
the cap, issuance is silently skipped (still `200`, logged server-side).

`POST /api/auth/password-reset/confirm` takes `{ "token": "...", "password":
"..." }`. The new password is validated (rune-count ≥ 8, same rule as
register) **before any database work**, so a rejected weak password never
touches — and never burns — the token. On a valid password, one transaction
runs `ConsumePasswordResetToken` → `UpdateUserPassword` (new Argon2id hash)
→ `InvalidateUserPasswordResetTokens` (kill any other outstanding reset
token) → `RevokeAllUserRefreshTokens` (log out every session) → commit. A
password reset always ends every existing session, not just the one making
the request.

### Mailpit (local email testing)

Verification and password-reset emails are dev-only and go to
[Mailpit](https://github.com/axllent/mailpit), a local SMTP catcher with a
web UI. The canonical way to run it is the repo's `docker-compose.yml`:

```bash
docker compose up -d postgres mailpit
```

Mailpit's UI is then at <http://localhost:8025> — every email the API sends
appears there instead of leaving the machine.

If Docker isn't available, run the Mailpit binary directly instead
(equivalent SMTP/UI ports, same defaults the API config expects):

```bash
go install github.com/axllent/mailpit@latest
mailpit --listen 127.0.0.1:8025 --smtp 127.0.0.1:1025
```

With Mailpit running (either way) and the API up
(`SMTP_ADDR=localhost:1025` is already the default), walk the full flow:

```bash
# Register — triggers an async verification email
curl -i -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"correct-horse-battery"}'

# Open http://localhost:8025, find the "Verify your email" message, copy the
# token (also printed on its own line in the body).
curl -i -X POST http://localhost:8080/api/auth/verify-email \
  -H 'Content-Type: application/json' \
  -d '{"token":"<paste token>"}'

# Login now succeeds.
curl -i -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"correct-horse-battery"}'

# Request a reset — triggers an async "Reset your password" email.
curl -i -X POST http://localhost:8080/api/auth/password-reset/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com"}'

# Copy the token from Mailpit again, then confirm with a new password. This
# also revokes every existing session.
curl -i -X POST http://localhost:8080/api/auth/password-reset/confirm \
  -H 'Content-Type: application/json' \
  -d '{"token":"<paste token>","password":"a-different-password"}'
```

Mailpit also exposes a JSON API (`GET /api/v1/messages`, `GET
/api/v1/message/{ID}`) if scripting the token extraction is more
convenient than the UI.

### curl examples

Against a locally running API (`go run ./cmd/api`, `BACKEND_PORT=8080`):

```bash
# Register
curl -i -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"correct-horse-battery"}'

# Verify (see Mailpit section above for getting a real token in dev)
# curl -i -X POST http://localhost:8080/api/auth/verify-email \
#   -H 'Content-Type: application/json' -d '{"token":"<token>"}'

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
| `users.sql` | `users` CRUD, email verification marking, password update, `LockUser` (row lock used to serialize password-reset token issuance) |
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
logout, cross-user isolation), the email verification / password reset
lifecycle in `internal/http/email_flows_live_test.go` (full
register→verify and request→confirm flows, token replay/expiry, the
enumeration-safe response shape, the per-hour issuance cap, concurrent
issuance and concurrent verify races, and a fault-injection test proving a
mid-transaction failure after a token consume rolls back and leaves the
token usable), and the sqlc round-trip / RLS tests in `internal/db`:

```bash
cd backend
TEST_DATABASE_URL="postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable" go test ./...
```

`internal/mail`'s tests (`SMTPMailer` against a local fake SMTP listener,
`CapturingMailer` concurrency safety, link building) need no database and
run in the default `go test ./...` above.

CI runs these live tests too: the `backend` job in
[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) provisions a
Postgres 17 service, applies the migrations with goose, and exports
`TEST_DATABASE_URL`. The gate helper (`liveDatabaseURL`, defined per package
in `livedb_test.go`) therefore **fails** rather than skips when
`TEST_DATABASE_URL` is unset while `CI` is set — nearly all of the backend's
behavioral coverage sits behind that gate, and a silent skip would take it
out of CI without turning anything red. Outside CI an unset value still
skips, so `go test ./...` stays useful with no database running.
