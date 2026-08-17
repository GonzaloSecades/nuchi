# Nuchi — Claude Code Guide

## Project

Personal finance app migrated from Next.js/Hono/Drizzle/Neon/Clerk to a Next
frontend plus a separate Go API (chi, pgxpool, sqlc, goose) over Dockerized
PostgreSQL, with owned JWT auth and RLS. The migration was port-not-redesign:
behavior frozen by fixtures, technology swapped, refactoring only after parity.

The legacy stack is gone — Hono in #84, Drizzle/Neon/Clerk in #85, and the
differential parity harness plus `pg` in #90. #27 closed on 2026-08-10, so the
migration itself is complete; what remains is ordinary work on the Go stack.

Source of truth:

- API contract: `openapi/nuchi.openapi.json` (OpenAPI-first: contract changes
  land there first, then both sides regenerate). This is the behavioral oracle
  now that the migration has closed.
- What shipped, and why: `docs/specs/18-go-backend-replacement/spec.md` and
  `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`. These describe
  the behavior the Go API was ported to. Read them to learn what current
  behavior is and where it came from; they no longer veto a ticket that
  deliberately changes it.
- Board: https://github.com/users/GonzaloSecades/projects/1

Queue protocol: the active backlog is the post-migration improvement tickets,
titled `[NNNN] <description>` after their registry entry in
`post-migration-improvements/claude-backend-improvements/`. There is no
dependency chain to walk anymore — the orchestrator picks by priority, with
security and correctness ahead of ergonomics. There are no `blocked`/`risk:*`
labels; they were retired 2026-07-23 as dead metadata. Every ticket is
attended: Gonzalo reviews the diff and gives the merge signal, so there is no
unattended lane to gate.

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

## Working Flow (streamlined 2026-07-23, generalized past the migration 2026-08-17)

The loop is: **orchestrator briefs → go-migrator develops → review → Gonzalo
merges → close out**. Gonzalo pushes all implementation to Claude, reviews the
diff himself, and gives the merge signal. Because he reviews everything, the
old `risk:*` attended/unattended split and the `blocked` label are gone —
retired as dead metadata. Keep the flow lean; do not re-add ceremony that this
flow removed.

Per ticket:
1. **Brief.** Orchestrator picks the next ticket by priority, makes the design
   decisions, and writes the briefing (ticket + OpenAPI operations in scope +
   the intended behavior change, if any, stated explicitly). Post it as a
   ticket comment so it survives a cold session.
2. **Develop.** Dispatch `go-migrator` in an isolated worktree with the
   briefing. It implements exactly that ticket and refreshes graphify before
   handing in.
3. **Review.** Run `parity-reviewer` on the diff when the ticket changes API
   behavior (skip it for pure CI/docs/config diffs — it has nothing to check
   there). It checks that every observable change is one the ticket asked for
   and reached the contract, not that behavior stayed frozen. Address real
   findings; surface the rest to Gonzalo with reasoning. Copilot is best-effort
   only: request it once via `gh pr edit <n> --add-reviewer "@copilot"`, and if
   it does not attach (quota), say so and move on — never stall the flow
   waiting for it.
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
  (ticket + OpenAPI operations + the intended behavior change), makes design
  decisions, gates reviews and merges.
- `.claude/agents/go-migrator.md` (Sonnet) implements one ticket per dispatch,
  in an isolated worktree. The name is historical — it implements ordinary
  backend tickets now, not migration ones.
- `.claude/agents/researcher.md` (Haiku) does web/docs lookups.
- `.claude/agents/parity-reviewer.md` reviews diffs against the OpenAPI
  contract before human review, checking that behavior changes are intended
  and declared. Also historical in name.

## Graphify

`graphify-out/` is the repo knowledge graph. For codebase questions run
`graphify query "<question>"` first; use `graphify path`/`explain` for
relationships and concepts. Dirty `graphify-out/` files are expected and not a
reason to skip it. Full rules: `AGENTS.md` (graphify section).

**Do not commit `graphify-out/` in a feature branch** (changed 2026-08-09).
The graph is regenerated wholesale, so every branch that refreshed it conflicted
with every other one as soon as any of them merged — with several streams open
at once that is a conflict per branch per merge, in generated JSON nobody reads.
Refresh it **on `master` after a merge** instead, as its own commit:

```
git checkout master && git pull --ff-only
graphify update .
git add graphify-out/ && git commit -m "Refresh graphify artifacts after #NN merge"
```

Reading the graph is unchanged and still expected. This is only about who
commits the regenerated output.

## Legacy Code

The Hono routes (`app/api/[[...route]]`) and the typed client (`lib/hono.ts`)
were **deleted in #84**. Git history is the reference now — do not reintroduce
them, and do not look for them on disk.

The Drizzle schema (`db/`), the `drizzle/` migrations, the Clerk packages, and
the `scripts/migrate.ts` / `scripts/seed.ts` helpers were **deleted in #85**.
The schema lives in `backend/migrations/` under goose now.

The differential parity harness (`tests/parity/**`) and the `pg` dependency it
needed were **deleted in #90**, the last teardown ticket. Nothing from the
legacy stack is on disk anymore.

Historical Hono references are deliberately retained in the migration spec,
`api-parity-fixtures.md`, and the improvement registry — those record what the
migration ported from and should not be rewritten.

## Post-Migration Improvements

`post-migration-improvements/claude-backend-improvements/` is the registry of
improvements deliberately deferred during the migration (rule + template +
index in the root README). Each entry records the ticket link, how the behavior
was migrated, why, the concern, and the proposed improvement.

**The deferral is over.** #27 closed on 2026-08-10, so entries are no longer
frozen — most are now open tickets titled `[NNNN]`, and working one is ordinary
work. Two things carry over from the old rule:

- Entries stay in the registry as the written history of *why* the behavior is
  what it is. Read the entry before working its ticket; don't delete it when
  the ticket closes.
- A change that alters observable behavior still lands contract-first: OpenAPI
  operation, then both sides regenerate, then the affected fixture entries
  updated in the same PR.

New improvement ideas that surface in review still get written up as registry
entries rather than acted on inline, so they get triaged instead of expanding
someone else's PR. The orchestrator writes entries; implementation agents flag
candidates in handoff notes. The sibling `codex-backend-improvements/` holds
Codex's cross-cutting optimization program; don't write there.
