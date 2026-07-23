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

Queue protocol: migration tickets are titled `[Backend Migration NN]`. The
orchestrator decides the working order from the dependency chain in `spec.md`
(there are no `blocked`/`risk:*` labels — they were retired 2026-07-23 as dead
metadata; see the flow below). Every ticket is attended: Gonzalo reviews the
diff and gives the merge signal, so there is no unattended lane to gate.

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

## Working Flow (streamlined 2026-07-23)

The loop is: **orchestrator briefs → go-migrator develops → review → Gonzalo
merges → close out**. Gonzalo pushes all implementation to Claude, reviews the
diff himself, and gives the merge signal. Because he reviews everything, the
old `risk:*` attended/unattended split and the `blocked` label are gone —
retired as dead metadata. Keep the flow lean; do not re-add ceremony that this
flow removed.

Per ticket:
1. **Brief.** Orchestrator picks the next ticket by dependency order, makes the
   design decisions, and writes the briefing (ticket + spec section + fixtures
   section + Hono reference source + OpenAPI operations). Post it as a ticket
   comment so it survives a cold session.
2. **Develop.** Dispatch `go-migrator` in an isolated worktree with the
   briefing. It implements exactly that ticket and refreshes graphify before
   handing in.
3. **Review.** Run `parity-reviewer` on the diff when the ticket changes API
   behavior (skip it for pure CI/docs/config diffs — it has nothing to check
   there). Address real findings; surface the rest to Gonzalo with reasoning.
   Copilot is best-effort only: request it once via the API, and if it does not
   attach (quota), say so and move on — never stall the flow waiting for it.
4. **Merge (Gonzalo's call).** Never merge without green CI **and** Gonzalo's
   explicit in-session merge signal. GitHub approval cannot substitute — PRs
   are self-authored through his `gh` auth, so GitHub forbids self-approval.
5. **Close out.** Merge commit with a descriptive message; comment verification
   evidence on the ticket (confirm live tests actually ran, not skipped); close
   the ticket; set board status to Done; refresh graphify on master.

Branch/PR hard rules (unchanged):
- Branch names: `claude/<issue-number>-<short-slug>`.
- PR titles: `[Issue - #<number>] <PR title>`.
- Never delete a branch after merge.

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

The legacy Hono implementation has known flaws; migrate first. Parity
freezes **observable behavior** (fixtures + OpenAPI contract), not
implementation quality: internal security/performance hardening that keeps
observable behavior identical (race fixes, overflow-safe SQL, ownership
predicates, indexes) is done in migration PRs. Improvements that would
change observable behavior are ported faithfully anyway and recorded as
numbered entries in `post-migration-improvements/claude-backend-improvements/`
(rule + template in the root README; index there too) — ticket link, how it
was migrated, why, the concern, the proposed improvement. Review notes in
that second lane go to the registry too. The sibling
`codex-backend-improvements/` holds Codex's cross-cutting optimization
program; don't write there. The orchestrator writes entries; implementation
agents flag candidates in handoff notes. Never act on an entry during the
migration; they become tickets for the backend-optimization project
after #27.
