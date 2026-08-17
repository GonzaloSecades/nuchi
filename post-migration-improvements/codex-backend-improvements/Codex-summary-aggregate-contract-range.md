# Codex — Summary Aggregate Contract Range Proposal

- **Modules/operations:** summary queries, `GET /api/summary`
- **Owner:** summary/API contract
- **Priority:** P2
- **Parent registry entry:** none; discovered during migration ticket #48
- **Phase 5 issue:** #111
- **Status:** implementation-ready proposal; approval still required before a
  behavior/contract change

## Phase 5 scope boundary

This proposal is the only behavior-evolution design owned by #111. The phase
does not implement or redesign the overlapping registry work already owned by
Claude:

- calendar-date bundle: [#112](https://github.com/GonzaloSecades/nuchi/issues/112),
  [#123](https://github.com/GonzaloSecades/nuchi/issues/123), and
  [#127](https://github.com/GonzaloSecades/nuchi/issues/127);
- UUIDv7 convergence: [#113](https://github.com/GonzaloSecades/nuchi/issues/113);
- distributed rate limiting: [#114](https://github.com/GonzaloSecades/nuchi/issues/114);
- bulk-delete ignored counts: [#115](https://github.com/GonzaloSecades/nuchi/issues/115); and
- uncategorized summary spending: [#126](https://github.com/GonzaloSecades/nuchi/issues/126).

Those checklist items are gates on their named tickets, not deliverables in
this proposal.

## Problem and measured boundary

Each transaction amount is an exact signed integer milliunit bounded to
JavaScript's safe range:

```text
-9,007,199,254,740,991 through 9,007,199,254,740,991
```

A summary adds an unbounded number of otherwise valid rows. PostgreSQL
`SUM(bigint)` returns `numeric`, but every current summary query casts the
result back to `bigint`; sqlc and the handler then use `int64`. Two distinct
boundaries therefore exist:

1. An aggregate outside the JavaScript-safe range contradicts the published
   response contract even while it still fits Go/PostgreSQL `int64`.
2. A larger aggregate fails the SQL `::bigint` cast and currently becomes the
   generic database 500.

The policy must cover current and previous-period totals, income, absolute
expenses, each category total, the `Other` bucket, and each daily total. A
safe `remainingAmount` does not make the response safe if large income and
expense values cancelled each other.

## Options considered

| Option                               | Compatibility                                             | Operational/data cost                                                                       | Decision                                                 |
| ------------------------------------ | --------------------------------------------------------- | ------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Decimal strings for every aggregate  | Breaking type change for every client formatter           | Exact and migration-free, but forces string arithmetic through the UI                       | Defer until aggregates need to exceed the current domain |
| Structured money object              | Largest breaking response rewrite                         | Useful for future multi-currency, but unrelated to this boundary                            | Reject for this ticket                                   |
| Smaller write-time user/period limit | Preserves response type                                   | Hard to define, expensive under concurrent writes, and rejects otherwise valid transactions | Reject                                                   |
| Explicit aggregate-range error       | Additive non-success response; success shape stays stable | Small OpenAPI/SQL/handler change, exact and fail-closed                                     | **Recommend**                                            |

Clamping, saturation, wrapping, floating-point amounts, or emitting unsafe JSON
numbers are not viable: each silently changes money.

## Recommended contract

Add a `422` response to `getSummary` using the shared API error envelope:

```json
{
  "error": {
    "code": "SUMMARY_AGGREGATE_OUT_OF_RANGE",
    "message": "The selected summary is too large to represent exactly. Choose a shorter date range or one account."
  }
}
```

The response is all-or-nothing. It must not return a partial summary, identify
which account/category crossed the limit, or include the numeric aggregate.
There is no `Retry-After`: an identical retry is deterministic until the
filter or underlying transactions change. Clients may narrow `from`/`to` or
select one owned `accountId`.

`422` distinguishes a valid query whose result cannot satisfy the response
domain from malformed filters (`400`), missing auth (`401`), and unexpected
dependency failure (`500`). The existing `200` schema and integer milliunit
representation do not change.

### OpenAPI sketch

The implementation ticket must edit `openapi/nuchi.openapi.json` first and add:

- `422` on `getSummary`;
- a reusable `SummaryAggregateRangeError` response referencing
  `ApiErrorResponse`; and
- the exact code/message example above.

Both generated clients must be regenerated in the same PR. This proposal does
not edit generated artifacts or claim the decision is already approved.

## Implementation outline

1. Remove the `::bigint` casts from summary aggregate expressions so PostgreSQL
   returns exact `numeric` values instead of overflowing before policy code can
   classify them.
2. Generate sqlc result types and compare every aggregate against
   ±9,007,199,254,740,991 before converting to `int64`.
3. Perform the category `Other` addition in exact numeric arithmetic and check
   the combined value as well as each returned category.
4. Validate previous-period totals before calculating percentages; an unsafe
   baseline makes the complete response unrepresentable even when its amount
   is not returned directly.
5. Map the classified range condition to the proposed 422. Keep unrelated SQL,
   connection, and scan failures on the existing safe 500 path.

No schema/data migration is required. Transaction rows and their write-time
range remain unchanged.

## Compatibility, rollout, and rollback

- **Compatibility:** additive for successful clients; the 200 body is byte-for-
  byte unchanged. Clients that assume every non-400/401 response is a server
  failure should add the new code before rollout.
- **Rollout:** deploy generated client handling first or in the same release,
  then deploy the server. Count the new error by stable code only—never attach
  user, account, category, or aggregate values.
- **Rollback:** reverting the server restores the prior behavior, including a
  possible generic 500 at the wider `int64` boundary. There is no data rollback.
  Keep the contract response until every server is rolled back, then remove it
  only in a separately reviewed compatibility change.

## Acceptance tests

- exact positive and negative JavaScript-safe boundaries succeed;
- boundary +1 and boundary −1 return 422 without a partial response;
- income/expense overflow is caught even when signed remaining cancels to a
  safe value;
- previous-period-only overflow returns 422 before percentage calculation;
- category, combined `Other`, and per-day overflow each return 422;
- empty data and ordinary maximum-range summaries retain their current shape;
- an unowned `accountId` remains a non-disclosing zero summary rather than an
  error; and
- generated Go/TypeScript types and frontend formatting never pass an unsafe
  integer through JavaScript `number`.

## Approval record

- **Recommendation:** explicit 422 aggregate-range error
- **Decision/date:** pending product/API approval
- **Implementation ticket:** create after approval; do not fold into #111
- **Approvers:** repository owner plus frontend/API owner
