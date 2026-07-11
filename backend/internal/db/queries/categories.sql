-- Mirrors accounts.sql exactly (categories.ts mirrors accounts.ts).

-- name: ListCategories :many
SELECT *
FROM categories
WHERE user_id = sqlc.arg(user_id)
ORDER BY name;

-- name: GetCategory :one
SELECT *
FROM categories
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id);

-- name: CreateCategory :one
-- Duplicate (user_id, name) surfaces as Postgres error 23505 for the
-- handler's 409 (DUPLICATE_CATEGORY_NAME).
INSERT INTO categories (id, plaid_id, name, user_id)
VALUES (sqlc.arg(id), sqlc.narg(plaid_id), sqlc.arg(name), sqlc.arg(user_id))
RETURNING *;

-- name: UpdateCategoryName :one
UPDATE categories
SET name = sqlc.arg(name)
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: DeleteCategory :one
DELETE FROM categories
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
RETURNING id;

-- name: BulkDeleteCategories :many
-- Missing or unowned ids are silently ignored: they simply do not appear in
-- the RETURNING set (matches fixtures: bulk-delete never 404s).
DELETE FROM categories
WHERE id = ANY(sqlc.arg(ids)::text[])
  AND user_id = sqlc.arg(user_id)
RETURNING id;
