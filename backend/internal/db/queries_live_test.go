package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestSqlcQueries_LiveDatabase exercises a representative path through the
// sqlc-generated query code (backend/internal/db/gen), not every query:
// create user -> account -> category -> transactions; ListTransactions
// (joined names, inclusive date filtering, DESC order); BulkDeleteTransactions
// silently ignoring an id owned by another user; the atomic one-time consume
// on password_reset_tokens; RevokeAllUserRefreshTokens; and GetPeriodTotals
// milliunit sums. Exhaustive per-query/per-endpoint behavior arrives with
// handlers (#44+) against the fixtures.
//
// Everything runs inside a single BEGIN ... ROLLBACK so the database is left
// clean regardless of outcome, following finance_rls_test.go conventions.
// Owned-resource queries run under FORCE RLS, so app.user_id is set via
// setAppUser (defined in finance_rls_test.go) before each is exercised.
func TestSqlcQueries_LiveDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live database sqlc round-trip test")
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

	q := dbgen.New(tx)

	// --- users.sql: create two users. The users table has no RLS (decided
	// in #38), so no app.user_id is needed for these inserts. ---
	userA, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		ID:           newPgUUID(),
		Email:        uniqueTestEmail("sqlc-a"),
		PasswordHash: "hash-a",
	})
	if err != nil {
		t.Fatalf("CreateUser(A): unexpected error: %v", err)
	}
	userB, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		ID:           newPgUUID(),
		Email:        uniqueTestEmail("sqlc-b"),
		PasswordHash: "hash-b",
	})
	if err != nil {
		t.Fatalf("CreateUser(B): unexpected error: %v", err)
	}

	// --- accounts.sql / categories.sql: one owned account + category per
	// user, seeded under each user's own app.user_id so RLS WITH CHECK
	// accepts the insert. ---
	if err := setAppUser(ctx, tx, userA.ID.String()); err != nil {
		t.Fatalf("failed to set app.user_id for user A: %v", err)
	}

	accountA, err := q.CreateAccount(ctx, dbgen.CreateAccountParams{
		ID:     "acc_sqlc_a",
		Name:   "Sqlc Checking",
		UserID: userA.ID,
	})
	if err != nil {
		t.Fatalf("CreateAccount(A): unexpected error: %v", err)
	}
	categoryA, err := q.CreateCategory(ctx, dbgen.CreateCategoryParams{
		ID:     "cat_sqlc_a",
		Name:   "Sqlc Groceries",
		UserID: userA.ID,
	})
	if err != nil {
		t.Fatalf("CreateCategory(A): unexpected error: %v", err)
	}

	// --- transactions.sql: three transactions for user A across different
	// dates, one of them categorized. ---
	txn1Date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txn2Date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	txn3Date := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)

	txn1, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID:         "txn_sqlc_1",
		Amount:     -12500,
		Payee:      "Market",
		Notes:      pgtype.Text{String: "weekly shop", Valid: true},
		Date:       pgtype.Timestamp{Time: txn1Date, Valid: true},
		AccountID:  accountA.ID,
		CategoryID: pgtype.Text{String: categoryA.ID, Valid: true},
		Currency:   "ARS",
	})
	if err != nil {
		t.Fatalf("CreateTransaction(txn1): unexpected error: %v", err)
	}
	txn2, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID:        "txn_sqlc_2",
		Amount:    500000,
		Payee:     "Salary",
		Date:      pgtype.Timestamp{Time: txn2Date, Valid: true},
		AccountID: accountA.ID,
		Currency:  "ARS",
	})
	if err != nil {
		t.Fatalf("CreateTransaction(txn2): unexpected error: %v", err)
	}
	txn3, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID:        "txn_sqlc_3",
		Amount:    -30000,
		Payee:     "Rent",
		Date:      pgtype.Timestamp{Time: txn3Date, Valid: true},
		AccountID: accountA.ID,
		Currency:  "ARS",
	})
	if err != nil {
		t.Fatalf("CreateTransaction(txn3): unexpected error: %v", err)
	}

	// --- ListTransactions: joined names, inclusive date range, DESC order. ---
	rangeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	listed, err := q.ListTransactions(ctx, dbgen.ListTransactionsParams{
		UserID:    userA.ID,
		StartDate: pgtype.Timestamp{Time: rangeStart, Valid: true},
		EndDate:   pgtype.Timestamp{Time: rangeEnd, Valid: true},
	})
	if err != nil {
		t.Fatalf("ListTransactions: unexpected error: %v", err)
	}
	if len(listed) != 3 {
		t.Fatalf("ListTransactions: expected 3 rows, got %d", len(listed))
	}
	wantOrder := []string{txn3.ID, txn2.ID, txn1.ID}
	for i, want := range wantOrder {
		if listed[i].ID != want {
			t.Errorf("ListTransactions: row %d: expected id %q (DESC by date), got %q", i, want, listed[i].ID)
		}
	}
	if !listed[2].Category.Valid || listed[2].Category.String != "Sqlc Groceries" {
		t.Errorf("ListTransactions: expected txn1 joined category name %q, got %+v", "Sqlc Groceries", listed[2].Category)
	}
	if listed[2].Account != "Sqlc Checking" {
		t.Errorf("ListTransactions: expected joined account name %q, got %q", "Sqlc Checking", listed[2].Account)
	}
	if listed[1].Category.Valid {
		t.Errorf("ListTransactions: expected txn2 (uncategorized) joined category to be NULL, got %+v", listed[1].Category)
	}

	// Narrower range excludes txn1 (Jan 10) but keeps txn2/txn3.
	narrowStart := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	listedNarrow, err := q.ListTransactions(ctx, dbgen.ListTransactionsParams{
		UserID:    userA.ID,
		StartDate: pgtype.Timestamp{Time: narrowStart, Valid: true},
		EndDate:   pgtype.Timestamp{Time: rangeEnd, Valid: true},
	})
	if err != nil {
		t.Fatalf("ListTransactions (narrow range): unexpected error: %v", err)
	}
	if len(listedNarrow) != 2 {
		t.Fatalf("ListTransactions (narrow range): expected 2 rows, got %d", len(listedNarrow))
	}

	// --- BulkDeleteTransactions: seed a second user's account + transaction,
	// then attempt to bulk-delete both txn1 (owned by A) and userB's
	// transaction while authenticated as A. Only the owned id comes back;
	// userB's row survives untouched. ---
	if err := setAppUser(ctx, tx, userB.ID.String()); err != nil {
		t.Fatalf("failed to set app.user_id for user B: %v", err)
	}
	accountB, err := q.CreateAccount(ctx, dbgen.CreateAccountParams{
		ID:     "acc_sqlc_b",
		Name:   "Sqlc B Checking",
		UserID: userB.ID,
	})
	if err != nil {
		t.Fatalf("CreateAccount(B): unexpected error: %v", err)
	}
	txnB, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
		ID:        "txn_sqlc_b",
		Amount:    -1000,
		Payee:     "Unowned",
		Date:      pgtype.Timestamp{Time: txn1Date, Valid: true},
		AccountID: accountB.ID,
		Currency:  "ARS",
	})
	if err != nil {
		t.Fatalf("CreateTransaction(B): unexpected error: %v", err)
	}

	if err := setAppUser(ctx, tx, userA.ID.String()); err != nil {
		t.Fatalf("failed to set app.user_id back to user A: %v", err)
	}
	deletedIDs, err := q.BulkDeleteTransactions(ctx, dbgen.BulkDeleteTransactionsParams{
		Ids:    []string{txn1.ID, txnB.ID},
		UserID: userA.ID,
	})
	if err != nil {
		t.Fatalf("BulkDeleteTransactions: unexpected error: %v", err)
	}
	if len(deletedIDs) != 1 || deletedIDs[0] != txn1.ID {
		t.Errorf("BulkDeleteTransactions: expected only [%q] deleted, got %v", txn1.ID, deletedIDs)
	}

	if err := setAppUser(ctx, tx, userB.ID.String()); err != nil {
		t.Fatalf("failed to set app.user_id for user B verification: %v", err)
	}
	survivor, err := q.GetTransaction(ctx, dbgen.GetTransactionParams{
		UserID: userB.ID,
		ID:     txnB.ID,
	})
	if err != nil {
		t.Fatalf("GetTransaction(B): expected user B's transaction to survive the bulk-delete, got error: %v", err)
	}
	if survivor.ID != txnB.ID {
		t.Errorf("GetTransaction(B): expected id %q, got %q", txnB.ID, survivor.ID)
	}

	// --- GetPeriodTotals: after the bulk-delete, user A has txn2 (+500000)
	// and txn3 (-30000) remaining. ---
	if err := setAppUser(ctx, tx, userA.ID.String()); err != nil {
		t.Fatalf("failed to set app.user_id back to user A: %v", err)
	}
	totals, err := q.GetPeriodTotals(ctx, dbgen.GetPeriodTotalsParams{
		UserID:    userA.ID,
		StartDate: pgtype.Timestamp{Time: rangeStart, Valid: true},
		EndDate:   pgtype.Timestamp{Time: rangeEnd, Valid: true},
	})
	if err != nil {
		t.Fatalf("GetPeriodTotals: unexpected error: %v", err)
	}
	if totals.Income != 500000 {
		t.Errorf("GetPeriodTotals: expected income 500000 milliunits, got %d", totals.Income)
	}
	if totals.Expenses != 30000 {
		t.Errorf("GetPeriodTotals: expected expenses 30000 milliunits, got %d", totals.Expenses)
	}
	if totals.Remaining != 470000 {
		t.Errorf("GetPeriodTotals: expected remaining 470000 milliunits, got %d", totals.Remaining)
	}

	// --- auth_tokens.sql: password reset token atomic one-time consume.
	// These tables carry no RLS; app.user_id is irrelevant to them. ---
	resetToken, err := q.CreatePasswordResetToken(ctx, dbgen.CreatePasswordResetTokenParams{
		ID:        newPgUUID(),
		UserID:    userA.ID,
		TokenHash: "reset-hash-sqlc-test",
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreatePasswordResetToken: unexpected error: %v", err)
	}

	consumedUserID, err := q.ConsumePasswordResetToken(ctx, resetToken.TokenHash)
	if err != nil {
		t.Fatalf("ConsumePasswordResetToken: expected first consume to succeed, got error: %v", err)
	}
	if consumedUserID != userA.ID {
		t.Errorf("ConsumePasswordResetToken: expected user id %v, got %v", userA.ID, consumedUserID)
	}

	if _, err := q.ConsumePasswordResetToken(ctx, resetToken.TokenHash); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("ConsumePasswordResetToken: expected second consume to fail with ErrNoRows, got %v", err)
	}

	// --- auth_tokens.sql: RevokeAllUserRefreshTokens. ---
	refresh1, err := q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		ID:        newPgUUID(),
		UserID:    userA.ID,
		TokenHash: "refresh-hash-sqlc-1",
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateRefreshToken(1): unexpected error: %v", err)
	}
	refresh2, err := q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		ID:        newPgUUID(),
		UserID:    userA.ID,
		TokenHash: "refresh-hash-sqlc-2",
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateRefreshToken(2): unexpected error: %v", err)
	}

	// Both are valid before revocation.
	if _, err := q.GetRefreshTokenByHash(ctx, refresh1.TokenHash); err != nil {
		t.Fatalf("GetRefreshTokenByHash(1): expected token to be valid before revocation, got error: %v", err)
	}
	if _, err := q.GetRefreshTokenByHash(ctx, refresh2.TokenHash); err != nil {
		t.Fatalf("GetRefreshTokenByHash(2): expected token to be valid before revocation, got error: %v", err)
	}

	if err := q.RevokeAllUserRefreshTokens(ctx, userA.ID); err != nil {
		t.Fatalf("RevokeAllUserRefreshTokens: unexpected error: %v", err)
	}

	if _, err := q.GetRefreshTokenByHash(ctx, refresh1.TokenHash); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("GetRefreshTokenByHash(1): expected ErrNoRows after RevokeAllUserRefreshTokens, got %v", err)
	}
	if _, err := q.GetRefreshTokenByHash(ctx, refresh2.TokenHash); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("GetRefreshTokenByHash(2): expected ErrNoRows after RevokeAllUserRefreshTokens, got %v", err)
	}
}

// newPgUUID generates a fresh random UUID as a valid pgtype.UUID, for
// supplying ids on inserts (the app supplies ids explicitly; sqlc queries
// never rely on column defaults).
func newPgUUID() pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}
}

// uniqueTestEmail builds a unique email address for a test user so repeated
// runs against the same database never collide on the users.email unique
// index.
func uniqueTestEmail(label string) string {
	return fmt.Sprintf("%s-%s@example.test", label, uuid.NewString())
}
