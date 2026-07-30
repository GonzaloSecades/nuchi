package httpapi

import (
	"testing"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
)

// TestOrderedByRequest is what actually proves the reordering guarantee.
//
// The live happy-path test asserts response order too, but it cannot
// distinguish a working helper from a broken one: PostgreSQL currently tends to
// emit this simple RETURNING result in source order, so that test would stay
// green even if orderedByRequest were deleted. Feeding the helper a deliberately
// shuffled `created` slice is the only way to show the ordering comes from the
// code rather than from executor behavior we do not control.
func TestOrderedByRequest(t *testing.T) {
	requested := []bulkTransactionRow{
		{ID: "row-a", Payee: "Market"},
		{ID: "row-b", Payee: "Salary"},
		{ID: "row-c", Payee: "Rent"},
	}

	t.Run("reverses a shuffled database result back to request order", func(t *testing.T) {
		created := []dbgen.Transaction{
			{ID: "row-c", Payee: "Rent"},
			{ID: "row-a", Payee: "Market"},
			{ID: "row-b", Payee: "Salary"},
		}

		ordered := orderedByRequest(requested, created)
		if len(ordered) != len(requested) {
			t.Fatalf("expected %d rows, got %d", len(requested), len(ordered))
		}
		for i, want := range []string{"row-a", "row-b", "row-c"} {
			if ordered[i].Id != want {
				t.Errorf("position %d: expected %q, got %q", i, want, ordered[i].Id)
			}
		}
		if ordered[0].Payee != "Market" || ordered[2].Payee != "Rent" {
			t.Errorf("rows were reordered but their contents did not follow: %+v", ordered)
		}
	})

	t.Run("already-ordered input is unchanged", func(t *testing.T) {
		created := []dbgen.Transaction{
			{ID: "row-a"}, {ID: "row-b"}, {ID: "row-c"},
		}
		ordered := orderedByRequest(requested, created)
		for i, want := range []string{"row-a", "row-b", "row-c"} {
			if ordered[i].Id != want {
				t.Errorf("position %d: expected %q, got %q", i, want, ordered[i].Id)
			}
		}
	})

	// Unreachable in the handler — a cardinality mismatch is caught inside the
	// transaction and rolls it back before this runs — but pinned so the helper
	// can never emit a zero-valued transaction for a row the database did not
	// return.
	t.Run("a missing row is skipped, never zero-valued", func(t *testing.T) {
		created := []dbgen.Transaction{{ID: "row-c"}, {ID: "row-a"}}
		ordered := orderedByRequest(requested, created)
		if len(ordered) != 2 {
			t.Fatalf("expected the two returned rows, got %d", len(ordered))
		}
		for _, row := range ordered {
			if row.Id == "" {
				t.Error("a missing row must be skipped rather than emitted with a zero id")
			}
		}
	})

	t.Run("empty input yields no rows", func(t *testing.T) {
		if ordered := orderedByRequest(nil, nil); len(ordered) != 0 {
			t.Errorf("expected no rows, got %d", len(ordered))
		}
	})
}
