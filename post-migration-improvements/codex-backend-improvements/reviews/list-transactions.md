# Endpoint Review — `listTransactions`

## Identity

- **Method/path:** `GET /transactions`
- **Module/owner:** transactions/API, Phase 3 #109
- **OpenAPI operation ID:** `listTransactions`
- **Implementation links:** `internal/http/transactions.go`,
  `internal/http/transaction_cursor.go`, `internal/db/queries/transactions.sql`
- **Observable behavior change:** additive `limit`, `cursor`, and optional
  `nextCursor`; omission preserves the prior response

## Contract

- Inclusive `from`/`to` (maximum 366 days), optional owned `accountId`,
  deterministic `(date DESC,id DESC)` ordering, `{data,nextCursor?}` envelope.
- Existing `400/401/500` errors remain; invalid pagination is `400 INVALID_QUERY`.
- Page limit is 1-500. The endpoint has no request body or mutation rate limit.
- Read-only and client-retry safe. Cursor values are opaque and must be reused
  with the same filters.

## Security

- Bearer JWT creates the typed principal; ownership is derived only from it.
- SQL joins accounts on authenticated `user_id`; forced RLS is the backstop.
- Existing cross-user list tests plus the RLS/runtime-role suites cover
  non-disclosure. Cursor data contains only a date and opaque transaction ID;
  changing filters cannot grant access to a row.
- Financial response data remains sensitive and server errors remain redacted.

## Performance

- Objective: bounded-page warm p95 <=500 ms; compatibility-list p95 <=1 s on
  baseline-v1 before production load objectives exist.
- One statement for min/typical/max inputs; `limit` requests at most 501 rows.
- Existing index: `(account_id,date DESC)`. Candidate: append `id DESC` after
  concurrent schema branches land and are remeasured.
- Baseline-v1 first page (`limit=100`): 50,142 qualifying rows scanned, 101
  returned by SQL, 37 kB top-N sort, 673 ms. This misses the page objective and
  is an accepted temporary query-cost risk; it bounds response bytes, not scan
  work.
- Request context reaches pgx/sqlc. Pool saturation still requires #66's
  production-shaped environment.

## Robustness

- Read committed inside the mandatory RLS-bound transaction.
- No mutation, retry, uncertain commit, or external side effect.
- Unit tests cover cursor round trips and invalid limits/cursors; the live test
  proves page concatenation exactly equals the compatibility list, including
  equal-date tie ordering.

## Documentation and observability readiness

- OpenAPI validation and Go/TypeScript regeneration are PR gates.
- Stable operation name remains `listTransactions`; filters/page size are
  bounded telemetry candidates, while cursor and resource IDs must not be
  labels.
- Request context and chi request ID continue through the existing pipeline.

## Decision

- **Status:** risk accepted for additive rollout; query-plan changes required
  before making pagination the compatibility default
- **Evidence links:** Phase 0 `performance-baseline-v1`, Phase 3 section in
  `03-performance-and-queries.md`, live pagination test
- **Accepted risks:** transactions/API owns the all-account scan; current
  unbounded compatibility mode is the compensating client-stability control;
  expires 2026-10-01 under #109
- **Reviewer/date:** backend optimization program, 2026-08-17
