-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
    id text PRIMARY KEY,
    plaid_id text,
    name citext NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX accounts_user_id_idx ON accounts (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX accounts_user_id_name_uniq ON accounts (user_id, name);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE categories (
    id text PRIMARY KEY,
    plaid_id text,
    name citext NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX categories_user_id_idx ON categories (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX categories_user_id_name_uniq ON categories (user_id, name);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transactions (
    id text PRIMARY KEY,
    amount integer NOT NULL,
    payee text NOT NULL,
    notes text,
    date timestamp NOT NULL,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    category_id text REFERENCES categories(id) ON DELETE SET NULL,
    currency text NOT NULL DEFAULT 'ARS'
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX transactions_account_id_date_idx ON transactions (account_id, date DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX transactions_category_id_idx ON transactions (category_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE transactions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE categories;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE accounts;
-- +goose StatementEnd
