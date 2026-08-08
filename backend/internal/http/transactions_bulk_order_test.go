package httpapi

import (
	"testing"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
)

// TestMatchCreatedToRequested is what actually proves the reordering guarantee
// and the set invariant behind it.
//
// The live happy-path test asserts response order too, but it cannot distinguish
// a working helper from a broken one: PostgreSQL currently tends to emit this
// simple RETURNING result in source order, so that test would stay green even if
// the matching were deleted. Feeding the helper a deliberately shuffled result
// is the only way to show the ordering comes from the code rather than from
// executor behavior we do not control.
func TestMatchCreatedToRequested(t *testing.T) {
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

		ordered, err := matchCreatedToRequested(requested, created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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
		created := []dbgen.Transaction{{ID: "row-a"}, {ID: "row-b"}, {ID: "row-c"}}
		ordered, err := matchCreatedToRequested(requested, created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, want := range []string{"row-a", "row-b", "row-c"} {
			if ordered[i].Id != want {
				t.Errorf("position %d: expected %q, got %q", i, want, ordered[i].Id)
			}
		}
	})

	// The case a length-only check could never catch, and the reason this
	// function verifies the id set rather than the row count. Three rows in,
	// three rows back, one of them an id nobody asked for. The old check passed
	// this and the ordering step then silently dropped row-b, producing a
	// committed 200 that was short by one row.
	t.Run("equal length with a wrong id is an error, not a short result", func(t *testing.T) {
		created := []dbgen.Transaction{
			{ID: "row-a"},
			{ID: "row-c"},
			{ID: "row-IMPOSTOR"},
		}
		ordered, err := matchCreatedToRequested(requested, created)
		if err == nil {
			t.Fatalf("expected an error, got %d rows: %+v", len(ordered), ordered)
		}
		if ordered != nil {
			t.Errorf("expected no rows alongside the error, got %+v", ordered)
		}
	})

	t.Run("a missing row is an error", func(t *testing.T) {
		created := []dbgen.Transaction{{ID: "row-c"}, {ID: "row-a"}}
		if _, err := matchCreatedToRequested(requested, created); err == nil {
			t.Error("expected a short result to be rejected")
		}
	})

	t.Run("an unrequested extra row is an error", func(t *testing.T) {
		created := []dbgen.Transaction{
			{ID: "row-a"}, {ID: "row-b"}, {ID: "row-c"}, {ID: "row-extra"},
		}
		if _, err := matchCreatedToRequested(requested, created); err == nil {
			t.Error("expected an unrequested id to be rejected")
		}
	})

	// A duplicate would otherwise let one returned row satisfy two requested
	// rows, so the count could match while a real row went missing.
	t.Run("a duplicated id is an error", func(t *testing.T) {
		created := []dbgen.Transaction{{ID: "row-a"}, {ID: "row-a"}, {ID: "row-c"}}
		if _, err := matchCreatedToRequested(requested, created); err == nil {
			t.Error("expected a duplicate id to be rejected")
		}
	})

	t.Run("empty input yields no rows and no error", func(t *testing.T) {
		ordered, err := matchCreatedToRequested(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ordered) != 0 {
			t.Errorf("expected no rows, got %d", len(ordered))
		}
	})
}
