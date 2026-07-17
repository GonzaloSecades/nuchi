# Post-Migration Improvements

A registry of improvements **deliberately not made** during the Go backend
migration (#18). The migration rule is port-not-redesign: legacy Hono
behavior is frozen by the parity fixtures and copied faithfully, even where
we know a better design. This directory is where "better" goes to wait.

Each entry is the seed of a future ticket in the backend-optimization
project that follows the migration. Entries are written for Claude Code and
humans alike: enough context to open a ticket and start work without
re-deriving the history.

## The rule

The legacy Hono implementation has known flaws, errors, and room for
improvement. That is expected and is not the migration's problem to solve:
the flow is **migrate first**. What parity freezes is **observable
behavior** — response shapes, status codes, error bodies, filtering/ordering
semantics, everything the fixtures and the OpenAPI contract pin down.

That gives improvements two lanes:

- **Internal hardening — do it during the migration.** Security and
  performance improvements that leave observable behavior identical are
  allowed and expected in migration PRs: closing race conditions (atomic
  token consume), overflow-safe SQL, explicit ownership predicates,
  better indexes, single-round-trip queries. Review findings of this kind
  get fixed in the PR, not deferred.
- **Behavior-visible improvements — record them here, don't do them.**
  Anything that would change what a client observes (status codes, response
  fields, semantics like bulk-delete's silent ignore, schema types that leak
  into serialization):

  1. **Port it faithfully anyway.** Fixtures and the OpenAPI contract win.
  2. **Record it here** as a numbered entry (`NNNN-short-slug.md`) using the
     template below, in the same PR as the migration work when practical.
  3. **Never act on an entry during the migration.** Entries become tickets
     only after #27 (legacy teardown) closes the migration.

Review notes and future-improvement ideas that surface in PR reviews belong
in this directory too when they fall in the second lane — a review comment
is not a license to change frozen behavior.

The orchestrator (main session) owns writing entries; implementation agents
flag candidates in their handoff notes.

## Entry template

```markdown
# NNNN — <title>

- **Migration ticket:** #NN (<link>)
- **Area:** <schema | queries | auth | api | infra>
- **Priority guess:** <high | medium | low> (perf/security first)

## How it was migrated

<What shipped, precisely.>

## Why it was done this way

<The parity/contract/fixture constraint that forced it.>

## The concern

<What is suboptimal — performance, security, correctness edge, ergonomics.>

## Proposed improvement

<Concrete change, expected benefit, and any migration/compat cost.>
```

## Layout

This README (the rule + entry template) stays at the root. The content lives
in two agent-owned subdirectories:

- [`claude-backend-improvements/`](claude-backend-improvements/) — Claude's
  numbered registry entries (`NNNN-short-slug.md`, template above). All
  future entries from Claude's migration reviews are written here.
- [`codex-backend-improvements/`](codex-backend-improvements/README.md) —
  Codex's cross-cutting backend-optimization program (security, performance,
  robustness, documentation, observability-readiness per module).

The numbered entries remain the registry for behavior-visible parity
deviations; the project directory defines the architecture and delivery
gates. Numbering stays global and sequential within
`claude-backend-improvements/`.

## Index (claude-backend-improvements/)

| # | Entry | Area | Priority |
| --- | --- | --- | --- |
| 0001 | [transactions.date is timestamp without time zone](claude-backend-improvements/0001-transactions-date-timestamp.md) | schema | medium |
| 0002 | [Finance tables use text cuid IDs; UUID default is v4](claude-backend-improvements/0002-finance-ids-and-uuidv7-default.md) | schema | low |
| 0003 | [Transaction rate limiting is in-memory](claude-backend-improvements/0003-in-memory-rate-limiting.md) | api/infra | medium |
| 0004 | [Bulk-delete silently ignores missing/unowned IDs](claude-backend-improvements/0004-bulk-delete-silent-ignore.md) | api | low |
| 0005 | [Category duplicate update returns 500, create returns 409](claude-backend-improvements/0005-category-duplicate-update-500.md) | api | high |
| 0006 | [transactions.amount is 32-bit, capping a single transaction near ±2.1M ARS](claude-backend-improvements/0006-amount-int32-milliunit-cap.md) | schema | high |
| 0007 | [JWT signing is HS256 with a single static secret](claude-backend-improvements/0007-jwt-hs256-single-secret.md) | auth | medium |
| 0008 | [Access tokens cannot be revoked mid-life](claude-backend-improvements/0008-no-midlife-access-token-revocation.md) | auth | low |
| 0009 | [No refresh-token reuse detection or session listing](claude-backend-improvements/0009-refresh-reuse-detection-session-listing.md) | auth | low |
| 0010 | [Auth operations do not declare 500 responses in the contract](claude-backend-improvements/0010-auth-contract-omits-500.md) | api | low |
