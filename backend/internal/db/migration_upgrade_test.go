package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestMigration00004_NullsPreexistingCrossOwnerCategory_LiveDatabase exercises
// the data-migration path in 00004 that the post-v4 policy tests cannot reach:
// a transaction that already references another tenant's category (legal under
// the v3 account-only policy) must have that category_id nulled when v4 is
// applied. It regresses the bug where the cleanup lifted FORCE RLS only on the
// UPDATE target (transactions) and not on the joined accounts/categories, so
// the join saw zero rows and the cleanup silently did nothing.
//
// It runs the real migrations with the goose binary against a throwaway
// database, bootstrapped exactly like CI (public schema owned by the ordinary
// app role, citext installed by the superuser) so FORCE RLS genuinely applies
// to the migration's own role. The throwaway database keeps it from disturbing
// the shared TEST_DATABASE_URL migration state that other packages test
// against in parallel.
func TestMigration00004_NullsPreexistingCrossOwnerCategory_LiveDatabase(t *testing.T) {
	appURL := liveDatabaseURL(t, "00004 upgrade test (app role)")
	adminURL := adminDatabaseURL(t, "00004 upgrade test (admin role)")
	gooseBin := gooseBinary(t)

	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}

	appRole := userInURL(t, appURL)
	tempDB := "nuchi_migtest_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	tempAppURL := withDatabase(t, appURL, tempDB)
	tempAdminURL := withDatabase(t, adminURL, tempDB)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the throwaway database and bootstrap it like CI: hand the public
	// schema to the app role so the migrations it runs own their tables (which
	// is what makes FORCE RLS apply to that role), and install citext as the
	// superuser so 00001's CREATE EXTENSION is a no-op regardless of whether
	// citext is a trusted extension here.
	adminConn := connect(ctx, t, adminURL)
	if _, err := adminConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgIdent(tempDB))); err != nil {
		adminConn.Close(ctx)
		t.Fatalf("create throwaway database: %v", err)
	}
	adminConn.Close(ctx)

	// Drop the throwaway database last (deferred first => runs last, after every
	// connection below is closed). WITH (FORCE) evicts any lingering backend.
	defer func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dropCancel()
		cleanupConn, err := pgx.Connect(dropCtx, adminURL)
		if err != nil {
			t.Errorf("cleanup: connect to drop throwaway database: %v", err)
			return
		}
		defer cleanupConn.Close(dropCtx)
		if _, err := cleanupConn.Exec(dropCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgIdent(tempDB))); err != nil {
			t.Errorf("cleanup: drop throwaway database %q: %v", tempDB, err)
		}
	}()

	bootstrapConn := connect(ctx, t, tempAdminURL)
	for _, stmt := range []string{
		fmt.Sprintf("ALTER SCHEMA public OWNER TO %s", pgIdent(appRole)),
		"CREATE EXTENSION IF NOT EXISTS citext",
	} {
		if _, err := bootstrapConn.Exec(ctx, stmt); err != nil {
			bootstrapConn.Close(ctx)
			t.Fatalf("bootstrap throwaway database (%q): %v", stmt, err)
		}
	}
	bootstrapConn.Close(ctx)

	// Migrate to v3 (pre-fix policy), as the app role.
	runGoose(ctx, t, gooseBin, migrationsDir, tempAppURL, "up-to", "3")

	// Seed a committed cross-owner reference the v3 policy still allows: user A's
	// transaction on user A's account pointing at user B's category. Runs on a
	// single connection so the session-level app.user_id set between inserts
	// sticks. The v3 transactions_owner WITH CHECK only proves account ownership,
	// so this insert is accepted; the FK to user B's category is satisfied
	// because FK checks bypass RLS.
	seedConn := connect(ctx, t, tempAppURL)
	userA := insertUpgradeUser(ctx, t, seedConn, "migup-a")
	userB := insertUpgradeUser(ctx, t, seedConn, "migup-b")
	setSessionAppUser(ctx, t, seedConn, userA)
	execUpgrade(ctx, t, seedConn, `INSERT INTO accounts (id, name, user_id) VALUES ($1,$2,$3)`, "migup-acct-a", "Mig Acct A", userA)
	execUpgrade(ctx, t, seedConn, `INSERT INTO categories (id, name, user_id) VALUES ($1,$2,$3)`, "migup-cat-a", "Mig Cat A", userA)
	setSessionAppUser(ctx, t, seedConn, userB)
	execUpgrade(ctx, t, seedConn, `INSERT INTO categories (id, name, user_id) VALUES ($1,$2,$3)`, "migup-cat-b", "Mig Cat B", userB)
	setSessionAppUser(ctx, t, seedConn, userA)
	execUpgrade(ctx, t, seedConn,
		`INSERT INTO transactions (id, amount, payee, date, account_id, category_id, currency) VALUES ($1,$2,$3,$4,$5,$6,'ARS')`,
		"migup-txn", 100, "mig payee", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "migup-acct-a", "migup-cat-b")

	// Confirm the seed really is cross-owner before the upgrade, so a later NULL
	// can only be the cleanup's doing.
	var before *string
	if err := seedConn.QueryRow(ctx, `SELECT category_id FROM transactions WHERE id = $1`, "migup-txn").Scan(&before); err != nil {
		t.Fatalf("read seeded transaction: %v", err)
	}
	if before == nil || *before != "migup-cat-b" {
		t.Fatalf("expected seeded category_id 'migup-cat-b' before upgrade, got %v", before)
	}
	seedConn.Close(ctx)

	// Apply v4 (the fix).
	runGoose(ctx, t, gooseBin, migrationsDir, tempAppURL, "up")

	// The cleanup must have nulled the cross-owner reference. Read it back over a
	// superuser connection so RLS cannot hide the row and mask the result.
	verifyConn := connect(ctx, t, tempAdminURL)
	defer verifyConn.Close(ctx)
	var after *string
	if err := verifyConn.QueryRow(ctx, `SELECT category_id FROM transactions WHERE id = $1`, "migup-txn").Scan(&after); err != nil {
		t.Fatalf("read transaction after upgrade: %v", err)
	}
	if after != nil {
		t.Fatalf("expected the pre-existing cross-owner category_id to be nulled by 00004, got %q", *after)
	}
}

// gooseBinary returns the path to the goose CLI, mirroring the URL helpers'
// skip-local / fail-in-CI contract: CI installs goose before the tests, so a
// missing binary there is a broken workflow rather than an untested path.
func gooseBinary(t *testing.T) string {
	t.Helper()

	path, err := exec.LookPath("goose")
	if err == nil {
		return path
	}
	if os.Getenv("CI") != "" {
		t.Fatalf("goose binary not found in CI; the 00004 upgrade test must run there: %v", err)
	}
	t.Skip("goose binary not found; skipping 00004 upgrade test")
	return ""
}

func runGoose(ctx context.Context, t *testing.T, gooseBin, dir, databaseURL string, args ...string) {
	t.Helper()

	full := append([]string{"-dir", dir, "postgres", databaseURL}, args...)
	cmd := exec.CommandContext(ctx, gooseBin, full...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("goose %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func connect(ctx context.Context, t *testing.T, databaseURL string) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return conn
}

func insertUpgradeUser(ctx context.Context, t *testing.T, conn *pgx.Conn, label string) string {
	t.Helper()

	var id string
	email := fmt.Sprintf("%s-%s@example.test", label, uuid.NewString())
	if err := conn.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, 'test-hash') RETURNING id`, email,
	).Scan(&id); err != nil {
		t.Fatalf("insert upgrade user %q: %v", label, err)
	}
	return id
}

func setSessionAppUser(ctx context.Context, t *testing.T, conn *pgx.Conn, userID string) {
	t.Helper()

	// Session-level (is_local=false) so it persists across the autocommit
	// inserts that follow on this same connection.
	if _, err := conn.Exec(ctx, `SELECT set_config('app.user_id', $1, false)`, userID); err != nil {
		t.Fatalf("set session app.user_id: %v", err)
	}
}

func execUpgrade(ctx context.Context, t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()

	if _, err := conn.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("exec seed statement: %v", err)
	}
}

// userInURL extracts the role name from a postgres connection URL.
func userInURL(t *testing.T, databaseURL string) string {
	t.Helper()

	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	if u.User == nil || u.User.Username() == "" {
		t.Fatalf("database url has no user: %q", databaseURL)
	}
	return u.User.Username()
}

// withDatabase returns databaseURL with its database name replaced by name.
func withDatabase(t *testing.T, databaseURL, name string) string {
	t.Helper()

	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// pgIdent quotes a SQL identifier. The names it guards (a generated temp
// database, the app role parsed from a trusted env var) are not attacker
// controlled, but identifiers cannot be bind parameters, so quoting is the
// correct way to interpolate them.
func pgIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// TestMigration00005_PreservesAmountsAndLiftsInt32Cap_LiveDatabase proves the
// int4 -> int8 widening is value-preserving for rows that already exist, and
// that the new column actually accepts amounts the old one could not.
//
// The risk this guards is silent corruption: if the widening were ever written
// with a lossy USING clause (or reverted carelessly), a large positive income
// could wrap into a negative expense with no error anywhere. Seeding the exact
// int4 boundaries is the sharpest available check, since those are the values a
// wraparound bug would mangle most visibly.
func TestMigration00005_PreservesAmountsAndLiftsInt32Cap_LiveDatabase(t *testing.T) {
	appURL := liveDatabaseURL(t, "00005 upgrade test (app role)")
	adminURL := adminDatabaseURL(t, "00005 upgrade test (admin role)")
	gooseBin := gooseBinary(t)

	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}

	appRole := userInURL(t, appURL)
	tempDB := "nuchi_migtest_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	tempAppURL := withDatabase(t, appURL, tempDB)
	tempAdminURL := withDatabase(t, adminURL, tempDB)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adminConn := connect(ctx, t, adminURL)
	if _, err := adminConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgIdent(tempDB))); err != nil {
		adminConn.Close(ctx)
		t.Fatalf("create throwaway database: %v", err)
	}
	adminConn.Close(ctx)

	defer func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dropCancel()
		cleanupConn, err := pgx.Connect(dropCtx, adminURL)
		if err != nil {
			t.Errorf("cleanup: connect to drop throwaway database: %v", err)
			return
		}
		defer cleanupConn.Close(dropCtx)
		if _, err := cleanupConn.Exec(dropCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgIdent(tempDB))); err != nil {
			t.Errorf("cleanup: drop throwaway database %q: %v", tempDB, err)
		}
	}()

	bootstrapConn := connect(ctx, t, tempAdminURL)
	for _, stmt := range []string{
		fmt.Sprintf("ALTER SCHEMA public OWNER TO %s", pgIdent(appRole)),
		"CREATE EXTENSION IF NOT EXISTS citext",
	} {
		if _, err := bootstrapConn.Exec(ctx, stmt); err != nil {
			bootstrapConn.Close(ctx)
			t.Fatalf("bootstrap throwaway database (%q): %v", stmt, err)
		}
	}
	bootstrapConn.Close(ctx)

	// Migrate to v4: amount is still int4 here.
	runGoose(ctx, t, gooseBin, migrationsDir, tempAppURL, "up-to", "4")

	seedConn := connect(ctx, t, tempAppURL)
	userID := insertUpgradeUser(ctx, t, seedConn, "migup5")
	setSessionAppUser(ctx, t, seedConn, userID)
	execUpgrade(ctx, t, seedConn, `INSERT INTO accounts (id, name, user_id) VALUES ($1,$2,$3)`, "mig5-acct", "Mig5 Acct", userID)

	// The int4 extremes plus an ordinary positive/negative pair and zero.
	seeded := map[string]int64{
		"mig5-max":      2147483647,
		"mig5-min":      -2147483648,
		"mig5-positive": 500000,
		"mig5-negative": -12500,
		"mig5-zero":     0,
	}
	for id, amount := range seeded {
		execUpgrade(ctx, t, seedConn,
			`INSERT INTO transactions (id, amount, payee, date, account_id, currency) VALUES ($1,$2,$3,$4,$5,'ARS')`,
			id, amount, "mig5 payee", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "mig5-acct")
	}
	seedConn.Close(ctx)

	// Apply 00005.
	runGoose(ctx, t, gooseBin, migrationsDir, tempAppURL, "up")

	verifyConn := connect(ctx, t, tempAdminURL)
	defer verifyConn.Close(ctx)

	var dataType string
	if err := verifyConn.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns WHERE table_name = 'transactions' AND column_name = 'amount'`,
	).Scan(&dataType); err != nil {
		t.Fatalf("read amount column type: %v", err)
	}
	if dataType != "bigint" {
		t.Fatalf("expected amount to be bigint after 00005, got %q", dataType)
	}

	// Every seeded value survives byte for byte, sign included.
	for id, want := range seeded {
		var got int64
		if err := verifyConn.QueryRow(ctx, `SELECT amount FROM transactions WHERE id = $1`, id).Scan(&got); err != nil {
			t.Fatalf("read %q after upgrade: %v", id, err)
		}
		if got != want {
			t.Errorf("%s: amount changed across the widening: want %d, got %d", id, want, got)
		}
	}

	// The point of the migration: an amount the int4 column could never hold.
	// 3_000_000_000 milliunits is 3,000,000 ARS — roughly USD 2,000, an
	// unremarkable rent or medical bill, and impossible before this change.
	setSessionAppUser(ctx, t, verifyConn, userID)
	const beyondInt32 = int64(3_000_000_000)
	execUpgrade(ctx, t, verifyConn,
		`INSERT INTO transactions (id, amount, payee, date, account_id, currency) VALUES ($1,$2,$3,$4,$5,'ARS')`,
		"mig5-beyond", beyondInt32, "mig5 payee", time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), "mig5-acct")

	var beyond int64
	if err := verifyConn.QueryRow(ctx, `SELECT amount FROM transactions WHERE id = $1`, "mig5-beyond").Scan(&beyond); err != nil {
		t.Fatalf("read beyond-int32 amount: %v", err)
	}
	if beyond != beyondInt32 {
		t.Errorf("beyond-int32 amount did not round-trip: want %d, got %d", beyondInt32, beyond)
	}
}
