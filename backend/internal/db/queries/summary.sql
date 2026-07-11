-- All three queries are filtered by owned accounts join + inclusive date
-- range + optional account id, exactly like the legacy summary.ts. The
-- handler calls GetPeriodTotals twice (current + previous period) and
-- computes percentage change, top-3 + "Other" category bucketing, and
-- missing-day zero-filling in Go, matching legacy JS.

-- name: GetPeriodTotals :one
-- income/expenses/remaining are SUM(...)::bigint, coalesced to 0 so an empty
-- period returns zeros instead of NULL (legacy JS does `value || 0`).
-- ABS operates on bigint: ABS(integer) raises integer-out-of-range for the
-- valid boundary value -2147483648, so amounts are widened first.
SELECT
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount::bigint) ELSE 0 END), 0)::bigint AS expenses,
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
    COALESCE(SUM(ABS(t.amount::bigint)), 0)::bigint AS value
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
JOIN categories c ON c.id = t.category_id AND c.user_id = sqlc.arg(user_id)
WHERE t.amount < 0
  AND t.date >= sqlc.arg(start_date)
  AND t.date <= sqlc.arg(end_date)
  AND (sqlc.narg(account_id)::text IS NULL OR t.account_id = sqlc.narg(account_id))
GROUP BY c.name
ORDER BY SUM(ABS(t.amount::bigint)) DESC;

-- name: GetDailyTotals :many
-- Per-day income/expenses. Zero-filling missing days in the requested
-- range stays in Go, as in legacy JS (fillMissingDays).
-- Aggregation is by calendar day (t.date::date), not the raw timestamp:
-- current write paths only store UTC midnight so the two are equivalent
-- today, but grouping by timestamp would silently split a day into
-- multiple rows if any future path stored a time of day, and the Go
-- zero-fill keyed by yyyy-MM-dd would drop all but one of them.
SELECT
    t.date::date AS day,
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount::bigint) ELSE 0 END), 0)::bigint AS expenses
FROM transactions t
JOIN accounts a ON a.id = t.account_id AND a.user_id = sqlc.arg(user_id)
WHERE t.date >= sqlc.arg(start_date)
  AND t.date <= sqlc.arg(end_date)
  AND (sqlc.narg(account_id)::text IS NULL OR t.account_id = sqlc.narg(account_id))
GROUP BY t.date::date
ORDER BY t.date::date;
