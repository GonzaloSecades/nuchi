# Nuchi

Nuchi is a personal finance app for tracking accounts, categories, transactions, CSV imports, and dashboard analytics.

The app is a Next.js frontend with TanStack Query, served by a separate Go API over PostgreSQL. The Go backend replacement tracked by [issue #18](https://github.com/GonzaloSecades/nuchi/issues/18) has reached parity: the Go service, owned JWT auth, the generated OpenAPI client, and the frontend cutover are all in place, and the legacy Hono API has been removed. The remaining Drizzle and Clerk packages are unused and are removed in [#85](https://github.com/GonzaloSecades/nuchi/issues/85).

## Migration Status

Current migration parent: [#18 Spec Go backend replacement for Hono/Drizzle/Neon](https://github.com/GonzaloSecades/nuchi/issues/18).

Completed migration issues:

- [#19](https://github.com/GonzaloSecades/nuchi/issues/19) Scaffold Go backend service and health route.
- [#20](https://github.com/GonzaloSecades/nuchi/issues/20) Add Docker Compose Postgres and local mail catcher.
- [#34](https://github.com/GonzaloSecades/nuchi/issues/34) Document current API parity fixtures.
- [#35](https://github.com/GonzaloSecades/nuchi/issues/35) Add OpenAPI scaffold and generation commands.
- [#28](https://github.com/GonzaloSecades/nuchi/issues/28) Finalize Go backend replacement spec.
- [#36](https://github.com/GonzaloSecades/nuchi/issues/36) Define shared API error and auth contract.

Next migration issue: [#29 Backend Migration 03: Define full OpenAPI contract](https://github.com/GonzaloSecades/nuchi/issues/29). Work must continue strictly in sequence: a ticket should be merged before the next starts, only the next unblocked low-risk ticket may be marked agent-ready, and high-risk migration tickets remain attended work.

The Hono API and its typed client were removed in [#84](https://github.com/GonzaloSecades/nuchi/issues/84). The Drizzle schema and Clerk packages are no longer used by any code path and are removed in [#85](https://github.com/GonzaloSecades/nuchi/issues/85).

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
- Owned email/password auth will replace Clerk later, with JWT access tokens and HttpOnly refresh-token cookies.
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
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_your_key_here
CLERK_SECRET_KEY=sk_test_your_key_here
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

Do not commit `.env.local`. Real Clerk keys are required for `bun run build`; the placeholder keys in `.env.example` document shape only and are not valid credentials.

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

`bun dev` runs the local development script. `bun run build` validates the Next.js production build and requires real Clerk environment variables.

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

Three terminals:

```bash
docker compose up -d postgres
```

```bash
cd backend && go run ./cmd/api
```

```bash
bun dev
```

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

Generation is intentionally deferred for normal work until #29 fills the full resource contract and generator versions/network use are pinned or explicitly approved. Generated code belongs only in generated paths:

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

Current Drizzle commands:

```bash
bun run db:generate
bun run db:migrate
bun run db:studio
bun run db:seed
```

During the current app phase, keep `db/schema.ts` as the database source of truth and keep `drizzle/` migrations in sync. The Go migration will move persistence toward `goose` migrations and `sqlc` queries in later issues; do not start that conversion outside the active migration issue sequence.

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
7. Next: #29 full OpenAPI contract.
8. Later: Go database foundation, auth/finance migrations, `sqlc` queries, owned auth/session/email flows, RLS-backed DB access, resource API parity, summary parity, frontend rewrite/client/hook migration, custom auth pages, and final legacy removal.

High-risk issues, including OpenAPI completion, database/RLS work, auth, resource parity, frontend client replacement, and legacy removal, remain attended work and should not be marked agent-ready.

## Troubleshooting

### Docker Engine Not Reachable

If `docker compose up` or `docker compose ps` cannot connect, start Docker Desktop or the Docker Engine service and rerun the command. On Windows, make sure the shell can access the same Docker context as Docker Desktop.

### Build Fails With Clerk Placeholder Keys

`.env.example` contains placeholder Clerk values. `bun run build` needs real `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` and `CLERK_SECRET_KEY` values in `.env.local`.

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
