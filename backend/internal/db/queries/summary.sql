-- All three queries are filtered by owned accounts join + inclusive date
-- range + optional account id, exactly like the legacy summary.ts. The
-- handler calls GetPeriodTotals twice (current + previous period) and
-- computes percentage change, top-3 + "Other" category bucketing, and
-- missing-day zero-filling in Go, matching legacy JS.

-- name: GetPeriodTotals :one
-- income/expenses/remaining are SUM(...)::bigint, coalesced to 0 so an empty
-- period returns zeros instead of NULL (legacy JS does `value || 0`).
SELECT
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END), 0)::bigint AS expenses,
    COALESCE(SUM(t.amount), 0)::bigint AS remaining
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
WHERE t.date >= sqlc.arg(start_date)
  AND t.date <= sqlc.arg(end_date)
  AND (sqlc.narg(account_id)::text IS NULL OR t.account_id = sqlc.narg(account_id));

-- name: GetCategorySpending :many
-- Expense rows only (amount < 0), inner-joined to categories (legacy
-- innerJoin excludes uncategorized transactions from the breakdown).
-- Top-3 + "Other" bucketing stays in Go, as in legacy JS.
SELECT
    c.name AS name,
    COALESCE(SUM(ABS(t.amount)), 0)::bigint AS value
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
JOIN categories c ON c.id = t.category_id
WHERE t.amount < 0
  AND t.date >= sqlc.arg(start_date)
  AND t.date <= sqlc.arg(end_date)
  AND (sqlc.narg(account_id)::text IS NULL OR t.account_id = sqlc.narg(account_id))
GROUP BY c.name
ORDER BY SUM(ABS(t.amount)) DESC;

-- name: GetDailyTotals :many
-- Per-date income/expenses. Zero-filling missing days in the requested
-- range stays in Go, as in legacy JS (fillMissingDays).
SELECT
    t.date AS date,
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END), 0)::bigint AS expenses
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
WHERE t.date >= sqlc.arg(start_date)
  AND t.date <= sqlc.arg(end_date)
  AND (sqlc.narg(account_id)::text IS NULL OR t.account_id = sqlc.narg(account_id))
GROUP BY t.date
ORDER BY t.date;
