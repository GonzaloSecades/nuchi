-- Phase 0 performance baseline dataset, version 1.
--
-- Run only against a disposable local database as the migration/bootstrap
-- role. The two fixed user IDs make cleanup deterministic and keep the data
-- free of real user information.

BEGIN;

DELETE FROM users
WHERE id IN (
    '10000000-0000-4000-8000-000000000001'::uuid,
    '10000000-0000-4000-8000-000000000002'::uuid
);

INSERT INTO users (id, email, password_hash, email_verified_at)
VALUES
    (
        '10000000-0000-4000-8000-000000000001',
        'baseline-owner@example.invalid',
        'not-a-login-capable-password-hash',
        now()
    ),
    (
        '10000000-0000-4000-8000-000000000002',
        'baseline-neighbor@example.invalid',
        'not-a-login-capable-password-hash',
        now()
    );

INSERT INTO accounts (id, name, user_id)
SELECT
    'baseline-account-' || account_number,
    'Baseline account ' || account_number,
    '10000000-0000-4000-8000-000000000001'::uuid
FROM generate_series(1, 6) AS account_number;

INSERT INTO accounts (id, name, user_id)
VALUES (
    'baseline-neighbor-account',
    'Neighbor account',
    '10000000-0000-4000-8000-000000000002'
);

INSERT INTO categories (id, name, user_id)
SELECT
    'baseline-category-' || category_number,
    'Baseline category ' || lpad(category_number::text, 2, '0'),
    '10000000-0000-4000-8000-000000000001'::uuid
FROM generate_series(1, 12) AS category_number;

-- 100,000 rows over two years. The distribution deliberately includes:
-- six accounts, repeated same-day timestamps, 12 uneven categories, 10%
-- uncategorized expenses, 8% income, and a long expense tail.
INSERT INTO transactions (
    id,
    amount,
    payee,
    notes,
    date,
    account_id,
    category_id,
    currency
)
SELECT
    'baseline-transaction-' || row_number,
    CASE
        WHEN row_number % 25 = 0 THEN 2500000 + (row_number % 17) * 10000
        WHEN row_number % 13 = 0 THEN 700000 + (row_number % 31) * 1000
        ELSE -(5000 + (row_number % 997) * 137)
    END,
    'Baseline payee ' || (row_number % 250),
    CASE WHEN row_number % 5 = 0 THEN 'deterministic baseline note' END,
    timestamp '2025-01-01' + (row_number % 730) * interval '1 day',
    'baseline-account-' || ((row_number % 6) + 1),
    CASE
        WHEN row_number % 10 = 0 THEN NULL
        ELSE 'baseline-category-' || (((row_number::bigint * row_number) % 12) + 1)
    END,
    'ARS'
FROM generate_series(1, 100000) AS row_number;

-- A second tenant catches plans or fixtures that accidentally depend on a
-- single-user database. These rows must remain invisible to the owner above.
INSERT INTO transactions (
    id,
    amount,
    payee,
    date,
    account_id,
    currency
)
SELECT
    'baseline-neighbor-transaction-' || row_number,
    -(1000 + row_number),
    'Neighbor payee',
    timestamp '2025-01-01' + (row_number % 730) * interval '1 day',
    'baseline-neighbor-account',
    'ARS'
FROM generate_series(1, 5000) AS row_number;

COMMIT;

ANALYZE users;
ANALYZE accounts;
ANALYZE categories;
ANALYZE transactions;
