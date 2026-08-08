package db

import (
	"context"
	"fmt"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	return WithUserTxOptions(ctx, pool, userID, pgx.TxOptions{}, fn)
}

// WithUserTxOptions is WithUserTx with explicit transaction semantics.
// Callers that issue several related reads can request RepeatableRead so every
// statement observes one database snapshot while retaining the same mandatory
// transaction-local RLS binding.
func WithUserTxOptions(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	options pgx.TxOptions,
	fn func(*dbgen.Queries) error,
) error {
	tx, err := pool.BeginTx(ctx, options)
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

// ownedTablesRequiringRLS lists the tables whose row-level security must be
// active for the API's connection role before the service accepts traffic.
// Keep it in sync with the FORCE ROW LEVEL SECURITY tables in the finance
// migrations (00003, 00004).
var ownedTablesRequiringRLS = []string{"accounts", "categories", "transactions"}

// VerifyRLSActive returns an error unless row-level security is active, for
// the pool's connection role, on every owned table. Call it once at startup
// and refuse to serve if it fails.
//
// WithUserTx binds the right app.user_id, but a binding enforces nothing when
// the connection authenticates as a superuser or a role with BYPASSRLS: both
// silently ignore FORCE ROW LEVEL SECURITY, so every owned-table query would
// read or write across all tenants regardless of the binding. CI provisions an
// ordinary role and a fresh Compose volume does too, but production credentials
// are arbitrary and an older local volume may still have the app role as the
// bootstrap superuser — a single such misconfiguration silently turns this
// otherwise-correct request path into unrestricted cross-user access. This
// guard makes that fail loud at startup instead.
//
// row_security_active(table) reports whether RLS would actually be enforced for
// the current role on that table, and returns false precisely for the
// superuser/BYPASSRLS roles this guards against — a more direct signal than
// inspecting role attributes, which are surfaced only to make the error
// diagnosable.
func VerifyRLSActive(ctx context.Context, pool *pgxpool.Pool) error {
	for _, table := range ownedTablesRequiringRLS {
		var active bool
		if err := pool.QueryRow(ctx, `SELECT row_security_active($1::text)`, table).Scan(&active); err != nil {
			return fmt.Errorf("db: check row security for %q: %w", table, err)
		}
		if !active {
			role, super, bypass := connectionRoleAttributes(ctx, pool)
			return fmt.Errorf(
				"db: row level security is not active on %q for role %q (rolsuper=%t, rolbypassrls=%t); refusing to serve because every owned-table query would bypass RLS and read or write across tenants",
				table, role, super, bypass,
			)
		}
	}
	return nil
}

// connectionRoleAttributes fetches the current role name and its
// superuser/bypassrls attributes for diagnostics. Best-effort: on error it
// returns a placeholder name so the caller's error still names the failing
// table.
func connectionRoleAttributes(ctx context.Context, pool *pgxpool.Pool) (role string, super, bypass bool) {
	role = "unknown"
	_ = pool.QueryRow(ctx,
		`SELECT current_user, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user`,
	).Scan(&role, &super, &bypass)
	return role, super, bypass
}
