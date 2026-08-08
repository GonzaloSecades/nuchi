-- name: ListAccounts :many
-- Legacy accounts.ts has no explicit ORDER BY (list order is whatever
-- Postgres returns). ORDER BY name is added here for deterministic output;
-- flagged as a behavior-preserving addition, not a fixture divergence, since
-- the fixture response shape does not depend on ordering.
SELECT *
FROM accounts
WHERE user_id = sqlc.arg(user_id)
ORDER BY name;

-- name: GetAccount :one
SELECT *
FROM accounts
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id);

-- name: CreateAccount :one
-- Duplicate (user_id, name) surfaces as Postgres error 23505 for the
-- handler's 409 (DUPLICATE_ACCOUNT_NAME).
INSERT INTO accounts (id, plaid_id, name, user_id)
VALUES (sqlc.arg(id), sqlc.narg(plaid_id), sqlc.arg(name), sqlc.arg(user_id))
RETURNING *;

-- name: UpdateAccountName :one
UPDATE accounts
SET name = sqlc.arg(name)
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeleteAccount :one
DELETE FROM accounts
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
RETURNING id;

-- name: BulkDeleteAccounts :many
-- Missing or unowned ids are silently ignored: they simply do not appear in
-- the RETURNING set (matches fixtures: bulk-delete never 404s).
DELETE FROM accounts
WHERE id = ANY(sqlc.arg(ids)::text[])
  AND user_id = sqlc.arg(user_id)
RETURNING id;

-- name: ListOwnedAccountIDs :many
-- Set-based ownership check for bulk operations: one round trip for every
-- distinct account a batch references, rather than a GetAccount per row (500
-- rows would be 500 queries inside one transaction). The caller compares the
-- returned count against the number of distinct ids it asked for.
SELECT id
FROM accounts
WHERE user_id = sqlc.arg(user_id)
  AND id = ANY(sqlc.arg(ids)::text[]);
