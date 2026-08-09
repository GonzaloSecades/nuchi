# Nuchi

Nuchi is a personal finance app for tracking accounts, categories, transactions, CSV imports, and dashboard analytics.

The app is a Next.js frontend with TanStack Query, served by a separate Go API over PostgreSQL. The Go backend replacement tracked by [issue #18](https://github.com/GonzaloSecades/nuchi/issues/18) is complete: the Go service, owned JWT auth, the generated OpenAPI client, and the frontend cutover are all in place, and the legacy stack is gone — Hono in [#84](https://github.com/GonzaloSecades/nuchi/issues/84), Drizzle/Neon/Clerk in [#85](https://github.com/GonzaloSecades/nuchi/issues/85). Only [#90](https://github.com/GonzaloSecades/nuchi/issues/90) remains, to retire the differential parity harness.

## Migration Status

Current migration parent: [#18 Spec Go backend replacement for Hono/Drizzle/Neon](https://github.com/GonzaloSecades/nuchi/issues/18).

Completed migration issues (the early tickets; the full set is on the issue tracker):

- [#19](https://github.com/GonzaloSecades/nuchi/issues/19) Scaffold Go backend service and health route.
- [#20](https://github.com/GonzaloSecades/nuchi/issues/20) Add Docker Compose Postgres and local mail catcher.
- [#34](https://github.com/GonzaloSecades/nuchi/issues/34) Document current API parity fixtures.
- [#35](https://github.com/GonzaloSecades/nuchi/issues/35) Add OpenAPI scaffold and generation commands.
- [#28](https://github.com/GonzaloSecades/nuchi/issues/28) Finalize Go backend replacement spec.
- [#36](https://github.com/GonzaloSecades/nuchi/issues/36) Define shared API error and auth contract.

Next migration issue: [#90](https://github.com/GonzaloSecades/nuchi/issues/90), which retires the differential parity harness and drops `pg` — the last teardown ticket before [#27](https://github.com/GonzaloSecades/nuchi/issues/27) closes. Work must continue strictly in sequence: a ticket should be merged before the next starts, only the next unblocked low-risk ticket may be marked agent-ready, and high-risk migration tickets remain attended work.

The Hono API and its typed client were removed in [#84](https://github.com/GonzaloSecades/nuchi/issues/84); the Drizzle schema, the `drizzle/` migrations and the Clerk packages in [#85](https://github.com/GonzaloSecades/nuchi/issues/85).

## Stack

- Next.js App Router, React, and TypeScript.
- Owned email/password auth with JWT access tokens and HttpOnly refresh cookies, served by the Go API.
- TanStack Query for server-state hooks, over the generated OpenAPI client in `lib/api/`.
- Bun package manager and runtime scripts.
- Go API under `backend/` (chi, pgxpool, sqlc, goose) serving all of `/api/*` over PostgreSQL.
- Docker Compose services for Postgres and Mailpit.
- OpenAPI source under `openapi/`.
- Graphify knowledge graph under `graphify-out/`.
- Tailwind CSS, shadcn/ui primitives, and Recharts for the frontend.

## Product Surface

- Dashboard analytics with KPI cards, period comparisons, category breakdowns, and daily charts.
- Account management with CRUD and bulk delete.
- Category management with CRUD, bulk delete, and case-insensitive uniqueness.
- Transaction management with CRUD, bulk actions, account/category assignment, date filtering, and mutation rate limits.
- CSV transaction import with column mapping and account assignment.
- Global account and date filters scoped through URL query params.

## Architecture

### Current App Shape

- `app/` contains App Router pages and layouts. There is no longer a Next API route; `/api/*` is proxied to the Go API.
- `backend/` contains the Go API for accounts, categories, transactions, summary, and auth.
- `features/<domain>/` contains UI, hooks, and domain-specific client logic.
- `lib/api/` exposes the generated OpenAPI client used by TanStack Query hooks.
- `backend/migrations/` holds the goose migrations that own the database schema.
- Auth-sensitive reads and writes are scoped by `auth.userId`.
- Transaction amounts are stored as signed integer milliunits.

### Target Go Backend Shape

The target architecture keeps Next.js serving the frontend while a separate Go API serves `/api/*` behind same-origin rewrites/proxying.

- Go service listens on `localhost:8080` by default.
- `chi` will own HTTP routing and middleware.
- `pgxpool` will own database pooling.
- `goose` will own SQL migrations.
- `sqlc` will own typed SQL access.
- OpenAPI will be the contract source of truth for Go server types and TypeScript client/types.
- Owned email/password auth, with JWT access tokens and HttpOnly refresh-token cookies.
- PostgreSQL RLS will become a required ownership backstop.

### Legacy And Reference Code

The Hono routes and `lib/hono.ts` are gone; git history is their reference. The behavior they defined is frozen in `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`, which the Go API was ported against and which stays as the migration's historical record.

## Local Setup

### Prerequisites

- Bun.
- Node.js compatible with the Next.js toolchain.
- Go for backend commands.
- Docker Desktop or Docker Engine for local Postgres and Mailpit.
- GitHub CLI is optional for issue/PR work.
- Graphify is optional but recommended for codebase navigation.

### Install Dependencies

```bash
bun install
```

### Environment

Copy `.env.example` to `.env.local` and fill local values:

```bash
DATABASE_URL=postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable
NEXT_PUBLIC_API_URL=http://localhost:3000
SMTP_ADDR=localhost:1025
MAILPIT_WEB_URL=http://localhost:8025
MAIL_FROM=nuchi@localhost
APP_BASE_URL=http://localhost:3000
AUTH_VERIFICATION_TOKEN_TTL=48h
AUTH_RESET_TOKEN_TTL=30m
```

The Go backend reads `SMTP_ADDR` as a single `host:port`; there is no
`SMTP_HOST`/`SMTP_PORT` split. `APP_BASE_URL` must be origin-only (`http`
or `https`, host, optional trailing `/`, no path, query, fragment, or
userinfo) — it is validated at startup, because it becomes a clickable link
inside verification and reset emails. Full backend table:
[`backend/README.md`](backend/README.md).

Do not commit `.env.local`. `bun run build` needs no credentials of any kind — that requirement disappeared with Clerk in #85. `AUTH_JWT_SECRET` is required by the Go API at startup, not by the frontend build; generate one with `openssl rand -base64 48` and never commit it.

## Docker And Local Services

Start local Postgres and Mailpit:

```bash
docker compose up -d postgres mailpit
```

Check service state:

```bash
docker compose ps
```

Local service defaults:

- Postgres: `postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable`.
- Mailpit UI: `http://localhost:8025`.
- Mailpit SMTP: `localhost:1025`.

### Postgres roles

The container's bootstrap superuser is `postgres`. The application role
`nuchi` is an ordinary (non-superuser) role created on first startup by
`docker/postgres/init/01-app-role.sql`, which also hands it ownership of the
`public` schema so the goose migrations create tables it owns. Applications
still connect as `nuchi`, so `DATABASE_URL` is unchanged.

The two must stay separate: **superusers bypass row level security
entirely, `FORCE` included**, so an app connecting as the bootstrap
superuser would have the ownership policies in
`backend/migrations/00003_finance_rls.sql` silently not apply — and the RLS
tests would pass while proving nothing. PostgreSQL also refuses to let the
bootstrap superuser drop its own `SUPERUSER` attribute, so this cannot be
corrected after the fact.

Docker init scripts run only when the data volume is first created. **A
volume created before this change still has `nuchi` as the bootstrap
superuser**; reset it once (destructive — see below) to pick up the split:

```bash
docker compose down --volumes
docker compose up -d postgres mailpit
```

Check Postgres readiness:

```bash
docker compose exec postgres pg_isready -U postgres -d nuchi
```

Destroy local service data only when you intentionally want a destructive reset:

```bash
docker compose down --volumes
```

## Frontend Commands

```bash
bun dev
bun run lint
bun run build
bun test
```

`bun dev` starts the Docker Compose `postgres` and `mailpit` services, waits for Postgres, then runs the Next dev server. It does not run migrations — goose owns those, from `backend/` — and it does not start the Go API, which you run separately with `cd backend && go run ./cmd/api`. `bun run build` validates the Next.js production build and needs no environment variables.

## Backend Commands

```bash
cd backend
go test ./...
go run ./cmd/api
```

The Go API listens on `0.0.0.0:8080` by default.

## Running Next and Go Together

The two run as separate processes: Next serves the UI, Go serves the API. Next
proxies `/api/*` to Go by default, so the browser only talks to the Next origin,
cookies stay same-origin, and no CORS setup is required.

Start Postgres:

```bash
docker compose up -d postgres
```

**Apply the migrations before starting the API.** This is not optional on a
fresh checkout, and nothing does it for you: a new Compose volume runs only
`docker/postgres/init/01-app-role.sql`, which creates the `nuchi` role and the
`citext` extension and no tables at all. The Go API does not migrate on startup
— it calls `VerifyRLSActive` first and refuses to serve unless row level
security is active on `accounts`, `categories` and `transactions`, so against an
unmigrated database it exits immediately rather than starting.

```bash
cd backend
go install github.com/pressly/goose/v3/cmd/goose@v3.27.2   # once
goose -dir migrations postgres "$DATABASE_URL" up
```

Then the two long-running processes, one terminal each:

```bash
cd backend && go run ./cmd/api
```

```bash
bun dev
```

`bun dev` deliberately does not run migrations. Schema ownership belongs to the
backend, and the frontend does not connect to Postgres at all.

### Go API proxy configuration

The proxy is **unconditional** — the Go API is the only backend, so there is no
flag to turn it off. Configure its origin in `.env.local` when it is not the
local default:

```bash
GO_API_URL=http://localhost:8080
```

One thing to know: the rewrite runs in Next's `beforeFiles` phase. Nothing
under `app/api/` competes with it any more, but `beforeFiles` keeps the proxy
authoritative — a Next route added there later would shadow the backend from
`afterFiles`, and would not from `beforeFiles`.

`GO_API_URL` must be origin-only (no credentials, path, query, or fragment).
The matched request path is appended to it, so a trailing path would produce
destinations like `/base/api/...`; the build fails loudly on that rather than
leaving you to debug a 404. Credentials are rejected because `URL.origin`
silently removes them, which would otherwise hide a bad deployment value.

Health check:

```bash
curl http://localhost:8080/api/health
```

Expected shape:

```json
{
  "service": "nuchi-api",
  "status": "ok",
  "time": "2026-06-29T00:00:00Z"
}
```

The backend exposes health, owned authentication/session flows, accounts,
categories, transactions (including bulk operations), and summary analytics.
`openapi/nuchi.openapi.json` is the authoritative HTTP contract.

## OpenAPI

The hand-edited contract source is `openapi/nuchi.openapi.json`. OpenAPI is intended to become the source of truth for Go server types and TypeScript client/types.

Validate the current contract:

```bash
bun run openapi:validate
```

Generation commands are wired:

```bash
bun run openapi:gen:go
bun run openapi:gen:ts
```

The contract is complete and both sides are generated from it, so regeneration is a normal part of a contract change rather than a deferred step. Generated code belongs only in generated paths:

- Go server types: `backend/internal/openapi/generated.gen.go`.
- TypeScript fetch client/types: `lib/api/generated/typescript-fetch/`.

The shared contract from #36 establishes structured API errors:

```json
{
  "error": {
    "code": "SOME_ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

App resource success responses should preserve the existing `{ "data": ... }` envelope where practical. App resource endpoints use Bearer access-token auth in the target contract, while refresh and logout use the documented HttpOnly refresh-token cookie.

## Database

The database is owned entirely by the Go API. There is no ORM and no frontend database access; `db/`, `drizzle/` and the `db:*` scripts were removed in #85.

Schema changes are goose migrations under `backend/migrations/`, reached through sqlc queries in `backend/internal/db/queries/`. See [`backend/README.md`](backend/README.md) for the goose commands and the pinned version.

## Graphify

This repo has a tracked Graphify knowledge graph in `graphify-out/`.

Useful commands:

```bash
graphify query "What owns transaction data?"
graphify explain "Go Backend Replacement Spec"
graphify update .
```

Use Graphify before broad codebase or architecture work when `graphify-out/graph.json` exists. After modifying code or docs, run `graphify update .` when available so the graph stays current.

Portable Graphify artifacts are tracked, including `graphify-out/graph.json`, `graphify-out/GRAPH_REPORT.md`, `graphify-out/manifest.json`, and `graphify-out/.graphify_labels.json`. Local/cache outputs such as `.graphify_*` intermediates, `graphify-out/cache/`, dated run directories, `graph.html`, and `cost.json` are ignored by `.gitignore`.

## Migration Roadmap

The replacement sprint proceeds in this order:

1. Completed: Go backend scaffold and health route.
2. Completed: Docker Compose Postgres and Mailpit.
3. Completed: API parity fixtures.
4. Completed: OpenAPI scaffold and generation command documentation.
5. Completed: Go backend replacement spec.
6. Completed: shared API error/auth contract.
7. Completed: the full OpenAPI contract, Go database foundation, auth/finance migrations, `sqlc` queries, owned auth/session/email flows, RLS-backed DB access, resource and summary API parity, the frontend rewrite/client/hook migration, and custom auth pages.
8. Completed: the legacy Hono API surface and `USE_GO_API` removed (#84).
9. Completed: Drizzle, Neon and Clerk removed (#85).
10. Next: #90 retires the differential parity harness and drops `pg`. #27 closes when it lands.

High-risk issues, including OpenAPI completion, database/RLS work, auth, resource parity, frontend client replacement, and legacy removal, remain attended work and should not be marked agent-ready.

## Troubleshooting

### Docker Engine Not Reachable

If `docker compose up` or `docker compose ps` cannot connect, start Docker Desktop or the Docker Engine service and rerun the command. On Windows, make sure the shell can access the same Docker context as Docker Desktop.

### GitHub Project Board Requires Scope

GitHub project board operations may require the GitHub CLI to be authenticated with the `project` scope. Re-authenticate or refresh scopes before managing project metadata.

### Graphify Warnings

Graphify may warn about local cache files, graph health, or changed artifacts after hooks/incremental updates. Dirty `graphify-out/` files are expected after updates; inspect them, but do not treat them as a reason to skip Graphify.

## Contributing And Workflow

- Keep feature code in `features/<domain>/`.
- Call the API through the generated OpenAPI client in `lib/api/` rather than ad hoc `fetch`.
- Keep server-state logic in TanStack Query hooks.
- Preserve auth-sensitive ownership scoping on every read and write.
- Store transaction amounts in milliunits.
- Prefer existing `components/ui/*` primitives.
- Avoid `any` in app code.
- Do not leave debug routes, raw `console.log`, or dead commented code in production paths.
- Do not commit `.env.local` or secrets.
- Codex-created branch names must not include `codex`.
- Issue-related PR titles must use `[Issue - #<number>] <PR title>`.

## Canonical Docs

- `AGENTS.md`: repo rules, verification expectations, and PR rules.
- `docs/CODEX_CONTEXT.md`: current architecture and feature guide.
- `docs/specs/18-go-backend-replacement/spec.md`: Go backend replacement plan.
- `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`: the legacy Hono behavior the Go API was ported against. Historical record.
- `backend/README.md`: Go backend scaffold details.
- `openapi/README.md`: OpenAPI layout, validation, generation, and shared contract notes.
- `PR_REVIEW_TECH_DEBT_CONSOLIDATED.md`: active debt tracker.
