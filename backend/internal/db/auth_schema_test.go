package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAuthBaseSchema_LiveDatabase asserts that the auth base migration
// (backend/migrations/00001_auth_base.sql) has produced the expected
// tables, columns, nullability, uniqueness, and foreign-key shape. It is
// skipped unless TEST_DATABASE_URL is set, so CI (which does not run
// migrations) stays green without a live database.
//
// This test assumes migrations have already been applied to the target
// database (e.g. via `goose -dir migrations postgres "$TEST_DATABASE_URL" up`
// run from backend/); it does not apply them itself.
func TestAuthBaseSchema_LiveDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live database schema test")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	defer pool.Close()

	type columnSpec struct {
		name     string
		udtName  string
		nullable bool
	}

	tables := []struct {
		table   string
		columns []columnSpec
	}{
		{
			table: "users",
			columns: []columnSpec{
				{"id", "uuid", false},
				{"email", "citext", false},
				{"password_hash", "text", false},
				{"email_verified_at", "timestamptz", true},
				{"created_at", "timestamptz", false},
				{"updated_at", "timestamptz", false},
			},
		},
		{
			table: "email_verification_tokens",
			columns: []columnSpec{
				{"id", "uuid", false},
				{"user_id", "uuid", false},
				{"token_hash", "text", false},
				{"expires_at", "timestamptz", false},
				{"used_at", "timestamptz", true},
				{"created_at", "timestamptz", false},
			},
		},
		{
			table: "password_reset_tokens",
			columns: []columnSpec{
				{"id", "uuid", false},
				{"user_id", "uuid", false},
				{"token_hash", "text", false},
				{"expires_at", "timestamptz", false},
				{"used_at", "timestamptz", true},
				{"created_at", "timestamptz", false},
			},
		},
		{
			table: "refresh_tokens",
			columns: []columnSpec{
				{"id", "uuid", false},
				{"user_id", "uuid", false},
				{"token_hash", "text", false},
				{"expires_at", "timestamptz", false},
				{"revoked_at", "timestamptz", true},
				{"created_at", "timestamptz", false},
			},
		},
	}

	for _, tbl := range tables {
		for _, col := range tbl.columns {
			var udtName string
			var isNullable string
			row := pool.QueryRow(ctx, `
				SELECT udt_name, is_nullable
				FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
			`, tbl.table, col.name)
			if err := row.Scan(&udtName, &isNullable); err != nil {
				t.Fatalf("%s.%s: expected column to exist, got error: %v", tbl.table, col.name, err)
			}
			if udtName != col.udtName {
				t.Errorf("%s.%s: expected type %q, got %q", tbl.table, col.name, col.udtName, udtName)
			}
			gotNullable := isNullable == "YES"
			if gotNullable != col.nullable {
				t.Errorf("%s.%s: expected nullable=%v, got %v", tbl.table, col.name, col.nullable, gotNullable)
			}
		}
	}

	// users.email must be unique (in addition to being citext, checked above).
	assertUniqueColumn(ctx, t, pool, "users", "email")

	// Token tables must have a unique token_hash and an FK to users(id) with
	// ON DELETE CASCADE.
	for _, table := range []string{"email_verification_tokens", "password_reset_tokens", "refresh_tokens"} {
		assertUniqueColumn(ctx, t, pool, table, "token_hash")
		assertCascadingUserFK(ctx, t, pool, table)
	}
}

// assertUniqueColumn fails the test unless table has a UNIQUE constraint
// covering exactly the given column.
func assertUniqueColumn(ctx context.Context, t *testing.T, pool *pgxpool.Pool, table, column string) {
	t.Helper()

	var count int
	row := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'UNIQUE'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
			AND kcu.column_name = $2
	`, table, column)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("%s.%s: failed to query unique constraint: %v", table, column, err)
	}
	if count == 0 {
		t.Errorf("%s.%s: expected a UNIQUE constraint, found none", table, column)
	}
}

// assertCascadingUserFK fails the test unless table has a foreign key on
// user_id referencing users(id) with ON DELETE CASCADE.
func assertCascadingUserFK(ctx context.Context, t *testing.T, pool *pgxpool.Pool, table string) {
	t.Helper()

	var deleteRule string
	row := pool.QueryRow(ctx, `
		SELECT rc.delete_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_name = rc.constraint_name
			AND tc.table_schema = rc.constraint_schema
		JOIN information_schema.constraint_column_usage ccu
			ON rc.unique_constraint_name = ccu.constraint_name
			AND rc.unique_constraint_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
			AND kcu.column_name = 'user_id'
			AND ccu.table_name = 'users'
			AND ccu.column_name = 'id'
	`, table)
	if err := row.Scan(&deleteRule); err != nil {
		t.Fatalf("%s.user_id: failed to query FK to users(id): %v", table, err)
	}
	if deleteRule != "CASCADE" {
		t.Errorf("%s.user_id: expected ON DELETE CASCADE, got %q", table, deleteRule)
	}
}
