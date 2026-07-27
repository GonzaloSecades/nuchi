package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestTransactionsCategoryOwnerRLS_LiveDatabase pins the 00004 policy tightening:
// a write may reference no category or a category owned by the same
// app.user_id, but never another tenant's category. Foreign-key checks run as
// the table owner and bypass RLS, so without the WITH CHECK clause the FK alone
// would happily accept the cross-owner reference — the regression this guards.
//
// Every probe is predicate-free (the category constraint lives only in the
// policy, never in a WHERE clause) so a pass proves the RLS policy itself is
// doing the work. Expected-error cases run inside pgx pseudo-nested
// transactions (SAVEPOINTs) so the outer transaction survives; the whole test
// rolls back, leaving the database clean.
func TestTransactionsCategoryOwnerRLS_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "transactions category-owner RLS test")

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

	userA := insertTestUser(ctx, t, tx, "cat-rls-a")
	userB := insertTestUser(ctx, t, tx, "cat-rls-b")

	txDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Seed user A's account + own category, and user B's category, each under
	// its own app.user_id so WITH CHECK accepts the seed writes.
	accountA := uuid.NewString()
	categoryA := uuid.NewString()
	if err := setAppUser(ctx, tx, userA); err != nil {
		t.Fatalf("failed to set app.user_id for user A seed: %v", err)
	}
	insertTestAccount(ctx, t, tx, accountA, "Cat RLS Account A", userA)
	insertTestCategory(ctx, t, tx, categoryA, "Cat RLS Category A", userA)

	categoryB := uuid.NewString()
	if err := setAppUser(ctx, tx, userB); err != nil {
		t.Fatalf("failed to set app.user_id for user B seed: %v", err)
	}
	insertTestCategory(ctx, t, tx, categoryB, "Cat RLS Category B", userB)

	// Act as user A for every case below.
	if err := setAppUser(ctx, tx, userA); err != nil {
		t.Fatalf("failed to set app.user_id for user A cases: %v", err)
	}

	// (1) Own category → accepted.
	ownTxnID := uuid.NewString()
	if _, err := tx.Exec(ctx, `
		INSERT INTO transactions (id, amount, payee, date, account_id, category_id, currency)
		VALUES ($1, $2, $3, $4, $5, $6, 'ARS')
	`, ownTxnID, 1000, "own category", txDate, accountA, categoryA); err != nil {
		t.Fatalf("user A: expected insert with own category to succeed, got %v", err)
	}

	// (2) NULL category → accepted.
	nullTxnID := uuid.NewString()
	if _, err := tx.Exec(ctx, `
		INSERT INTO transactions (id, amount, payee, date, account_id, category_id, currency)
		VALUES ($1, $2, $3, $4, $5, NULL, 'ARS')
	`, nullTxnID, 1000, "null category", txDate, accountA); err != nil {
		t.Fatalf("user A: expected insert with NULL category to succeed, got %v", err)
	}

	// (3) Another tenant's category on a single insert → rejected by WITH CHECK.
	func() {
		nested, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("nested tx for cross-owner insert: %v", err)
		}
		defer func() {
			if err := nested.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				t.Errorf("rollback nested (insert): %v", err)
			}
		}()

		_, err = nested.Exec(ctx, `
			INSERT INTO transactions (id, amount, payee, date, account_id, category_id, currency)
			VALUES ($1, $2, $3, $4, $5, $6, 'ARS')
		`, uuid.NewString(), 1000, "cross-owner category insert", txDate, accountA, categoryB)
		assertRLSViolation(t, err, "user A inserting a transaction referencing user B's category")
	}()

	// (4) Update an owned transaction to point at another tenant's category →
	// rejected by WITH CHECK.
	func() {
		nested, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("nested tx for cross-owner update: %v", err)
		}
		defer func() {
			if err := nested.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				t.Errorf("rollback nested (update): %v", err)
			}
		}()

		_, err = nested.Exec(ctx, `
			UPDATE transactions SET category_id = $1 WHERE id = $2
		`, categoryB, ownTxnID)
		assertRLSViolation(t, err, "user A updating an owned transaction to user B's category")
	}()

	// (5) Bulk insert where one row references another tenant's category → the
	// whole multi-row statement is rejected (atomic), so the sibling good row
	// is not inserted either.
	func() {
		nested, err := tx.Begin(ctx)
		if err != nil {
			t.Fatalf("nested tx for bulk cross-owner insert: %v", err)
		}
		defer func() {
			if err := nested.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				t.Errorf("rollback nested (bulk): %v", err)
			}
		}()

		goodID := uuid.NewString()
		badID := uuid.NewString()
		_, err = nested.Exec(ctx, `
			INSERT INTO transactions (id, amount, payee, date, account_id, category_id, currency)
			VALUES ($1, $2, $3, $4, $5, $6, 'ARS'),
			       ($7, $8, $9, $10, $11, $12, 'ARS')
		`,
			goodID, 1000, "bulk good", txDate, accountA, categoryA,
			badID, 1000, "bulk bad", txDate, accountA, categoryB,
		)
		assertRLSViolation(t, err, "user A bulk-inserting transactions with one cross-owner category")
	}()

	// Only the two accepted rows (own category, NULL category) exist for user A;
	// none of the rejected single/bulk writes leaked through.
	assertRowCount(ctx, t, tx, "transactions", 2)
}
