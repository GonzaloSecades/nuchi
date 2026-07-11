# Nuchi — Claude Code Guide

## Project

Personal finance app mid-migration from Next.js/Hono/Drizzle/Neon/Clerk to a
separate Go API (chi, pgxpool, sqlc, goose) with Dockerized PostgreSQL, owned
JWT auth, and RLS. Port-not-redesign: freeze behavior via fixtures, swap the
technology, refactor only after parity.

Source of truth:

- Spec + child issue plan: `docs/specs/18-go-backend-replacement/spec.md`
- Behavioral oracle: `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`
- API contract: `openapi/nuchi.openapi.json` (OpenAPI-first: contract changes
  land there first, then both sides regenerate)
- Board: https://github.com/users/GonzaloSecades/projects/1

Queue protocol: migration tickets are titled `[Backend Migration NN]`. Work the
lowest open NN not labeled `blocked`. Only `risk:low` tickets may run
unattended; `risk:high` tickets are attended work with human merge review.

## Commands

- Frontend: `bun run lint`, `bun run build`, `bun test`
- Backend (run from `backend/`): `cd backend && go test ./...`,
  `cd backend && go vet ./...`, `cd backend && go run ./cmd/api`
- Contract: `bun run openapi:validate`, `bun run openapi:gen:go`, `bun run openapi:gen:ts`
- Services: `docker compose up -d postgres` (Mailpit included in dev compose)
- Graph: `graphify update .` after code changes (AST-only, no API cost)

## Hard Invariants

- Transaction amounts are signed integer milliunits. Never floats for money.
- App resource success responses use the `{ "data": ... }` envelope.
- Default transaction currency is `ARS`; currency is required on transactions.
- Auth-sensitive reads/writes derive identity from the verified token, never
  from request body fields.
- RLS is the security backstop; SQL still includes ownership predicates.
- Generated code (`backend/internal/openapi/`, `lib/api/generated/`) is never
  hand-edited.

## Risk Policy

`risk:high` (attended only): money math, SQL migrations, auth, RLS policies,
bulk import, secrets, data deletion, production deploy changes.
`risk:low` (unattended after calibration): docs, isolated tests, UI copy,
dev scripts, CI config, codegen.

## Branch And PR Hard Rules

- Branch names: `claude/<issue-number>-<short-slug>`.
- PR titles: `[Issue - #<number>] <PR title>`.
- Never delete a branch after merge.
- Never merge without green CI and Gonzalo's explicit merge instruction (in
  session or as a PR comment). GitHub review approval cannot be the signal:
  Claude works through Gonzalo's `gh` auth, so PRs are self-authored and
  GitHub forbids self-approval.
- Copilot is the default first reviewer. Its comments are processed by the
  `pr-review-cycle` skill: address medium/high findings that are on point,
  reply to the rest with reasoning, push, let Copilot re-review. Hard cap of
  3 automated iterations per PR; after that, only Gonzalo re-triggers.
- On Gonzalo's approval: merge with a merge commit and a descriptive message,
  comment verification evidence on the ticket, close the ticket, set its board
  status to Done, unblock the next queue ticket, and refresh graphify on
  master.

## Model Orchestration

- Main session (Fable) is the tech lead/PM: picks tickets, writes briefings
  (ticket + spec section + fixtures section + Hono reference source + OpenAPI
  operations), makes design decisions, gates reviews and merges.
- `.claude/agents/go-migrator.md` (Sonnet) implements one ticket per dispatch,
  in an isolated worktree.
- `.claude/agents/researcher.md` (Haiku) does web/docs lookups.
- `.claude/agents/parity-reviewer.md` reviews diffs against fixtures before
  human review.

## Graphify

`graphify-out/` is the repo knowledge graph. For codebase questions run
`graphify query "<question>"` first; use `graphify path`/`explain` for
relationships and concepts. Dirty `graphify-out/` files are expected and not a
reason to skip it. Refresh with `graphify update .` after code changes and
before handing in a ticket. Full rules: `AGENTS.md` (graphify section).

## Legacy Code

Existing Hono routes (`app/api/[[...route]]`), Drizzle schema (`db/`), and
Clerk usage are reference material for parity work. Do not extend them with
new features; they are removed in #27 after Go parity.

## Post-Migration Improvements

When migration work surfaces something that should be improved (performance,
security, ergonomics, data modeling) but parity forbids changing it now:
port it faithfully anyway, then record it as a numbered entry in
`post-migration-improvements/` (template in its README) — ticket link, how
it was migrated, why, the concern, the proposed improvement. The orchestrator
writes entries; implementation agents flag candidates in handoff notes.
Never act on an entry during the migration; they become tickets for the
backend-optimization project after #27.
