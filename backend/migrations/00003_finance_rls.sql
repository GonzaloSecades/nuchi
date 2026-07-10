-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE accounts FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE categories FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE POLICY accounts_owner ON accounts
    FOR ALL
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE POLICY categories_owner ON categories
    FOR ALL
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);
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

-- +goose Down
-- +goose StatementBegin
DROP POLICY transactions_owner ON transactions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP POLICY categories_owner ON categories;
-- +goose StatementEnd

-- +goose StatementBegin
DROP POLICY accounts_owner ON accounts;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions NO FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions DISABLE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE categories NO FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE categories DISABLE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE accounts NO FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE accounts DISABLE ROW LEVEL SECURITY;
-- +goose StatementEnd
