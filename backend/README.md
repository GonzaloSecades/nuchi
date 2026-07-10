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
[goose](https://github.com/pressly/goose). No migration files exist yet
(tracked in a later issue); this issue only wires the location and tooling.

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

### Typed queries (sqlc)

Typed query code is generated with [sqlc](https://sqlc.dev) from
`backend/sqlc.yaml`: hand-written SQL in `internal/db/queries/` plus the
schema in `migrations/` generate Go code into `internal/db/gen/` (package
`dbgen`). `emit_json_tags` is disabled because JSON response shapes come
from the OpenAPI layer, not the DB layer. No queries exist yet and
`sqlc generate` has not been run (tracked in a later issue); this issue only
wires the config.

Install the pinned CLI version:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
```

Generate from `backend/` once queries exist:

```bash
cd backend
sqlc generate
```

## Verification

Tests run without a database. Running the API requires PostgreSQL to be up
(see Database above): startup pings the database and exits with status 1 if
it is unreachable.

```bash
cd backend
go test ./...
go run ./cmd/api
```
