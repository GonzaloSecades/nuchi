package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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

// TestWithUserTx_CancelledRequestRollsBackAndReleasesConnection_LiveDatabase
// proves transaction cleanup does not reuse the cancelled request context.
// A one-connection pool makes a leaked in-transaction connection observable:
// the follow-up read would time out instead of acquiring the connection.
func TestWithUserTx_CancelledRequestRollsBackAndReleasesConnection_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "WithUserTx cancelled-request cleanup test")

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database config: %v", err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("create one-connection pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping one-connection pool: %v", err)
	}

	userID := createRLSTestUser(ctx, t, pool, "withusertx-cancel")
	t.Cleanup(func() { deleteRLSTestUsers(ctx, t, pool, userID) })
	owner := pgtype.UUID{Bytes: [16]byte(userID), Valid: true}
	accountID := "wut-cancel-" + uuid.NewString()

	requestCtx, cancelRequest := context.WithCancel(ctx)
	err = WithUserTx(requestCtx, pool, userID, func(q *dbgen.Queries) error {
		if _, err := q.CreateAccount(requestCtx, dbgen.CreateAccountParams{
			ID: accountID, Name: "Cancelled Request Account", UserID: owner,
		}); err != nil {
			return err
		}
		cancelRequest()
		return requestCtx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	followUpCtx, cancelFollowUp := context.WithTimeout(ctx, 2*time.Second)
	defer cancelFollowUp()
	err = WithUserTx(followUpCtx, pool, userID, func(q *dbgen.Queries) error {
		_, err := q.GetAccount(followUpCtx, dbgen.GetAccountParams{ID: accountID, UserID: owner})
		return err
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected cancelled transaction to roll back and release its connection, got %v", err)
	}
}

// TestWithUserTxOptions_RepeatableReadUsesOneSnapshot_LiveDatabase proves the
// option used by the summary handler, rather than merely checking a constant.
// A row committed between two reads must stay outside the reader's snapshot
// until that repeatable-read transaction ends.
func TestWithUserTxOptions_RepeatableReadUsesOneSnapshot_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "WithUserTxOptions repeatable-read test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	userID := createRLSTestUser(ctx, t, pool, "withusertx-repeatable-read")
	t.Cleanup(func() { deleteRLSTestUsers(ctx, t, pool, userID) })

	owner := pgtype.UUID{Bytes: [16]byte(userID), Valid: true}
	accountID := "wut-repeatable-" + uuid.NewString()

	err = WithUserTxOptions(ctx, pool, userID, pgx.TxOptions{IsoLevel: pgx.RepeatableRead}, func(q *dbgen.Queries) error {
		before, err := q.ListAccounts(ctx, owner)
		if err != nil {
			return fmt.Errorf("first snapshot read: %w", err)
		}
		if len(before) != 0 {
			return fmt.Errorf("first snapshot read: expected no accounts, got %d", len(before))
		}

		writeDone := make(chan error, 1)
		go func() {
			writeDone <- WithUserTx(ctx, pool, userID, func(writeQueries *dbgen.Queries) error {
				_, err := writeQueries.CreateAccount(ctx, dbgen.CreateAccountParams{
					ID: accountID, Name: "Committed Between Reads", UserID: owner,
				})
				return err
			})
		}()
		if err := <-writeDone; err != nil {
			return fmt.Errorf("concurrent account insert: %w", err)
		}

		after, err := q.ListAccounts(ctx, owner)
		if err != nil {
			return fmt.Errorf("second snapshot read: %w", err)
		}
		if len(after) != 0 {
			return fmt.Errorf("repeatable-read snapshot changed: expected 0 accounts, got %d", len(after))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("repeatable-read transaction: %v", err)
	}

	err = WithUserTx(ctx, pool, userID, func(q *dbgen.Queries) error {
		accounts, err := q.ListAccounts(ctx, owner)
		if err != nil {
			return err
		}
		if len(accounts) != 1 || accounts[0].ID != accountID {
			return fmt.Errorf("expected committed account %q after snapshot ended, got %+v", accountID, accounts)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("post-snapshot read: %v", err)
	}
}

// TestVerifyRLSActive_PassesForOrdinaryRole_LiveDatabase is the positive half
// of the startup guard: the ordinary application role the migrations run as
// has RLS active on every owned table, so VerifyRLSActive accepts it.
func TestVerifyRLSActive_PassesForOrdinaryRole_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "VerifyRLSActive ordinary-role test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := VerifyRLSActive(ctx, pool); err != nil {
		t.Fatalf("expected VerifyRLSActive to pass for the ordinary application role, got %v", err)
	}
}

// TestVerifyRLSActive_RejectsBypassingRole_LiveDatabase is the negative half:
// a connection authenticating as a role that bypasses RLS (the bootstrap
// superuser, reached via ADMIN_DATABASE_URL) silently voids FORCE RLS, so
// VerifyRLSActive must refuse it. This is the exact misconfiguration the guard
// exists to stop — a deployment whose DATABASE_URL points at a superuser would
// otherwise serve every tenant's data to every request.
func TestVerifyRLSActive_RejectsBypassingRole_LiveDatabase(t *testing.T) {
	databaseURL := adminDatabaseURL(t, "VerifyRLSActive bypassing-role test")

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	err = VerifyRLSActive(ctx, pool)
	if err == nil {
		t.Fatal("expected VerifyRLSActive to reject a role that bypasses row level security, got nil")
	}
	if !strings.Contains(err.Error(), "row level security is not active") {
		t.Errorf("expected the rejection to explain the RLS bypass, got %v", err)
	}
}
