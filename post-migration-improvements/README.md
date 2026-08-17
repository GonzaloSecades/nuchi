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

Every open entry is tracked on the **PostMigration Backend Improvements**
board (https://github.com/users/GonzaloSecades/projects/3), alongside the
phases of the `codex-backend-improvements/` program. A new entry is not
finished when the file is written: open its issue, add it to the board, and
add the row here. The `Priority` column is the priority the entry was
*filed* with — the board's `Priority` field is the one that gets reordered
by evidence, and Phase 5 of the program (#111) is where that reordering
happens.

| # | Entry | Area | Priority | Issue |
| --- | --- | --- | --- | --- |
| 0001 | [transactions.date is timestamp without time zone](claude-backend-improvements/0001-transactions-date-timestamp.md) | schema | medium | #112 |
| 0002 | [Finance tables use text cuid IDs; UUID default is v4](claude-backend-improvements/0002-finance-ids-and-uuidv7-default.md) | schema | low | #113 |
| 0003 | [Transaction rate limiting is in-memory](claude-backend-improvements/0003-in-memory-rate-limiting.md) | api/infra | medium | #114 |
| 0004 | [Bulk-delete silently ignores missing/unowned IDs](claude-backend-improvements/0004-bulk-delete-silent-ignore.md) | api | low | #115 |
| 0005 | [Category duplicate update returns 500, create returns 409](claude-backend-improvements/0005-category-duplicate-update-500.md) — **resolved in #45** | api | high | — |
| 0006 | [transactions.amount is 32-bit, capping a single transaction near ±2.1M ARS](claude-backend-improvements/0006-amount-int32-milliunit-cap.md) — **resolved in #46** | schema | high | — |
| 0007 | [JWT signing is HS256 with a single static secret](claude-backend-improvements/0007-jwt-hs256-single-secret.md) | auth | medium | #116 |
| 0008 | [Access tokens cannot be revoked mid-life](claude-backend-improvements/0008-no-midlife-access-token-revocation.md) | auth | low | #117 |
| 0009 | [No refresh-token reuse detection or session listing](claude-backend-improvements/0009-refresh-reuse-detection-session-listing.md) | auth | low | #118 |
| 0010 | [Auth operations do not declare 500 responses in the contract](claude-backend-improvements/0010-auth-contract-omits-500.md) | api | low | #119 |
| 0011 | [No resend endpoint; email delivery is fire-and-forget](claude-backend-improvements/0011-no-resend-endpoint-fire-and-forget-mail.md) | auth/infra | medium | #120 |
| 0012 | [Residual timing oracle on password-reset request](claude-backend-improvements/0012-reset-request-timing-oracle.md) | auth | low | #121 |
| 0013 | [Resource endpoints accept unbounded request bodies](claude-backend-improvements/0013-no-request-body-size-cap-on-resource-endpoints.md) | http | medium | #122 |
| 0014 | [Date filters are parsed in UTC, not the host timezone](claude-backend-improvements/0014-date-filters-parsed-in-utc.md) | api | low | #123 |
| 0015 | [Out-of-range amounts return 400, where legacy returned 500](claude-backend-improvements/0015-out-of-range-amount-returns-400.md) | api | low | #124 † |
| 0016 | [Bulk body limits are enforced against the stream, not just Content-Length](claude-backend-improvements/0016-bulk-body-limit-enforced-against-the-stream.md) | http | low | #125 † |
| 0017 | [The summary category chart silently excludes uncategorized spending](claude-backend-improvements/0017-summary-category-chart-excludes-uncategorized.md) | api/product | medium | #126 |
| 0018 | [A transaction's date is serialized as a UTC instant, not a calendar date](claude-backend-improvements/0018-date-response-serialized-as-utc.md) | api/schema | medium | #127 |

† 0015 and 0016 both conclude that the Go behavior is the correct one and
nothing should be undone. Their entry titles name the recorded divergence,
as every entry title does; their issues are titled and scoped for the
residual follow-up work instead — contract amount bounds that generate no
validation (#124), and bulk byte limits that live in prose where they can
drift from the handler constants (#125). Read the entry for the history and
the issue for the work.

0001, 0014 and 0018 are three layers of one modelling decision and must land
together. #127 carries the executable plan, including the row audit that has
to run before any `ALTER TABLE`.
