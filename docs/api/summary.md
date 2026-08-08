# Summary API — Technical Reference

Go implementation reference for `GET /api/summary`, the dashboard's single
aggregate endpoint. Companion to the contract
(`openapi/nuchi.openapi.json`), which is authoritative for shapes and status
codes; this covers what the contract cannot express.

Shipped in [#48](https://github.com/GonzaloSecades/nuchi/issues/48).

## The operation

| Method | Path | Operation |
| --- | --- | --- |
| GET | `/api/summary` | `getSummary` |

Mounted inside the `RequireAuth` group. Read-only, so no rate limiting — the
mutation limiter covers writes only.

Takes the **same three query parameters as the transaction list** —
`from`, `to`, `accountId` — with identical semantics. That is not a
coincidence to preserve loosely: the dashboard filters both views from one
control, so a difference between them would show up as the list and the chart
disagreeing about what period is on screen.

## Date handling is shared, not reimplemented

`parseDateRange` (`daterange.go`) is used verbatim. Defaults, the
"provided `to` does not re-anchor the default `from`" rule, start/end-of-day
boundaries, the 366-day cap, UTC pinning, and the three exact `INVALID_QUERY`
messages all come from there. See the [transactions
reference](transactions.md) for the full rules.

`accountId` goes through `accountIDFilter`, also shared. An explicitly empty
`?accountId=` is a 400, not "no filter".

## Four queries, one transaction

`GetPeriodTotals` runs twice — current window and comparison window — alongside
`GetCategorySpending` and `GetDailyTotals`, all inside a single `WithUserTx`.

The handler requests PostgreSQL `REPEATABLE READ`, rather than relying on the
default `READ COMMITTED` transaction. Two reasons this matters beyond tidiness:

- One RLS binding covers all four.
- All four see **one snapshot**. Legacy fires them independently, so a write
  landing mid-request can produce totals that disagree with the endpoint's own
  daily series.

## The comparison period

Length-based, not calendar-based:

```
periodLength  = calendarDays(end, start) + 1
previousStart = start - periodLength days
previousEnd   = end   - periodLength days
```

A 30-day view compares against the 30 days immediately before it — **not** the
previous calendar month. A March view does not compare against February. This
reads as wrong at first glance and is a faithful port; legacy computes it the
same way with `differenceInCalendarDays` and `subDays`.

## Percentage change

`percentageChange` ports `calculatePercentageChange` from `lib/utils.ts`:

| previous | current | result |
| --- | --- | --- |
| 0 | 0 | `0` |
| 0 | anything else | `100` |
| non-zero | any | `(current − previous) / previous × 100` |

The zero rule looks like a divide-by-zero guard and is a product decision:
"everything is new growth". Both branches are deliberate and both are tested.

This is the **only float in the money path**. It is a ratio, not an amount —
amounts remain integer milliunits everywhere. The division is done in `float64`
and narrowed once at the end; an integer division here would silently truncate
every sub-1% change to zero.

## Category breakdown

The query returns expense totals per category, ordered by value descending. The
handler takes the top three as-is and folds everything past them into a single
`Other` entry.

- Exactly three categories → **no** `Other` row. Adding an empty one would be a
  visible defect.
- `Other` sums every remaining category, not just the fourth.

### What the breakdown excludes

Two exclusions, both inherited from legacy's `innerJoin`:

- **Income is never included** — the query filters `t.amount < 0`.
- **Uncategorized expenses are excluded**, while still counted in
  `expensesAmount`.

So **the chart does not sum to the expenses total** when the user has
uncategorized spending, and nothing in the response explains the gap. Ported
deliberately; recorded as improvement 0017 with the product options.

## Daily series

`fillMissingDays` returns one entry per calendar day in the inclusive range,
using queried totals where a day has activity and zeros where it does not. The
chart needs a continuous series — a gap would otherwise be drawn as a straight
line between active days, implying activity that did not happen.

Keyed by `yyyy-MM-dd` in UTC, matching the query's `GROUP BY t.date::date`.
That cast is load-bearing: grouping by the raw timestamp would split one
calendar day into several rows if any write path ever stored a time of day, and
this map would then keep only the last of them.

Iteration runs over **day boundaries**, not the raw range, because `end` is the
last instant of its day — iterating on raw timestamps would drop the final day.

## Empty state

A period with no transactions returns zeros for all three amounts, zeros for
all three changes, `categories: []`, and a **fully populated** `days` array of
zero entries. Both arrays serialize as `[]`, never `null`; the dashboard maps
over them.

An `accountId` the caller does not own yields the same zeros rather than a 404 —
the ownership join simply matches nothing, and confirming the account exists
would leak another user's data.

## Where the code lives

| Concern | File |
| --- | --- |
| Handler, percentage change, bucketing, day filling | `backend/internal/http/summary.go` |
| Date range semantics | `backend/internal/http/daterange.go` |
| Shared `accountId` filter | `backend/internal/http/resources.go` |
| Queries | `backend/internal/db/queries/summary.sql` |

## Divergences and gaps

| Entry | Note |
| --- | --- |
| 0014 | Date filters parsed in UTC rather than the host timezone (shared with transactions) |
| 0017 | Category chart excludes uncategorized spending while the total includes it |

Individual amounts fit the contract, but a sufficiently large aggregate can
still exceed PostgreSQL `bigint` or the stricter JavaScript-safe range declared
for summary fields. Resolving that requires an explicit response-contract
decision; it is tracked in
[`Codex-summary-aggregate-contract-range.md`](../../post-migration-improvements/codex-backend-improvements/Codex-summary-aggregate-contract-range.md)
rather than silently clamping or changing parity behavior in this migration.

No pagination or caching: the endpoint recomputes everything per request. Fine
at personal-finance scale, and the first thing to revisit if a user's history
grows large.
