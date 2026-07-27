-- +goose Up
-- Tightens transactions_owner so a write must reference either no category or
-- a category owned by the same app.user_id, and cleans up any pre-existing
-- cross-owner references first.
--
-- Background: the #39 policy proved transaction ownership only through
-- account_id. A user could therefore write a transaction on their own account
-- while pointing category_id at another tenant's category. PostgreSQL foreign
-- key checks run as the table owner and bypass RLS, so the FK does not reject
-- that reference. Left unfixed it breaks tenant integrity, leaks a category
-- existence oracle, and lets the other tenant's ON DELETE SET NULL mutate this
-- user's row. The legacy Hono transactions handler already rejects an unowned
-- category with 404 "Category not found"; this is the RLS backstop for that
-- same rule, per the "RLS is the security backstop; SQL still includes
-- ownership predicates" invariant.

-- The cleanup UPDATE must run with FORCE RLS momentarily lifted: goose applies
-- this migration as the ordinary table-owning role, which FORCE RLS otherwise
-- subjects to the (unset app.user_id => zero visible rows) policy, making the
-- UPDATE a silent no-op. The whole migration runs in one transaction and the
-- ALTER takes an ACCESS EXCLUSIVE lock, so the lifted window is atomic and not
-- observable to any concurrent session. A transaction's owner is its account's
-- owner; null any category that belongs to a different user.
-- +goose StatementBegin
ALTER TABLE transactions NO FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE transactions t
SET category_id = NULL
FROM accounts a, categories c
WHERE t.account_id = a.id
  AND t.category_id = c.id
  AND c.user_id <> a.user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
DROP POLICY transactions_owner ON transactions;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE POLICY transactions_owner ON transactions
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM accounts a
            WHERE a.id = transactions.account_id
                AND a.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM accounts a
            WHERE a.id = transactions.account_id
                AND a.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
        )
        AND (
            transactions.category_id IS NULL
            OR EXISTS (
                SELECT 1 FROM categories c
                WHERE c.id = transactions.category_id
                    AND c.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
            )
        )
    );
-- +goose StatementEnd

-- +goose Down
-- Restores the account-only policy from #39. The cleanup UPDATE is not
-- reversible (nulled categories are not recorded); Down restores the policy
-- shape, not the data, which is the standard expectation for a data-cleanup
-- migration.
-- +goose StatementBegin
DROP POLICY transactions_owner ON transactions;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE POLICY transactions_owner ON transactions
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM accounts a
            WHERE a.id = transactions.account_id
                AND a.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM accounts a
            WHERE a.id = transactions.account_id
                AND a.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
        )
    );
-- +goose StatementEnd
