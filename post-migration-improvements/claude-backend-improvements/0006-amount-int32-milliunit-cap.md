# 0006 — transactions.amount is 32-bit, capping a single transaction near ±2.1M ARS

- **Migration ticket:** #39 (schema), surfaced during #40 (https://github.com/GonzaloSecades/nuchi/issues/40)
- **Area:** schema
- **Priority guess:** high

## How it was migrated

`transactions.amount` is Postgres `integer` (32-bit), exactly as the legacy
Drizzle schema, holding signed milliunits. sqlc therefore generates `int32`
for row-level amounts in Go; summary aggregations (`SUM`) are cast to
`bigint`/`int64`, so *totals* cannot overflow — only individual rows can.

## Why it was done this way

Parity: the legacy column is `integer`, the fixtures freeze existing
behavior, and widening a money column mid-migration is precisely the kind of
"improvement" the port-not-redesign rule defers.

## The concern

A signed 32-bit milliunit value caps a single transaction at
±2,147,483,647 milliunits ≈ **±2,147,483 ARS** (about ±2.1 million pesos).
The default currency is ARS in a high-inflation economy: rents, salaries,
car purchases, and medical bills can plausibly exceed 2.1M ARS today, and
inflation only moves the boundary closer. Overflow behavior differs by
layer: Postgres rejects the insert (`numeric_value_out_of_range`), but any
Go arithmetic performed in `int32` before insert could wrap silently. For a
personal finance app, a transaction that cannot be recorded — or worse,
records wrongly — is a core-product failure, not an edge case.

## Resolved during the migration (#46)

**This entry is closed. Kept for history.**

Widened to `bigint` / Go `int64` in migration `00005`, ahead of the #46
handlers, because implementing transactions against a cap we already knew was
wrong would have meant writing throwaway `int32` bounds checks and tests and
then deleting them.

What forced the timing: at the BCRA reference rate the old ceiling of
2,147,483.647 ARS is roughly **USD 1,400** — below a month's rent, a medical
bill, or a used-car payment in Argentina. That is a product defect, not a
theoretical limit.

Shipped together, because a partial change would have left two different caps
in one API:

- `00005_transactions_amount_bigint.sql` — `ALTER COLUMN amount TYPE bigint`,
  append-only. Its Down deliberately fails if any row no longer fits in int4,
  rather than truncating financial data during a rollback.
- The `jsonb_to_recordset` column list in `BulkCreateTransactions` — left at
  `integer`, bulk-create and the whole CSV import path would have kept the old
  cap while single create no longer had it.
- OpenAPI: every milliunit field (`TransactionInput`, `Transaction`,
  `TransactionListItem`, `SummaryCategory.value`, `SummaryDay.income/expenses`,
  `Summary.*Amount`) is now `format: int64` bounded to ±(2^53−1).
- Drizzle: `bigint('amount', { mode: 'number' })`. Required, not cosmetic —
  node-postgres returns int8 as a *string*, so the legacy stack would have
  started receiving `"12500"` and breaking its arithmetic.
- Frontend: `isSafeMiliunitAmount` guards the transaction form and the CSV
  importer.

The API bound is deliberately **narrower than the column**: ±(2^53−1)
milliunits, JavaScript's safe-integer limit, so every value the API returns is
exact in the browser. Values between that and bigint's range are reachable only
by direct SQL. Do not "fix" the validator to match the column.

Note the OpenAPI `minimum`/`maximum` are documentation only — oapi-codegen
emits no range validation — so the handler must check the bound explicitly.

## Original proposed improvement (superseded)

Widen the column to `bigint` (`ALTER TABLE transactions ALTER COLUMN amount
TYPE bigint` — cheap table rewrite at personal-app scale) and regenerate
sqlc so Go uses `int64` end to end. The OpenAPI contract's
`amount: integer` is JSON — JSON integers are not 32-bit-bounded, so the
contract likely needs only a `format: int64` annotation, but the frontend's
number handling must be audited (JS safe-integer bound is 2^53, far beyond
any realistic amount). Coordinate with fixtures regeneration post-parity.
Candidate for the **first** optimization ticket alongside 0005: it is the
only registry entry where the current design can lose or reject user money
data outright.
