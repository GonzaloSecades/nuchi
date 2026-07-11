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

When work on a migration ticket surfaces something that should be improved —
performance, security, API ergonomics, data modeling — but parity forbids
changing it now:

1. **Port it faithfully anyway.** Fixtures and the OpenAPI contract win.
2. **Record it here** as a numbered entry (`NNNN-short-slug.md`) using the
   template below, in the same PR as the migration work when practical.
3. **Never act on an entry during the migration.** Entries become tickets
   only after #27 (legacy teardown) closes the migration.

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

## Index

| # | Entry | Area | Priority |
| --- | --- | --- | --- |
| 0001 | [transactions.date is timestamp without time zone](0001-transactions-date-timestamp.md) | schema | medium |
| 0002 | [Finance tables use text cuid IDs; UUID default is v4](0002-finance-ids-and-uuidv7-default.md) | schema | low |
| 0003 | [Transaction rate limiting is in-memory](0003-in-memory-rate-limiting.md) | api/infra | medium |
| 0004 | [Bulk-delete silently ignores missing/unowned IDs](0004-bulk-delete-silent-ignore.md) | api | low |
| 0005 | [Category duplicate update returns 500, create returns 409](0005-category-duplicate-update-500.md) | api | high |
