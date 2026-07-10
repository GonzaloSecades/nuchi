package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestFinanceRLSConfiguration_LiveDatabase asserts that the finance RLS
// migration (backend/migrations/00003_finance_rls.sql) has enabled and
// forced row level security on accounts, categories, and transactions, and
// that exactly the expected policy exists on each. It is skipped unless
// TEST_DATABASE_URL is set.
func TestFinanceRLSConfiguration_LiveDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live database RLS test")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	defer pool.Close()

	for _, table := range []string{"accounts", "categories", "transactions"} {
		var rowSecurity, forceRowSecurity bool
		row := pool.QueryRow(ctx, `
			SELECT relrowsecurity, relforcerowsecurity
			FROM pg_class
			WHERE relname = $1 AND relnamespace = 'public'::regnamespace
		`, table)
		if err := row.Scan(&rowSecurity, &forceRowSecurity); err != nil {
			t.Fatalf("%s: failed to query pg_class RLS flags: %v", table, err)
		}
		if !rowSecurity {
			t.Errorf("%s: expected relrowsecurity=true", table)
		}
		if !forceRowSecurity {
			t.Errorf("%s: expected relforcerowsecurity=true", table)
		}
	}

	expectedPolicies := map[string]string{
		"accounts":     "accounts_owner",
		"categories":   "categories_owner",
		"transactions": "transactions_owner",
	}
	for table, policyName := range expectedPolicies {
		var total int
		row := pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_policies WHERE schemaname = 'public' AND tablename = $1
		`, table)
		if err := row.Scan(&total); err != nil {
			t.Fatalf("%s: failed to count policies: %v", table, err)
		}
		if total != 1 {
			t.Errorf("%s: expected exactly 1 policy, got %d", table, total)
		}

		var named int
		row = pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_policies WHERE schemaname = 'public' AND tablename = $1 AND policyname = $2
		`, table, policyName)
		if err := row.Scan(&named); err != nil {
			t.Fatalf("%s: failed to count named policy %q: %v", table, policyName, err)
		}
		if named != 1 {
			t.Errorf("%s: expected a policy named %q, found %d", table, policyName, named)
		}
	}
}

// TestFinanceRLS_LiveDatabase exercises RLS enforcement on accounts and
// transactions: an owner sees only their own rows, cross-user reads return
// nothing, cross-user inserts are rejected by WITH CHECK, cross-user
// updates/deletes affect zero rows, and an unset app.user_id fails closed.
// Everything runs inside a single BEGIN ... ROLLBACK so the database is left
// clean regardless of outcome. It is skipped unless TEST_DATABASE_URL is
// set.
func TestFinanceRLS_LiveDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live database RLS test")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("expected to acquire a connection, got error: %v", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("expected to begin a transaction, got error: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("cleanup: failed to roll back test transaction: %v", err)
		}
	}()

	// Seed two users. The users table has no RLS (auth-layer, decided in
	// #38), so these inserts need no app.user_id.
	userA := insertTestUser(ctx, t, tx, "rls-a")
	userB := insertTestUser(ctx, t, tx, "rls-b")

	txDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Seed one account + transaction per user, each under its own
	// app.user_id so WITH CHECK accepts the insert.
	accountA := uuid.NewString()
	transactionA := uuid.NewString()
	if err := setAppUser(ctx, tx, userA); err != nil {
		t.Fatalf("failed to set app.user_id for user A seed: %v", err)
	}
	insertTestAccount(ctx, t, tx, accountA, "RLS Account A", userA)
	insertTestTransaction(ctx, t, tx, transactionA, accountA, txDate)

	accountB := uuid.NewString()
	transactionB := uuid.NewString()
	if err := setAppUser(ctx, tx, userB); err != nil {
		t.Fatalf("failed to set app.user_id for user B seed: %v", err)
	}
	insertTestAccount(ctx, t, tx, accountB, "RLS Account B", userB)
	insertTestTransaction(ctx, t, tx, transactionB, accountB, txDate)

	// --- As user A ---
	if err := setAppUser(ctx, tx, userA); err != nil {
		t.Fatalf("failed to set app.user_id for user A: %v", err)
	}

	assertRowCount(ctx, t, tx, "accounts", 1)
	assertRowCount(ctx, t, tx, "transactions", 1)

	var gotAccountID string
	if err := tx.QueryRow(ctx, `SELECT id FROM accounts`).Scan(&gotAccountID); err != nil {
		t.Fatalf("user A: failed to read own account: %v", err)
	}
	if gotAccountID != accountA {
		t.Errorf("user A: expected to see own account %q, got %q", accountA, gotAccountID)
	}

	var gotTransactionID string
	if err := tx.QueryRow(ctx, `SELECT id FROM transactions`).Scan(&gotTransactionID); err != nil {
		t.Fatalf("user A: failed to read own transaction: %v", err)
	}
	if gotTransactionID != transactionA {
		t.Errorf("user A: expected to see own transaction %q, got %q", transactionA, gotTransactionID)
	}

	// User A cannot see user B's account or transaction directly by id.
	var probe string
	err = tx.QueryRow(ctx, `SELECT id FROM accounts WHERE id = $1`, accountB).Scan(&probe)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("user A: expected no rows selecting user B's account, got err=%v", err)
	}
	err = tx.QueryRow(ctx, `SELECT id FROM transactions WHERE id = $1`, transactionB).Scan(&probe)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("user A: expected no rows selecting user B's transaction, got err=%v", err)
	}

	// User A cannot INSERT a transaction into user B's account: WITH CHECK
	// rejects it. Run inside a pseudo-nested transaction (pgx issues a
	// SAVEPOINT) so the outer transaction survives the expected error.
	func() {
		nested, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("failed to open nested transaction for negative insert test: %v", err)
		}
		defer func() {
			if err := nested.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				t.Errorf("failed to roll back nested transaction: %v", err)
			}
		}()

		_, err = nested.Exec(ctx, `
			INSERT INTO transactions (id, amount, payee, date, account_id, currency)
			VALUES ($1, $2, $3, $4, $5, 'ARS')
		`, uuid.NewString(), -500, "cross-user insert attempt", txDate, accountB)
		assertRLSViolation(t, err, "user A inserting a transaction into user B's account")
	}()

	// User A cannot UPDATE or DELETE user B's rows: RLS makes them invisible
	// to the WHERE clause, so zero rows are affected (no error).
	tag, err := tx.Exec(ctx, `UPDATE accounts SET name = $1 WHERE id = $2`, "hacked", accountB)
	if err != nil {
		t.Fatalf("user A: UPDATE on user B's account returned an unexpected error: %v", err)
	}
	if tag.RowsAffected() != 0 {
		t.Errorf("user A: expected 0 rows affected updating user B's account, got %d", tag.RowsAffected())
	}

	tag, err = tx.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, accountB)
	if err != nil {
		t.Fatalf("user A: DELETE on user B's account returned an unexpected error: %v", err)
	}
	if tag.RowsAffected() != 0 {
		t.Errorf("user A: expected 0 rows affected deleting user B's account, got %d", tag.RowsAffected())
	}

	tag, err = tx.Exec(ctx, `UPDATE transactions SET payee = $1 WHERE id = $2`, "hacked", transactionB)
	if err != nil {
		t.Fatalf("user A: UPDATE on user B's transaction returned an unexpected error: %v", err)
	}
	if tag.RowsAffected() != 0 {
		t.Errorf("user A: expected 0 rows affected updating user B's transaction, got %d", tag.RowsAffected())
	}

	tag, err = tx.Exec(ctx, `DELETE FROM transactions WHERE id = $1`, transactionB)
	if err != nil {
		t.Fatalf("user A: DELETE on user B's transaction returned an unexpected error: %v", err)
	}
	if tag.RowsAffected() != 0 {
		t.Errorf("user A: expected 0 rows affected deleting user B's transaction, got %d", tag.RowsAffected())
	}

	// User B's rows must still be present (the above UPDATE/DELETE attempts
	// were no-ops), confirmed from user B's own vantage point.
	if err := setAppUser(ctx, tx, userB); err != nil {
		t.Fatalf("failed to set app.user_id for user B verification: %v", err)
	}
	assertRowCount(ctx, t, tx, "accounts", 1)
	assertRowCount(ctx, t, tx, "transactions", 1)

	// --- With app.user_id unset: fail closed ---
	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', '', true)`); err != nil {
		t.Fatalf("failed to unset app.user_id: %v", err)
	}
	assertRowCount(ctx, t, tx, "accounts", 0)
	assertRowCount(ctx, t, tx, "categories", 0)
	assertRowCount(ctx, t, tx, "transactions", 0)

	// Writes fail closed too: WITH CHECK rejects an insert with no
	// app.user_id set.
	func() {
		nested, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("failed to open nested transaction for unset-user insert test: %v", err)
		}
		defer func() {
			if err := nested.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				t.Errorf("failed to roll back nested transaction: %v", err)
			}
		}()

		_, err = nested.Exec(ctx, `
			INSERT INTO accounts (id, name, user_id) VALUES ($1, $2, $3)
		`, uuid.NewString(), "no owner set", userA)
		assertRLSViolation(t, err, "inserting an account with app.user_id unset")
	}()
}

// insertTestUser inserts a user with a unique test email and returns its
// generated id.
func insertTestUser(ctx context.Context, t *testing.T, tx pgx.Tx, label string) string {
	t.Helper()

	var id string
	email := fmt.Sprintf("%s-%s@example.test", label, uuid.NewString())
	row := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, 'test-hash') RETURNING id
	`, email)
	if err := row.Scan(&id); err != nil {
		t.Fatalf("failed to insert test user %q: %v", label, err)
	}
	return id
}

// setAppUser sets the app.user_id GUC for the remainder of the current
// (sub)transaction, mirroring the SET LOCAL app.user_id binding #43 performs
// per request. set_config(..., true) is used instead of SET LOCAL because it
// accepts a bind parameter.
func setAppUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, userID)
	return err
}

func insertTestAccount(ctx context.Context, t *testing.T, tx pgx.Tx, id, name, userID string) {
	t.Helper()

	if _, err := tx.Exec(ctx, `
		INSERT INTO accounts (id, name, user_id) VALUES ($1, $2, $3)
	`, id, name, userID); err != nil {
		t.Fatalf("failed to insert test account %q: %v", id, err)
	}
}

func insertTestTransaction(ctx context.Context, t *testing.T, tx pgx.Tx, id, accountID string, date time.Time) {
	t.Helper()

	if _, err := tx.Exec(ctx, `
		INSERT INTO transactions (id, amount, payee, date, account_id, currency)
		VALUES ($1, $2, $3, $4, $5, 'ARS')
	`, id, 1000, "test payee", date, accountID); err != nil {
		t.Fatalf("failed to insert test transaction %q: %v", id, err)
	}
}

func assertRowCount(ctx context.Context, t *testing.T, tx pgx.Tx, table string, want int) {
	t.Helper()

	var got int
	if err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s`, table)).Scan(&got); err != nil {
		t.Fatalf("%s: failed to count rows: %v", table, err)
	}
	if got != want {
		t.Errorf("%s: expected %d visible row(s), got %d", table, want, got)
	}
}

// assertRLSViolation fails the test unless err is a Postgres row-level
// security policy violation (SQLSTATE 42501), distinguishing that failure
// mode from other kinds of errors (e.g. a missing FK, a syntax error).
func assertRLSViolation(t *testing.T, err error, action string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected a row-level security violation, got nil error", action)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("%s: expected a *pgconn.PgError, got %T: %v", action, err, err)
	}
	if pgErr.Code != "42501" {
		t.Fatalf("%s: expected SQLSTATE 42501 (insufficient_privilege), got %q: %v", action, pgErr.Code, err)
	}
}
