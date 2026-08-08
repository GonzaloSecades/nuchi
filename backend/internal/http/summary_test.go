package httpapi

import (
	"math"
	"testing"
	"time"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestPercentageChange pins the zero rule, which is the part of this function
// that looks like a divide-by-zero guard and is actually a product decision:
// with no previous activity, legacy reports 0 when there is still none and 100
// when there is some, rather than an undefined or infinite ratio.
func TestPercentageChange(t *testing.T) {
	cases := []struct {
		name              string
		current, previous int64
		want              float32
	}{
		{"both zero is zero, not 100", 0, 0, 0},
		{"something from nothing is 100", 5000, 0, 100},
		{"negative something from nothing is still 100", -5000, 0, 100},
		{"doubling is +100", 2000, 1000, 100},
		{"halving is -50", 500, 1000, -50},
		{"unchanged is 0", 1000, 1000, 0},
		{"to zero is -100", 0, 1000, -100},
		// Sign of the previous period carries through the division, so a
		// negative baseline flips the direction. Legacy has the same behavior.
		{"negative previous", -500, -1000, -50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := percentageChange(tc.current, tc.previous); got != tc.want {
				t.Errorf("percentageChange(%d, %d) = %v, want %v", tc.current, tc.previous, got, tc.want)
			}
		})
	}
}

// TestPercentageChange_UsesFloatDivision guards against an integer-division
// port, which would silently truncate every fractional change to zero.
func TestPercentageChange_UsesFloatDivision(t *testing.T) {
	// 1000 -> 1005 is +0.5%, which integer division would report as 0.
	if got := percentageChange(1005, 1000); got != 0.5 {
		t.Errorf("expected 0.5, got %v — a truncating division would give 0", got)
	}
}

func TestPercentageChange_WidensBeforeSubtracting(t *testing.T) {
	// Subtracting in int64 wraps both of these cases and flips the sign before
	// the result reaches float64.
	if got := percentageChange(math.MaxInt64, -1); got >= 0 {
		t.Errorf("expected a negative change for MaxInt64 versus -1, got %v", got)
	}
	if got := percentageChange(math.MinInt64, 1); got >= 0 {
		t.Errorf("expected a negative change for MinInt64 versus 1, got %v", got)
	}
}

func categoryRow(name string, value int64) dbgen.GetCategorySpendingRow {
	return dbgen.GetCategorySpendingRow{Name: name, Value: value}
}

func TestBucketCategories(t *testing.T) {
	t.Run("fewer than the cap are returned as-is with no Other", func(t *testing.T) {
		got := bucketCategories([]dbgen.GetCategorySpendingRow{
			categoryRow("Rent", 900), categoryRow("Food", 500),
		})
		if len(got) != 2 {
			t.Fatalf("expected 2 categories, got %d: %+v", len(got), got)
		}
		for _, c := range got {
			if c.Name == otherCategoryName {
				t.Error("no Other bucket should appear when nothing was left over")
			}
		}
	})

	// The boundary: exactly the cap means nothing remains, so adding an empty
	// "Other" row here would be a visible defect in the chart.
	t.Run("exactly the cap produces no Other", func(t *testing.T) {
		got := bucketCategories([]dbgen.GetCategorySpendingRow{
			categoryRow("Rent", 900), categoryRow("Food", 500), categoryRow("Transport", 100),
		})
		if len(got) != maxSummaryCategories {
			t.Fatalf("expected %d categories, got %d: %+v", maxSummaryCategories, len(got), got)
		}
		if got[len(got)-1].Name == otherCategoryName {
			t.Error("exactly the cap must not produce an Other bucket")
		}
	})

	t.Run("one past the cap folds the remainder into Other", func(t *testing.T) {
		got := bucketCategories([]dbgen.GetCategorySpendingRow{
			categoryRow("Rent", 900), categoryRow("Food", 500),
			categoryRow("Transport", 100), categoryRow("Books", 40),
		})
		if len(got) != maxSummaryCategories+1 {
			t.Fatalf("expected %d entries, got %d: %+v", maxSummaryCategories+1, len(got), got)
		}
		last := got[len(got)-1]
		if last.Name != otherCategoryName || last.Value != 40 {
			t.Errorf("expected Other=40, got %+v", last)
		}
	})

	t.Run("Other sums every remaining category", func(t *testing.T) {
		got := bucketCategories([]dbgen.GetCategorySpendingRow{
			categoryRow("A", 900), categoryRow("B", 500), categoryRow("C", 100),
			categoryRow("D", 40), categoryRow("E", 30), categoryRow("F", 5),
		})
		last := got[len(got)-1]
		if last.Value != 75 {
			t.Errorf("expected Other to sum 40+30+5=75, got %d", last.Value)
		}
	})

	t.Run("no categories yields an empty slice, not nil", func(t *testing.T) {
		got := bucketCategories(nil)
		if got == nil {
			t.Fatal("expected an empty slice so it serializes as [], not null")
		}
		if len(got) != 0 {
			t.Errorf("expected no categories, got %+v", got)
		}
	})
}

func dailyRow(day string, income, expenses int64) dbgen.GetDailyTotalsRow {
	parsed, _ := time.Parse(time.DateOnly, day)
	return dbgen.GetDailyTotalsRow{
		Day:      pgtype.Date{Time: parsed, Valid: true},
		Income:   income,
		Expenses: expenses,
	}
}

func TestFillMissingDays(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 5, 23, 59, 59, 0, time.UTC)

	t.Run("fills gaps at both ends and the middle", func(t *testing.T) {
		got := fillMissingDays([]dbgen.GetDailyTotalsRow{
			dailyRow("2026-06-02", 1000, 0),
			dailyRow("2026-06-04", 0, 250),
		}, start, end)

		if len(got) != 5 {
			t.Fatalf("expected one entry per day in the inclusive range (5), got %d", len(got))
		}
		want := []struct {
			day              string
			income, expenses int64
		}{
			{"2026-06-01", 0, 0},
			{"2026-06-02", 1000, 0},
			{"2026-06-03", 0, 0},
			{"2026-06-04", 0, 250},
			{"2026-06-05", 0, 0},
		}
		for i, w := range want {
			if got[i].Date.Format(time.DateOnly) != w.day {
				t.Errorf("position %d: expected %s, got %s", i, w.day, got[i].Date.Format(time.DateOnly))
			}
			if got[i].Income != w.income || got[i].Expenses != w.expenses {
				t.Errorf("%s: expected income=%d expenses=%d, got %d/%d",
					w.day, w.income, w.expenses, got[i].Income, got[i].Expenses)
			}
		}
	})

	t.Run("a range with no activity is still fully populated", func(t *testing.T) {
		got := fillMissingDays(nil, start, end)
		if len(got) != 5 {
			t.Fatalf("expected 5 zero-filled days, got %d", len(got))
		}
		for _, day := range got {
			if day.Income != 0 || day.Expenses != 0 {
				t.Errorf("expected zeros, got %+v", day)
			}
		}
	})

	t.Run("a single-day range yields one entry", func(t *testing.T) {
		single := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
		got := fillMissingDays(nil, single, single.Add(24*time.Hour-time.Nanosecond))
		if len(got) != 1 {
			t.Fatalf("expected 1 day, got %d", len(got))
		}
	})

	// The end boundary is the last instant of its day, so iterating on raw
	// timestamps rather than day boundaries would drop the final day.
	t.Run("the last day of the range is included", func(t *testing.T) {
		got := fillMissingDays([]dbgen.GetDailyTotalsRow{dailyRow("2026-06-05", 7, 0)}, start, end)
		last := got[len(got)-1]
		if last.Date.Format(time.DateOnly) != "2026-06-05" || last.Income != 7 {
			t.Errorf("expected the end day present with its totals, got %+v", last)
		}
	})

	t.Run("days always serialize as a slice, never nil", func(t *testing.T) {
		if got := fillMissingDays(nil, start, end); got == nil {
			t.Error("expected an empty slice rather than nil")
		}
	})
}
