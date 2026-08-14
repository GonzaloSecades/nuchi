# Nuchi Codex Guide

## Stack

- Next.js App Router (frontend only — there is no Next API layer)
- Go API (chi, pgxpool, sqlc, goose) over PostgreSQL, reached same-origin
  through the `/api/*` rewrite in `next.config.ts`
- Owned JWT auth (`lib/auth/`)
- TanStack Query + the generated OpenAPI client in `lib/api/`
- Bun package manager
- No ORM and no Clerk: both were removed in #85. The Go API owns all database
  access through sqlc, and auth is app-owned end to end.

## Commands

- `bun dev`
- `bun run lint`
- `bun run build`

## Env

`bun run build` and `bun dev` need no environment variables at all. The list
splits by which process reads them, and **nothing bridges the two**:
`.env.local` is loaded by Bun for the Next process, while the Go API reads
`os.Getenv` only, so its values must be exported in the shell that starts it.

Read by **Next**:

- `GO_API_URL` — where `/api/*` is proxied. Origin-only, validated at build
  time. Server-side only; the browser always calls the Next origin.
- `NEXT_PUBLIC_API_URL` — optional origin for the API client. Read only on the
  server (`lib/api-base-url.ts`); in the browser the base is always relative so
  requests stay same-origin. Leaving it unset is correct behind the proxy.

Read by the **Go API**:

- `AUTH_JWT_SECRET` — required, no default. The API exits at startup if it is
  unset or shorter than 32 bytes. Generate with `openssl rand -base64 48`;
  never commit a value.
- `AUTH_COOKIE_SECURE` — `false` locally; must be `true` anywhere deployed,
  which requires HTTPS.
- `DATABASE_URL` — the Go API's connection string. The frontend does not connect
  to Postgres.
- `APP_BASE_URL` — origin used to build links in verification and reset mail;
  origin-only and validated at backend startup.
- `SMTP_ADDR`, `MAIL_FROM` — outgoing mail. `SMTP_ADDR` is one `host:port`.

`AUTH_JWT_SECRET` is the only one with no default, so it is the only one that
must be exported for `go run ./cmd/api`. Reference:
[`.env.example`](.env.example), and the full backend table in
[`backend/README.md`](backend/README.md).

## Repo Rules

- Keep feature code in `features/<domain>/`.
- Call the API through the generated client in `lib/api/`, never ad hoc `fetch`.
  Server behavior lives in the Go API; contract changes land in
  `openapi/nuchi.openapi.json` first, then both sides regenerate.
- Keep server-state logic in TanStack Query hooks.
- The database schema lives in `backend/migrations/` (goose) and is reached
  through sqlc. There is no frontend database access.
- Scope all auth-sensitive reads and writes by `auth.userId`.
- Transaction amounts are stored in milliunits.
- Prefer existing `components/ui/*` primitives.
- Avoid `any` in app code.
- Do not leave debug routes, raw `console.log`, or dead commented code in production paths.

## Current Risk Areas

These described the Hono implementation and were resolved by the Go migration —
ownership predicates plus RLS, strict date and body validation, and rate
limiting all live in the Go API now. The remaining frontend items are now
regression-protected too:

- CSV import rejects parser errors, empty/header-only files, blank internal
  rows, and uneven records before column mapping; row values are strictly typed
  and validated in `lib/transaction-import.ts`.
- Header filters own URL/account state only. A regression test prevents them
  from depending on summary query state.

Known backend limitations are tracked as numbered entries in
`post-migration-improvements/claude-backend-improvements/`, not here.

## Verify

CI gates the frontend job on all four of these, so run all four:

- `bun run lint`
- `bunx tsc --noEmit`
- `bun test`
- `bun run build`

Backend changes: `cd backend && go vet ./...` and `cd backend && go test ./...`.
Schema changes: add a goose migration under `backend/migrations/`, update the
sqlc queries, and regenerate.

## Pull Requests Hard Rules

- Do not prefix Codex-created branch names with `codex/` or include `codex` in the branch name.
- For issue-related PRs, title the PR as: `[Issue - #<number>] <PR title>`.
  Example: `[Issue - #35] Add OpenAPI scaffold and generation commands`

## Reference

- Active backlog: [`PR_REVIEW_TECH_DEBT_CONSOLIDATED.md`](PR_REVIEW_TECH_DEBT_CONSOLIDATED.md)

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost) — but **do not commit `graphify-out/` from a feature branch**. The graph is regenerated wholesale across 22 files, so a committed refresh conflicts with every other open branch the moment any one of them merges. Refresh and commit it on `master` after a merge instead. `.gitattributes` marks the directory `-merge` and `linguist-generated` so those conflicts are trivial to resolve and stay collapsed in PR diffs.
