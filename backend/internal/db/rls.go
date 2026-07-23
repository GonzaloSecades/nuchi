package db

import (
	"context"
	"fmt"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithUserTx begins a transaction on pool, binds the RLS user for the
// lifetime of that transaction, and runs fn against a *dbgen.Queries bound
// to the transaction. It commits on success and rolls back on error or
// panic (a panic inside fn propagates after the deferred rollback runs,
// same as every hand-rolled tx pattern elsewhere in this codebase).
//
// The bind is transaction-local — set_config('app.user_id', $1, true), with
// is_local=true as the third argument — not session-level. pgxpool reuses
// connections across requests, so a session-level (is_local=false) bind
// would outlive this request on its pooled connection and leak this user's
// identity into whichever unrelated request grabs that connection next: a
// cross-user data breach. Transaction-local scope resets automatically at
// COMMIT or ROLLBACK, so there is nothing to leak. The user id is always
// passed as a bind parameter (userID.String(), since set_config's value
// argument is text), never string-interpolated into SQL.
//
// This is the ONLY sanctioned way to touch owned tables (accounts,
// categories, transactions, and anything summarizing them) from here on.
// dbgen.New(pool) called directly against the pool, outside any
// transaction, has no transaction for a transaction-local set_config to
// attach to — the RLS policy then sees a NULL app.user_id and silently
// returns zero rows instead of erroring. That is not a security hole (fail
// closed still fails closed), but it is a silent-empty-result bug that is
// easy to ship by accident, which is why every owned-resource query must
// route through this helper instead.
func WithUserTx(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, fn func(*dbgen.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin user tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, userID.String()); err != nil {
		return fmt.Errorf("db: bind rls user: %w", err)
	}

	q := dbgen.New(tx)
	if err := fn(q); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit user tx: %w", err)
	}
	committed = true

	return nil
}
