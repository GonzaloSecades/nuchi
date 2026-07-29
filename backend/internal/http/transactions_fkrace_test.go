package httpapi

import (
	"errors"
	"testing"

	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestReferenceErrorFromForeignKeyViolation covers the gap between
// validateTransactionReferences and the write.
//
// The validation reads are unlocked SELECTs, so under READ COMMITTED a
// concurrent delete of the referenced account or category can commit between
// the check and the insert, and the write then fails with SQLSTATE 23503.
// Without the mapping under test the caller would see a generic 500 for what
// is really "that account no longer exists".
//
// This exercises the mapping directly rather than by racing two transactions:
// provoking the interleaving reliably needs a lock-step coordination point
// inside the handler that exists only to be tested, and a timing-dependent
// test that usually passes is worse than none. The classification is the part
// that can be wrong, so it is what gets pinned.
func TestReferenceErrorFromForeignKeyViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "account fk violation maps to the account reference error",
			err:  &pgconn.PgError{Code: "23503", ConstraintName: "transactions_account_id_fkey"},
			want: errAccountReferenceNotFound,
		},
		{
			name: "category fk violation maps to the category reference error",
			err:  &pgconn.PgError{Code: "23503", ConstraintName: "transactions_category_id_fkey"},
			want: errCategoryReferenceNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := referenceErrorFromForeignKeyViolation(tc.err); !errors.Is(got, tc.want) {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}

	// Anything that is not a transactions foreign-key violation must pass
	// through untouched, so a genuine database fault still surfaces as a 500
	// instead of being mislabelled a missing reference.
	t.Run("unrelated errors pass through", func(t *testing.T) {
		unrelated := []error{
			errors.New("connection reset"),
			&pgconn.PgError{Code: "23505", ConstraintName: "transactions_pkey"},
			&pgconn.PgError{Code: "23503", ConstraintName: "some_other_table_fkey"},
			&pgconn.PgError{Code: "40001"},
		}
		for _, err := range unrelated {
			got := referenceErrorFromForeignKeyViolation(err)
			if !errors.Is(got, err) {
				t.Errorf("expected %v to pass through unchanged, got %v", err, got)
			}
			if errors.Is(got, errAccountReferenceNotFound) || errors.Is(got, errCategoryReferenceNotFound) {
				t.Errorf("%v must not be reported as a missing reference", err)
			}
		}
	})
}

// TestForeignKeyConstraintNamesMatchTheSchema guards the mapping's only fragile
// assumption: it switches on PostgreSQL's auto-generated constraint names, so a
// migration that renames or re-creates either foreign key would silently turn
// the mapping back into a 500. Asserted against the live schema.
func TestForeignKeyConstraintNamesMatchTheSchema_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "transactions foreign-key name check")

	pool, err := db.NewPool(t.Context(), databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	for _, want := range []string{"transactions_account_id_fkey", "transactions_category_id_fkey"} {
		var exists bool
		err = pool.QueryRow(t.Context(),
			`SELECT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conrelid = 'transactions'::regclass AND contype = 'f' AND conname = $1
			)`, want).Scan(&exists)
		if err != nil {
			t.Fatalf("query constraint %q: %v", want, err)
		}
		if !exists {
			t.Errorf("foreign key %q is missing; referenceErrorFromForeignKeyViolation switches on this name and would stop mapping the race to a 404", want)
		}
	}
}
