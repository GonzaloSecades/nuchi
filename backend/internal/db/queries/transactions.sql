-- Transactions have no direct user_id column; every query below owns them
-- through their required account (transactions.account_id -> accounts.id,
-- accounts.user_id = the authenticated user). This mirrors both the RLS
-- policy (transactions_owner) and the legacy Drizzle joins in
-- app/api/[[...route]]/transactions.ts.

-- name: ListTransactions :many
-- Matches the legacy list query: join through the owned account for the
-- name, left join category for its (optional) name, date range inclusive on
-- both ends, optional accountId filter. Handler concerns (date
-- defaulting/validation, 366-day cap) live outside this query.
SELECT
    t.id,
    t.date,
    c.name AS category,
    t.category_id,
    t.payee,
    t.amount,
    t.notes,
    a.name AS account,
    t.account_id
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
LEFT JOIN categories c ON c.id = t.category_id AND c.user_id = sqlc.arg(user_id)
WHERE t.date >= sqlc.arg(start_date)
  AND t.date <= sqlc.arg(end_date)
  AND (sqlc.narg(account_id)::text IS NULL OR t.account_id = sqlc.narg(account_id))
ORDER BY t.date DESC;

-- name: GetTransaction :one
-- Legacy GET /:id selects only the transaction's own columns (no joined
-- names) after proving ownership through the account join; the ownership
-- join does not also filter categories.
SELECT t.*
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
WHERE t.id = sqlc.arg(id);

-- name: CreateTransaction :one
-- Ownership of account_id/category_id is validated by the handler with
-- GetAccount/GetCategory before this insert runs (legacy behavior: friendly
-- 400/404 before insert); RLS WITH CHECK is the backstop, not the primary
-- mechanism.
INSERT INTO transactions (id, amount, payee, notes, date, account_id, category_id, currency)
VALUES (
    sqlc.arg(id),
    sqlc.arg(amount),
    sqlc.arg(payee),
    sqlc.narg(notes),
    sqlc.arg(date),
    sqlc.arg(account_id),
    sqlc.narg(category_id),
    sqlc.arg(currency)
)
RETURNING *;

-- name: UpdateTransaction :one
-- Ownership is proven against the transaction's *current* account_id (the
-- EXISTS subquery evaluates against the pre-update row, same as the legacy
-- CTE-scoped update). The handler validates the new account_id/category_id
-- ownership separately before calling this, exactly like CreateTransaction.
UPDATE transactions t
SET amount = sqlc.arg(amount),
    payee = sqlc.arg(payee),
    notes = sqlc.narg(notes),
    date = sqlc.arg(date),
    account_id = sqlc.arg(account_id),
    category_id = sqlc.narg(category_id),
    currency = sqlc.arg(currency)
WHERE t.id = sqlc.arg(id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    WHERE a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
  )
RETURNING t.*;

-- name: DeleteTransaction :one
DELETE FROM transactions t
USING accounts a
WHERE t.id = sqlc.arg(id)
  AND t.account_id = a.id
  AND a.user_id = sqlc.arg(user_id)
RETURNING t.id;

-- name: BulkCreateTransactions :many
-- Single INSERT ... SELECT round trip from one structured jsonb array
-- parameter. A single parameter makes per-row integrity structural: a row
-- either exists in the JSON with all its fields or it does not - there is no
-- multi-array cardinality to keep in sync. NULL notes/categoryId in the JSON
-- land as SQL NULLs natively.
INSERT INTO transactions (id, amount, payee, notes, date, account_id, category_id, currency)
SELECT r.id, r.amount, r.payee, r.notes, r.date, r.account_id, r.category_id, r.currency
FROM jsonb_to_recordset(sqlc.arg(payload)::jsonb) AS r(
    id text,
    amount integer,
    payee text,
    notes text,
    date timestamp,
    account_id text,
    category_id text,
    currency text
)
RETURNING *;

-- name: BulkDeleteTransactions :many
-- Missing or unowned ids are silently ignored: they simply do not appear in
-- the RETURNING set (matches fixtures: bulk-delete never 404s). A second
-- user's row is left untouched because it never satisfies the owned-account
-- join.
DELETE FROM transactions t
USING accounts a
WHERE t.id = ANY(sqlc.arg(ids)::text[])
  AND t.account_id = a.id
  AND a.user_id = sqlc.arg(user_id)
RETURNING t.id;
