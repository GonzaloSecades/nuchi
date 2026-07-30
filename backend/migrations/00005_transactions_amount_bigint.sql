-- +goose Up
-- +goose StatementBegin
-- Widen transactions.amount from integer (int4) to bigint (int8).
--
-- int4 milliunits cap a single transaction at 2,147,483.647 ARS. At the
-- BCRA reference rate that is roughly USD 1,400 — below a month's rent, a
-- medical bill, or a used-car payment, so the cap is a product defect for
-- an Argentine finance app rather than a theoretical limit.
--
-- The API validates a narrower range than this column accepts:
-- ±(2^53 - 1) milliunits, JavaScript's safe-integer limit, so every value
-- the API returns is exact in the browser. bigint's full range is wider;
-- values between the two are only reachable by direct SQL. Do not "fix"
-- the validator to match the column.
--
-- USING is redundant (int4 -> int8 is an implicit assignment cast) but kept
-- explicit about what happens to existing rows: values are preserved,
-- sign included.
ALTER TABLE transactions
ALTER COLUMN amount TYPE bigint
USING amount::bigint;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Deliberately fails if any row no longer fits in int4.
--
-- PostgreSQL raises "integer out of range" rather than truncating, which is
-- the behavior we want: silently wrapping a 3,000,000.000 ARS income into a
-- negative expense would corrupt financial data during a rollback. To roll
-- back past this migration, first reconcile or remove the out-of-range rows.
ALTER TABLE transactions
ALTER COLUMN amount TYPE integer
USING amount::integer;
-- +goose StatementEnd
