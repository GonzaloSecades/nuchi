package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestFinanceBaseSchema_LiveDatabase asserts that the finance base migration
// (backend/migrations/00002_finance_base.sql) has produced the expected
// tables, columns, nullability, uniqueness, foreign-key, and index shape. It
// runs against the postgres service CI provisions, and is skipped only
// when TEST_DATABASE_URL is unset outside CI (see liveDatabaseURL).
//
// This test assumes migrations have already been applied to the target
// database (e.g. via `goose -dir migrations postgres "$TEST_DATABASE_URL" up`
// run from backend/); it does not apply them itself.
func TestFinanceBaseSchema_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "live database schema test")

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
			table: "accounts",
			columns: []columnSpec{
				{"id", "text", false},
				{"plaid_id", "text", true},
				{"name", "citext", false},
				{"user_id", "uuid", false},
			},
		},
		{
			table: "categories",
			columns: []columnSpec{
				{"id", "text", false},
				{"plaid_id", "text", true},
				{"name", "citext", false},
				{"user_id", "uuid", false},
			},
		},
		{
			table: "transactions",
			columns: []columnSpec{
				{"id", "text", false},
				{"amount", "int4", false},
				{"payee", "text", false},
				{"notes", "text", true},
				{"date", "timestamp", false},
				{"account_id", "text", false},
				{"category_id", "text", true},
				{"currency", "text", false},
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

	// accounts/categories have a unique (user_id, name) index (citext makes
	// name comparisons case-insensitive).
	assertUniqueIndexColumns(ctx, t, pool, "accounts", "accounts_user_id_name_uniq", []string{"user_id", "name"})
	assertUniqueIndexColumns(ctx, t, pool, "categories", "categories_user_id_name_uniq", []string{"user_id", "name"})

	// FK delete rules.
	assertFK(ctx, t, pool, "accounts", "user_id", "users", "id", "CASCADE")
	assertFK(ctx, t, pool, "categories", "user_id", "users", "id", "CASCADE")
	assertFK(ctx, t, pool, "transactions", "account_id", "accounts", "id", "CASCADE")
	assertFK(ctx, t, pool, "transactions", "category_id", "categories", "id", "SET NULL")

	// transactions.currency defaults to 'ARS'.
	var currencyDefault string
	row := pool.QueryRow(ctx, `
		SELECT column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'transactions' AND column_name = 'currency'
	`)
	if err := row.Scan(&currencyDefault); err != nil {
		t.Fatalf("transactions.currency: expected column_default to be queryable, got error: %v", err)
	}
	if currencyDefault != "'ARS'::text" {
		t.Errorf("transactions.currency: expected default 'ARS'::text, got %q", currencyDefault)
	}

	// Composite index serving the hot list query (per-account date-range scan,
	// newest-first). Asserting the full definition pins the column order and
	// DESC direction, not just the name.
	assertIndexDefinition(ctx, t, pool, "transactions", "transactions_account_id_date_idx",
		"CREATE INDEX transactions_account_id_date_idx ON public.transactions USING btree (account_id, date DESC)")
	assertIndexDefinition(ctx, t, pool, "transactions", "transactions_category_id_idx",
		"CREATE INDEX transactions_category_id_idx ON public.transactions USING btree (category_id)")
}

// assertUniqueIndexColumns fails the test unless table has a unique index
// with the given name covering exactly the given ordered columns.
func assertUniqueIndexColumns(ctx context.Context, t *testing.T, pool *pgxpool.Pool, table, indexName string, columns []string) {
	t.Helper()

	rows, err := pool.Query(ctx, `
		SELECT a.attname
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_class tbl ON tbl.oid = ix.indrelid
		JOIN pg_attribute a ON a.attrelid = tbl.oid AND a.attnum = ANY(ix.indkey)
		WHERE tbl.relname = $1 AND i.relname = $2 AND ix.indisunique
		ORDER BY array_position(ix.indkey, a.attnum)
	`, table, indexName)
	if err != nil {
		t.Fatalf("%s.%s: failed to query unique index: %v", table, indexName, err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("%s.%s: failed to scan index column: %v", table, indexName, err)
		}
		got = append(got, col)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("%s.%s: rows error: %v", table, indexName, err)
	}

	if len(got) != len(columns) {
		t.Fatalf("%s.%s: expected unique index on %v, got %v", table, indexName, columns, got)
	}
	for i, col := range columns {
		if got[i] != col {
			t.Errorf("%s.%s: expected column %d to be %q, got %q", table, indexName, i, col, got[i])
		}
	}
}

// assertIndexDefinition fails the test unless table has an index with the
// given name whose full definition (columns, order, sort direction) matches
// wantDef exactly, as rendered by pg_indexes.indexdef.
func assertIndexDefinition(ctx context.Context, t *testing.T, pool *pgxpool.Pool, table, indexName, wantDef string) {
	t.Helper()

	var gotDef string
	row := pool.QueryRow(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = $1 AND indexname = $2
	`, table, indexName)
	if err := row.Scan(&gotDef); err != nil {
		t.Fatalf("%s.%s: expected index to exist, got error: %v", table, indexName, err)
	}
	if gotDef != wantDef {
		t.Errorf("%s.%s: index definition mismatch\nwant: %s\ngot:  %s", table, indexName, wantDef, gotDef)
	}
}

// assertFK fails the test unless table has a foreign key on column
// referencing refTable(refColumn) with the given delete rule.
func assertFK(ctx context.Context, t *testing.T, pool *pgxpool.Pool, table, column, refTable, refColumn, deleteRule string) {
	t.Helper()

	var gotRule string
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
			AND kcu.column_name = $2
			AND ccu.table_name = $3
			AND ccu.column_name = $4
	`, table, column, refTable, refColumn)
	if err := row.Scan(&gotRule); err != nil {
		t.Fatalf("%s.%s: failed to query FK to %s(%s): %v", table, column, refTable, refColumn, err)
	}
	if gotRule != deleteRule {
		t.Errorf("%s.%s: expected ON DELETE %s, got %q", table, column, deleteRule, gotRule)
	}
}
