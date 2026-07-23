package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createRLSTestUser inserts a user with a unique test email and returns its
// generated id. The users table carries no RLS (auth-layer, decided in
// #38), so this insert needs no app.user_id binding.
func createRLSTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool, label string) uuid.UUID {
	t.Helper()

	var id string
	email := fmt.Sprintf("%s-%s@example.test", label, uuid.NewString())
	row := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, 'test-hash') RETURNING id
	`, email)
	if err := row.Scan(&id); err != nil {
		t.Fatalf("failed to insert test user %q: %v", label, err)
	}
	return uuid.MustParse(id)
}

// deleteRLSTestUsers removes test users. Deleting the user cascades to any
// accounts/categories/transactions created for it (ON DELETE CASCADE), so
// this alone is sufficient cleanup even though those owned-table deletes
// would themselves be subject to RLS.
func deleteRLSTestUsers(ctx context.Context, t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) {
	t.Helper()

	for _, id := range ids {
		if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id); err != nil {
			t.Errorf("cleanup: failed to delete test user %q: %v", id, err)
		}
	}
}

func createRLSTestAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, id, name string) {
	t.Helper()

	if err := WithUserTx(ctx, pool, userID, func(q *dbgen.Queries) error {
		_, err := q.CreateAccount(ctx, dbgen.CreateAccountParams{
			ID:     id,
			Name:   name,
			UserID: pgtype.UUID{Bytes: [16]byte(userID), Valid: true},
		})
		return err
	}); err != nil {
		t.Fatalf("seed account %q for user %s via WithUserTx: %v", id, userID, err)
	}
}

// TestWithUserTx_OwnerRoundTrip_LiveDatabase proves WithUserTx's binding
// survives the round trip through a real dbgen query: data written inside
// one WithUserTx call for user A is readable inside a later WithUserTx call
// for the same user, and invisible inside a WithUserTx call for a different
// user — even though GetAccount's own SQL already carries an ownership
// predicate (belt-and-suspenders, per the "SQL still includes ownership
// predicates even though RLS exists" invariant). The predicate-free proof
// that the RLS *policy* itself (not the predicate) is what blocks
// cross-user access lives in TestWithUserTx_PolicyBlocksCrossUserAccess_LiveDatabase
// below.
func TestWithUserTx_OwnerRoundTrip_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "WithUserTx owner round-trip test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	userA := createRLSTestUser(ctx, t, pool, "withusertx-owner-a")
	userB := createRLSTestUser(ctx, t, pool, "withusertx-owner-b")
	t.Cleanup(func() { deleteRLSTestUsers(ctx, t, pool, userA, userB) })

	accountID := "wut-owner-" + uuid.NewString()
	createRLSTestAccount(ctx, t, pool, userA, accountID, "WithUserTx Owner Account")

	var got dbgen.Account
	if err := WithUserTx(ctx, pool, userA, func(q *dbgen.Queries) error {
		var err error
		got, err = q.GetAccount(ctx, dbgen.GetAccountParams{
			ID:     accountID,
			UserID: pgtype.UUID{Bytes: [16]byte(userA), Valid: true},
		})
		return err
	}); err != nil {
		t.Fatalf("user A: expected to read own account via WithUserTx, got error: %v", err)
	}
	if got.ID != accountID {
		t.Errorf("user A: expected account id %q, got %q", accountID, got.ID)
	}

	err = WithUserTx(ctx, pool, userB, func(q *dbgen.Queries) error {
		_, err := q.GetAccount(ctx, dbgen.GetAccountParams{
			ID:     accountID,
			UserID: pgtype.UUID{Bytes: [16]byte(userB), Valid: true},
		})
		return err
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("user B: expected ErrNoRows reading user A's account via WithUserTx, got %v", err)
	}
}

// TestWithUserTx_PolicyBlocksCrossUserAccess_LiveDatabase proves isolation
// is enforced by the RLS POLICY itself, not by an application WHERE clause.
// WithUserTx's fn parameter only exposes *dbgen.Queries, and every
// generated dbgen query in this codebase deliberately carries its own
// ownership predicate, so proving the predicate-free case requires a raw
// statement issued directly on a transaction. This test opens its own
// transaction and issues the exact set_config('app.user_id', $1, true)
// statement WithUserTx runs as its first statement, then probes with SQL
// that has no user_id predicate at all — isolating the RLS policy as the
// only thing standing between user B and user A's row.
func TestWithUserTx_PolicyBlocksCrossUserAccess_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "WithUserTx policy-level cross-user test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	userA := createRLSTestUser(ctx, t, pool, "withusertx-policy-a")
	userB := createRLSTestUser(ctx, t, pool, "withusertx-policy-b")
	t.Cleanup(func() { deleteRLSTestUsers(ctx, t, pool, userA, userB) })

	accountID := "wut-policy-" + uuid.NewString()
	createRLSTestAccount(ctx, t, pool, userA, accountID, "WithUserTx Policy Account")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin probe transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("cleanup: failed to roll back probe transaction: %v", err)
		}
	}()

	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, userB.String()); err != nil {
		t.Fatalf("bind app.user_id for user B: %v", err)
	}

	var probeID string
	err = tx.QueryRow(ctx, `SELECT id FROM accounts WHERE id = $1`, accountID).Scan(&probeID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("user B: expected the RLS policy (not a WHERE clause) to hide user A's account, got err=%v", err)
	}

	tag, err := tx.Exec(ctx, `UPDATE accounts SET name = $1 WHERE id = $2`, "hacked-by-b", accountID)
	if err != nil {
		t.Fatalf("user B: UPDATE on user A's account returned an unexpected error: %v", err)
	}
	if tag.RowsAffected() != 0 {
		t.Errorf("user B: expected 0 rows affected updating user A's account, got %d", tag.RowsAffected())
	}

	tag, err = tx.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, accountID)
	if err != nil {
		t.Fatalf("user B: DELETE on user A's account returned an unexpected error: %v", err)
	}
	if tag.RowsAffected() != 0 {
		t.Errorf("user B: expected 0 rows affected deleting user A's account, got %d", tag.RowsAffected())
	}
}

// TestWithUserTx_UnboundTransactionFailsClosed_LiveDatabase pins the #43
// fail-closed property (design decision 4c): a transaction that never runs
// the set_config('app.user_id', ...) bind sees zero rows on an owned table
// instead of an error or every row. current_setting('app.user_id', true)
// (missing_ok=true) returns NULL when unset, and the policy's
// NULLIF(...)::uuid comparison against NULL matches nothing — silent, but
// closed. This is exactly the shape of the dbgen.New(pool)-without-a-
// transaction bug WithUserTx exists to make impossible: an owned-table
// query with no transaction-local binding in effect.
func TestWithUserTx_UnboundTransactionFailsClosed_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "WithUserTx fail-closed test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	userA := createRLSTestUser(ctx, t, pool, "withusertx-failclosed")
	t.Cleanup(func() { deleteRLSTestUsers(ctx, t, pool, userA) })

	accountID := "wut-failclosed-" + uuid.NewString()
	createRLSTestAccount(ctx, t, pool, userA, accountID, "WithUserTx Fail-Closed Account")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin unbound transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("cleanup: failed to roll back unbound transaction: %v", err)
		}
	}()

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM accounts WHERE id = $1`, accountID).Scan(&count); err != nil {
		t.Fatalf("unbound transaction: unexpected query error: %v", err)
	}
	if count != 0 {
		t.Errorf("unbound transaction: expected 0 visible rows (fail closed), got %d", count)
	}
}

// TestWithUserTx_RollsBackOnError_LiveDatabase proves fn's error both
// prevents the commit and is returned to the caller: a create that
// succeeds followed by a returned error must leave no row behind.
func TestWithUserTx_RollsBackOnError_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "WithUserTx rollback-on-error test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	userA := createRLSTestUser(ctx, t, pool, "withusertx-rollback")
	t.Cleanup(func() { deleteRLSTestUsers(ctx, t, pool, userA) })

	accountID := "wut-rollback-" + uuid.NewString()
	sentinel := errors.New("boom")

	err = WithUserTx(ctx, pool, userA, func(q *dbgen.Queries) error {
		if _, err := q.CreateAccount(ctx, dbgen.CreateAccountParams{
			ID:     accountID,
			Name:   "WithUserTx Rollback Account",
			UserID: pgtype.UUID{Bytes: [16]byte(userA), Valid: true},
		}); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected WithUserTx to return fn's error, got %v", err)
	}

	err = WithUserTx(ctx, pool, userA, func(q *dbgen.Queries) error {
		_, err := q.GetAccount(ctx, dbgen.GetAccountParams{
			ID:     accountID,
			UserID: pgtype.UUID{Bytes: [16]byte(userA), Valid: true},
		})
		return err
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected the account created before the returned error to have been rolled back, got %v", err)
	}
}
