# Nuchi

Nuchi is a personal finance app for tracking accounts, categories, transactions, CSV imports, and dashboard analytics.

The app is currently a full-stack Next.js application with Clerk authentication, Hono API routes, Drizzle ORM, TanStack Query, and PostgreSQL. The project is also in the middle of the Go backend replacement tracked by [issue #18](https://github.com/GonzaloSecades/nuchi/issues/18): the Go service, Docker Compose services, OpenAPI scaffold, API parity fixtures, replacement spec, and shared API auth/error contract are in place, but the production app still depends on the existing Next/Hono/Drizzle/Clerk path until Go parity is implemented.

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

Do not remove the Hono routes, Drizzle schema, Clerk auth, or typed Hono client yet. They are still the current app backend and the parity reference for the Go migration.

## Stack

- Next.js App Router, React, and TypeScript.
- Clerk authentication in the current frontend/app runtime.
- Hono API mounted in `app/api/[[...route]]` as the current backend and legacy parity reference.
- Drizzle ORM with PostgreSQL. Local development uses Docker Compose Postgres; existing production-oriented configuration still supports Neon/Postgres-style URLs.
- TanStack Query for server-state hooks.
- Bun package manager and runtime scripts.
- Go backend scaffold under `backend/`, currently exposing `/api/health`.
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

- `app/` contains App Router pages, layouts, and the Hono API entrypoint.
- `app/api/[[...route]]` contains the current JSON API for accounts, categories, transactions, and summary.
- `features/<domain>/` contains UI, hooks, and domain-specific client logic.
- `lib/hono.ts` exposes the typed Hono client used by TanStack Query hooks.
- `db/schema.ts` is the current database source of truth, with generated Drizzle migrations in `drizzle/`.
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

The current Hono routes, Drizzle schema, Clerk usage, and `lib/hono.ts` must remain until the Go backend reaches verified parity. They define behavior that the Go API must preserve unless the replacement spec names an intentional product change. The current parity fixtures live in `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`.

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
SMTP_HOST=localhost
SMTP_PORT=1025
MAILPIT_WEB_URL=http://localhost:8025
```

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

Check Postgres readiness:

```bash
docker compose exec postgres pg_isready -U nuchi -d nuchi
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

The scaffolded Go API listens on `0.0.0.0:8080` by default.

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

The backend currently exposes only the health endpoint. Database access, auth, migrations, resource handlers, and frontend integration remain future migration issues.

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
- Prefer extending typed Hono routes over ad hoc `fetch` while Hono remains current.
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
- `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`: current Hono API behavior reference.
- `backend/README.md`: Go backend scaffold details.
- `openapi/README.md`: OpenAPI layout, validation, generation, and shared contract notes.
- `PR_REVIEW_TECH_DEBT_CONSOLIDATED.md`: active debt tracker.
