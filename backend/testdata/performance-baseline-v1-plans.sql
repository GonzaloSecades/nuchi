-- Query-plan capture for performance-baseline-v1.sql.
-- Run with psql so \timing and the EXPLAIN output are preserved together.

\timing on

BEGIN;
SET LOCAL ROLE nuchi;
SELECT set_config(
    'app.user_id',
    '10000000-0000-4000-8000-000000000001',
    true
);

-- listAccounts: one query.
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT id, name
FROM accounts
WHERE user_id = '10000000-0000-4000-8000-000000000001'::uuid
ORDER BY name;

-- listCategories: one query.
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT id, name
FROM categories
WHERE user_id = '10000000-0000-4000-8000-000000000001'::uuid
ORDER BY name;

-- listTransactions: one query, maximum supported 366-day window, all six
-- accounts, and the same deterministic date/id ordering as sqlc.
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT
    t.id,
    t.date,
    c.name AS category,
    t.category_id,
    t.payee,
    t.amount,
    t.notes,
    a.name AS account,
    t.account_id,
    t.currency
FROM transactions t
JOIN accounts a
  ON a.id = t.account_id
 AND a.user_id = '10000000-0000-4000-8000-000000000001'::uuid
LEFT JOIN categories c
  ON c.id = t.category_id
 AND c.user_id = '10000000-0000-4000-8000-000000000001'::uuid
WHERE t.date >= timestamp '2025-07-01'
  AND t.date <= timestamp '2026-07-01 23:59:59.999999'
ORDER BY t.date DESC, t.id DESC;

-- getSummary: four queries in one repeatable-read transaction in the
-- handler: current totals, previous totals, category spending, daily totals.
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END), 0)::bigint AS expenses,
    COALESCE(SUM(t.amount), 0)::bigint AS remaining
FROM transactions t
JOIN accounts a
  ON a.id = t.account_id
 AND a.user_id = '10000000-0000-4000-8000-000000000001'::uuid
WHERE t.date >= timestamp '2025-07-01'
  AND t.date <= timestamp '2026-07-01 23:59:59.999999';

EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END), 0)::bigint AS expenses,
    COALESCE(SUM(t.amount), 0)::bigint AS remaining
FROM transactions t
JOIN accounts a
  ON a.id = t.account_id
 AND a.user_id = '10000000-0000-4000-8000-000000000001'::uuid
WHERE t.date >= timestamp '2024-06-30'
  AND t.date <= timestamp '2025-06-30 23:59:59.999999';

EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT
    c.name,
    COALESCE(SUM(ABS(t.amount)), 0)::bigint AS value
FROM transactions t
JOIN accounts a
  ON a.id = t.account_id
 AND a.user_id = '10000000-0000-4000-8000-000000000001'::uuid
JOIN categories c
  ON c.id = t.category_id
 AND c.user_id = '10000000-0000-4000-8000-000000000001'::uuid
WHERE t.amount < 0
  AND t.date >= timestamp '2025-07-01'
  AND t.date <= timestamp '2026-07-01 23:59:59.999999'
GROUP BY c.name
ORDER BY SUM(ABS(t.amount)) DESC;

EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT
    t.date::date AS day,
    COALESCE(SUM(CASE WHEN t.amount >= 0 THEN t.amount ELSE 0 END), 0)::bigint AS income,
    COALESCE(SUM(CASE WHEN t.amount < 0 THEN ABS(t.amount) ELSE 0 END), 0)::bigint AS expenses
FROM transactions t
JOIN accounts a
  ON a.id = t.account_id
 AND a.user_id = '10000000-0000-4000-8000-000000000001'::uuid
WHERE t.date >= timestamp '2025-07-01'
  AND t.date <= timestamp '2026-07-01 23:59:59.999999'
GROUP BY t.date::date
ORDER BY t.date::date;

ROLLBACK;
