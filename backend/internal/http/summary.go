package httpapi

import (
	"net/http"
	"time"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// maxSummaryCategories is how many expense categories are returned
// individually. Anything past this is summed into a single "Other" entry, so
// the chart stays readable regardless of how many categories a user has.
const maxSummaryCategories = 3

// otherCategoryName is the label for that bucket. It is a literal rather than
// a translated string because legacy emits exactly this and the frontend
// matches on it.
const otherCategoryName = "Other"

// GetSummary implements GET /api/summary.
//
// Four queries run inside one WithUserTx: current-period totals, the previous
// period's totals for the change percentages, the category breakdown, and the
// per-day series. The transaction uses repeatable-read isolation, giving all
// four one RLS binding and one snapshot — legacy fires them independently, so
// a write landing mid-request can make its totals disagree with its own daily
// series.
//
// The aggregation itself is deliberately split: SQL sums, Go shapes. Bucketing
// and zero-filling stay here because they are presentation decisions, and
// because that is where legacy does them (lib/utils.ts), so the arithmetic can
// be compared line for line.
func (s *ResourceServer) GetSummary(w http.ResponseWriter, r *http.Request, params openapi.GetSummaryParams) {
	query := r.URL.Query()
	start, end, dateErr := parseDateRange(
		optionalQueryParam(query, "from"),
		optionalQueryParam(query, "to"),
		s.now(),
	)
	if dateErr != nil {
		resp := openapi.GetSummary400JSONResponse{InvalidQueryErrorJSONResponse: invalidDateQueryError(dateErr)}
		_ = resp.VisitGetSummaryResponse(w)
		return
	}

	accountID, accountErr := accountIDFilter(params.AccountId)
	if accountErr != nil {
		resp := openapi.GetSummary400JSONResponse{InvalidQueryErrorJSONResponse: invalidQueryError(accountErr.Error())}
		_ = resp.VisitGetSummaryResponse(w)
		return
	}

	// The comparison window is the same length as the selected one, immediately
	// before it — not the previous calendar month. Legacy computes it as
	// differenceInCalendarDays(end, start) + 1 and subtracts that from both
	// ends, so a 30-day view compares against the 30 days before it.
	periodLength := calendarDaysBetween(start, end) + 1
	previousStart := start.AddDate(0, 0, -periodLength)
	previousEnd := end.AddDate(0, 0, -periodLength)

	var current, previous dbgen.GetPeriodTotalsRow
	var categoryRows []dbgen.GetCategorySpendingRow
	var dailyRows []dbgen.GetDailyTotalsRow

	err, ok := s.withUserTxOptions(w, r, pgx.TxOptions{IsoLevel: pgx.RepeatableRead}, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		current, err = q.GetPeriodTotals(r.Context(), dbgen.GetPeriodTotalsParams{
			UserID:    pgUserID(userID),
			StartDate: pgTimestamp(start),
			EndDate:   pgTimestamp(end),
			AccountID: accountID,
		})
		if err != nil {
			return err
		}

		previous, err = q.GetPeriodTotals(r.Context(), dbgen.GetPeriodTotalsParams{
			UserID:    pgUserID(userID),
			StartDate: pgTimestamp(previousStart),
			EndDate:   pgTimestamp(previousEnd),
			AccountID: accountID,
		})
		if err != nil {
			return err
		}

		categoryRows, err = q.GetCategorySpending(r.Context(), dbgen.GetCategorySpendingParams{
			UserID:    pgUserID(userID),
			StartDate: pgTimestamp(start),
			EndDate:   pgTimestamp(end),
			AccountID: accountID,
		})
		if err != nil {
			return err
		}

		dailyRows, err = q.GetDailyTotals(r.Context(), dbgen.GetDailyTotalsParams{
			UserID:    pgUserID(userID),
			StartDate: pgTimestamp(start),
			EndDate:   pgTimestamp(end),
			AccountID: accountID,
		})
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logResourceError(r, "fetch summary", err)
		resp := openapi.GetSummary500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch summary")}
		_ = resp.VisitGetSummaryResponse(w)
		return
	}

	resp := openapi.GetSummary200JSONResponse{Data: openapi.Summary{
		RemainingAmount: current.Remaining,
		RemainingChange: percentageChange(current.Remaining, previous.Remaining),
		IncomeAmount:    current.Income,
		IncomeChange:    percentageChange(current.Income, previous.Income),
		ExpensesAmount:  current.Expenses,
		ExpensesChange:  percentageChange(current.Expenses, previous.Expenses),
		Categories:      bucketCategories(categoryRows),
		Days:            fillMissingDays(dailyRows, start, end),
	}}
	_ = resp.VisitGetSummaryResponse(w)
}

// percentageChange ports lib/utils.ts calculatePercentageChange exactly.
//
// The zero rule is the part worth stating: when the previous period is 0 the
// ratio is undefined, and legacy answers 0 if the current period is also 0 and
// 100 otherwise. That is a product decision ("everything is new growth"),
// not a divide-by-zero guard bolted on afterwards, so both branches are
// deliberate.
//
// This is the one place in the money path that returns a float. It is a ratio,
// not an amount — amounts stay integer milliunits everywhere.
func percentageChange(current, previous int64) float32 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	// Widen before subtracting: current-previous can overflow int64 even though
	// each operand is valid on its own.
	return float32((float64(current) - float64(previous)) / float64(previous) * 100)
}

// bucketCategories returns the largest maxSummaryCategories entries as-is and
// folds everything past them into a single "Other" row.
//
// The query already orders by value descending, so this is a slice rather than
// a sort. With exactly maxSummaryCategories entries there is no remainder and
// no "Other" row is added — the boundary legacy also has.
//
// Note what the query excludes and this therefore inherits: only negative
// amounts, and only those with a category. Uncategorized spending is absent
// from this breakdown while still counted in expensesAmount, so the chart does
// not sum to the expenses total. That is legacy behavior, ported deliberately
// and recorded as post-migration improvement 0017.
func bucketCategories(rows []dbgen.GetCategorySpendingRow) []openapi.SummaryCategory {
	categories := make([]openapi.SummaryCategory, 0, min(len(rows), maxSummaryCategories)+1)

	for i, row := range rows {
		if i >= maxSummaryCategories {
			break
		}
		categories = append(categories, openapi.SummaryCategory{Name: row.Name, Value: row.Value})
	}

	if len(rows) <= maxSummaryCategories {
		return categories
	}

	var other int64
	for _, row := range rows[maxSummaryCategories:] {
		other += row.Value
	}
	return append(categories, openapi.SummaryCategory{Name: otherCategoryName, Value: other})
}

// fillMissingDays returns one entry per calendar day in [start, end], using the
// queried totals where a day has transactions and zeros where it does not.
//
// Ports lib/utils.ts fillMissingDays. The chart needs a continuous series: a
// gap would otherwise be drawn as a line straight from one active day to the
// next, implying activity that did not happen.
//
// Keyed by yyyy-MM-dd in UTC, matching the query's GROUP BY t.date::date. That
// cast is load-bearing rather than cosmetic — grouping by the raw timestamp
// would split one calendar day into several rows if any write path ever stored
// a time of day, and this map would then keep only the last of them.
func fillMissingDays(rows []dbgen.GetDailyTotalsRow, start, end time.Time) []openapi.SummaryDay {
	totals := make(map[string]dbgen.GetDailyTotalsRow, len(rows))
	for _, row := range rows {
		totals[row.Day.Time.UTC().Format(time.DateOnly)] = row
	}

	// Day boundaries, not the raw range: `end` is the last instant of its day,
	// so iterating from midnight to midnight covers every day inclusively.
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)

	days := make([]openapi.SummaryDay, 0, calendarDaysBetween(start, end)+1)
	for !day.After(last) {
		entry := openapi.SummaryDay{Date: day}
		if row, found := totals[day.Format(time.DateOnly)]; found {
			entry.Income = row.Income
			entry.Expenses = row.Expenses
		}
		days = append(days, entry)
		day = day.AddDate(0, 0, 1)
	}
	return days
}

// pgTimestamp converts a UTC instant into the pgtype.Timestamp the summary
// query parameters expect.
func pgTimestamp(at time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: at, Valid: true}
}
