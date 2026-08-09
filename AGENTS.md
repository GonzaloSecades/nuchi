# Nuchi Codex Guide

## Stack

- Next.js App Router (frontend only — there is no Next API layer)
- Go API (chi, pgxpool, sqlc, goose) over PostgreSQL, reached same-origin
  through the `/api/*` rewrite in `next.config.ts`
- Owned JWT auth (`lib/auth/`), not Clerk
- TanStack Query + the generated OpenAPI client in `lib/api/`
- Bun package manager
- Drizzle ORM and the `@clerk/*` packages are still installed but unused by any
  code path; they are removed in #85

## Commands

- `bun dev`
- `bun run lint`
- `bun run build`
- `bun run db:generate`
- `bun run db:migrate`

## Env

- `DATABASE_URL`
- `NEXT_PUBLIC_API_URL`
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `CLERK_SECRET_KEY`
- Reference: [`.env.example`](/home/gonzalo/projects/nuchi/.env.example)

## Repo Rules

- Keep feature code in `features/<domain>/`.
- Call the API through the generated client in `lib/api/`, never ad hoc `fetch`.
  Server behavior lives in the Go API; contract changes land in
  `openapi/nuchi.openapi.json` first, then both sides regenerate.
- Keep server-state logic in TanStack Query hooks.
- Use `db/schema.ts` as the DB source of truth; keep migrations in sync.
- Scope all auth-sensitive reads and writes by `auth.userId`.
- Transaction amounts are stored in milliunits.
- Prefer existing `components/ui/*` primitives.
- Avoid `any` in app code.
- Do not leave debug routes, raw `console.log`, or dead commented code in production paths.

## Current Risk Areas

These described the Hono implementation and were resolved by the Go migration —
ownership predicates plus RLS, strict date and body validation, and rate
limiting all live in the Go API now. Two frontend items remain:

- Harden CSV import validation, typing, and empty-state handling.
- Avoid adding more coupling between header filters and summary loading.

Known backend limitations are tracked as numbered entries in
`post-migration-improvements/claude-backend-improvements/`, not here.

## Verify

CI gates the frontend job on all four of these, so run all four:

- `bun run lint`
- `bunx tsc --noEmit`
- `bun test`
- `bun run build`

Backend changes: `cd backend && go vet ./...` and `cd backend && go test ./...`.
Schema changes: also run `bun run db:generate` or explain why not.

## Pull Requests Hard Rules

- Do not prefix Codex-created branch names with `codex/` or include `codex` in the branch name.
- For issue-related PRs, title the PR as: `[Issue - #<number>] <PR title>`.
  Example: `[Issue - #35] Add OpenAPI scaffold and generation commands`

## Reference

- Active backlog: [`PR_REVIEW_TECH_DEBT_CONSOLIDATED.md`](/home/gonzalo/projects/nuchi/PR_REVIEW_TECH_DEBT_CONSOLIDATED.md)

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost) — but **do not commit `graphify-out/` from a feature branch**. The graph is regenerated wholesale across 22 files, so a committed refresh conflicts with every other open branch the moment any one of them merges. Refresh and commit it on `master` after a merge instead. `.gitattributes` marks the directory `-merge` and `linguist-generated` so those conflicts are trivial to resolve and stay collapsed in PR diffs.
