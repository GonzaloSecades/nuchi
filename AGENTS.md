# Nuchi Codex Guide

## Stack

- Next.js App Router
- Clerk auth
- Hono API in `app/api/[[...route]]`
- Drizzle ORM + Neon/Postgres
- TanStack Query + typed Hono client in `lib/hono.ts`
- Bun package manager

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
- Prefer extending typed Hono routes over ad hoc `fetch`.
- Keep server-state logic in TanStack Query hooks.
- Use `db/schema.ts` as the DB source of truth; keep migrations in sync.
- Scope all auth-sensitive reads and writes by `auth.userId`.
- Transaction amounts are stored in milliunits.
- Prefer existing `components/ui/*` primitives.
- Avoid `any` in app code.
- Do not leave debug routes, raw `console.log`, or dead commented code in production paths.

## Current Risk Areas

- Validate ownership in transaction create and bulk-create flows.
- Harden CSV import validation, typing, and empty-state handling.
- Keep mutating routes protected; CSRF/rate limiting are still open debt.
- `summary.ts` still needs strict date validation and debug-route removal.
- Avoid adding more coupling between header filters and summary loading.

## Verify

- UI-only: `bun run lint`
- Route/schema changes: `bun run lint` and `bun run build`
- Schema changes: also run `bun run db:generate` or explain why not

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
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
