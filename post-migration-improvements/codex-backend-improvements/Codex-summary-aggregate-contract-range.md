# Codex — Summary Aggregate Contract Range

- **Modules/operations:** summary queries, `GET /api/summary`
- **Owner:** summary/API contract
- **Priority:** P2
- **Parent registry entry:** none; discovered during migration ticket #48
- **Target milestone:** post-migration Phase 2 — monetary correctness hardening

## Problem and evidence

Each transaction amount is bounded to JavaScript's safe integer range, but a
summary adds an unbounded number of rows. The OpenAPI summary amount fields use
that same safe range, while the SQL currently casts PostgreSQL's wider
`SUM(bigint)` result back to `bigint`.

Consequently, individually valid rows can produce an aggregate that does not
fit either the API contract or, at a higher threshold, PostgreSQL `bigint`.
There is no honest in-place parity fix: clamping returns incorrect money,
floating-point output loses milliunit precision, and silently widening the JSON
number violates the published safe-integer bound.

## Required decision

Choose and document one representation for aggregates beyond the current
range:

1. decimal strings with generated client helpers;
2. a structured money value carrying a decimal amount and currency;
3. a deliberately smaller per-user/per-period domain limit enforced at write
   time and transactionally safe under concurrent mutations; or
4. an explicit aggregate-range error response added to the contract.

Do not use saturation or wrapping arithmetic. Until this decision lands, the
migration retains existing response shapes and records the limitation rather
than claiming the aggregate cannot overflow.

## Verification

- Boundary tests cover the largest representable positive and negative totals.
- Bulk and concurrent writes cannot bypass any chosen domain limit.
- Generated Go and TypeScript types match the selected representation.
- Frontend currency formatting never passes an unsafe integer through a
  JavaScript `number`.
- Summary totals, categories, daily values, and percentage baselines follow the
  same range policy.

## Decision record

- **Status:** proposed
- **Decision/date:** pending
- **Approvers:** pending
- **Follow-up tickets:** create after the Go migration and legacy teardown are
  complete
