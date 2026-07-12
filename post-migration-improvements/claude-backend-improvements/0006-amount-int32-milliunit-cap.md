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

## Proposed improvement

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
