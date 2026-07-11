# 0004 — Bulk-delete silently ignores missing/unowned IDs

- **Migration ticket:** #40 (queries), #44+ (handlers) (https://github.com/GonzaloSecades/nuchi/issues/40)
- **Area:** api
- **Priority guess:** low

## How it was migrated

`BulkDeleteAccounts` / `BulkDeleteCategories` / `BulkDeleteTransactions`
delete `WHERE id = ANY($ids)` with ownership predicates and return only the
ids actually deleted. IDs that don't exist — or belong to another user — are
silently ignored, and the response contains no acknowledgment of them.
Fixtures: "missing or unowned IDs are ignored and the response contains only
deleted owned IDs."

## Why it was done this way

Parity with the legacy Hono behavior, which the frontend already expects.
It also has a real security virtue: not distinguishing "doesn't exist" from
"not yours" avoids leaking whether an id exists at all.

## The concern

Ergonomics, not security: a client that sends 10 ids and gets 7 back has no
way to know *why* three were skipped (already deleted? typo? never owned?).
For a personal finance app where users bulk-manage transactions, silent
partial success can hide client bugs — a stale UI resubmitting old ids looks
identical to a successful cleanup.

## Proposed improvement

Post-parity, consider an explicit response shape: `deleted: [...]` plus a
count of ignored ids (count only — no per-id reasons, preserving the
existence-leak protection). This is an additive OpenAPI change the frontend
can adopt gradually. Low priority; revisit only if client-side confusion
actually shows up.
