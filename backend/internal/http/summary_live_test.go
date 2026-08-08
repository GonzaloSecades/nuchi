package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
)

// getSummary issues a summary request and decodes it, failing on anything but
// 200.
func getSummary(t *testing.T, env transactionsTestEnv, token, query string) openapi.Summary {
	t.Helper()
	rec := env.do(t, http.MethodGet, "/api/summary"+query, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary%s: expected 200, got %d (body: %s)", query, rec.Code, rec.Body.String())
	}
	var parsed openapi.SummaryResponse
	if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	return parsed.Data
}

// seedSummaryTransaction creates a transaction on the given date through the
// API, so it travels the same validated path a real client would.
func seedSummaryTransaction(t *testing.T, env transactionsTestEnv, token, accountID, date string, amount int64, categoryID *string) {
	t.Helper()
	body := validTransactionBody(accountID)
	body.Date = date
	body.Amount = amount
	body.CategoryID = categoryID
	createTestTransaction(t, env, token, body)
}

func TestSummaryLive_Totals(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-totals")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	// Inside the default 30-day window ending at the pinned clock (2026-07-29).
	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", 500000, nil) // income
	seedSummaryTransaction(t, env, token, accountID, "2026-07-21", -12500, nil) // expense
	seedSummaryTransaction(t, env, token, accountID, "2026-07-22", -7500, nil)  // expense

	got := getSummary(t, env, token, "")

	if got.IncomeAmount != 500000 {
		t.Errorf("expected income 500000, got %d", got.IncomeAmount)
	}
	// expenses is the sum of absolute values of negative amounts.
	if got.ExpensesAmount != 20000 {
		t.Errorf("expected expenses 20000, got %d", got.ExpensesAmount)
	}
	// remaining is the signed sum, so income minus expenses.
	if got.RemainingAmount != 480000 {
		t.Errorf("expected remaining 480000, got %d", got.RemainingAmount)
	}
}

// TestSummaryLive_EmptyStateIsZerosAndFilledDays covers what the dashboard
// renders for a brand-new user: zeros rather than nulls, an empty category
// list, and a fully populated day series so the chart still draws a flat line.
func TestSummaryLive_EmptyStateIsZerosAndFilledDays(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-empty")

	rec := env.do(t, http.MethodGet, "/api/summary", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()

	var parsed openapi.SummaryResponse
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := parsed.Data

	for name, value := range map[string]int64{
		"remainingAmount": got.RemainingAmount,
		"incomeAmount":    got.IncomeAmount,
		"expensesAmount":  got.ExpensesAmount,
	} {
		if value != 0 {
			t.Errorf("%s: expected 0 for an empty period, got %d", name, value)
		}
	}
	for name, value := range map[string]float32{
		"remainingChange": got.RemainingChange,
		"incomeChange":    got.IncomeChange,
		"expensesChange":  got.ExpensesChange,
	} {
		if value != 0 {
			t.Errorf("%s: expected 0 when both periods are empty, got %v", name, value)
		}
	}

	// Arrays must be [] on the wire, never null — the dashboard maps over both.
	if !strings.Contains(raw, `"categories":[]`) {
		t.Errorf("expected categories to serialize as [], got %s", raw)
	}
	// 31 days: the default 30-day window is inclusive at both ends.
	if len(got.Days) != 31 {
		t.Errorf("expected the default range fully zero-filled (31 days), got %d", len(got.Days))
	}
	for _, day := range got.Days {
		if day.Income != 0 || day.Expenses != 0 {
			t.Errorf("expected a zero day, got %+v", day)
		}
	}
}

// TestSummaryLive_ChangeComparesThePrecedingWindow pins that the comparison
// period is the same length immediately before the selected one, rather than
// the previous calendar month.
func TestSummaryLive_ChangeComparesThePrecedingWindow(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-change")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	// Selected window is 2026-07-11..2026-07-20 (10 days), so the comparison
	// window is 2026-07-01..2026-07-10.
	seedSummaryTransaction(t, env, token, accountID, "2026-07-15", 200000, nil) // current
	seedSummaryTransaction(t, env, token, accountID, "2026-07-05", 100000, nil) // previous

	got := getSummary(t, env, token, "?from=2026-07-11&to=2026-07-20")

	if got.IncomeAmount != 200000 {
		t.Errorf("expected only the current window's income, got %d", got.IncomeAmount)
	}
	// 200000 vs 100000 is +100%.
	if got.IncomeChange != 100 {
		t.Errorf("expected incomeChange 100, got %v", got.IncomeChange)
	}
}

func TestSummaryLive_ChangeIs100WhenPreviousPeriodIsEmpty(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-change-empty")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	seedSummaryTransaction(t, env, token, accountID, "2026-07-15", 200000, nil)

	got := getSummary(t, env, token, "?from=2026-07-11&to=2026-07-20")
	if got.IncomeChange != 100 {
		t.Errorf("expected 100 when the previous period had nothing, got %v", got.IncomeChange)
	}
	if got.ExpensesChange != 0 {
		t.Errorf("expected 0 when both periods had no expenses, got %v", got.ExpensesChange)
	}
}

// TestSummaryLive_CategoryBreakdown covers the bucketing against real data,
// including the two exclusions that make the chart disagree with the expenses
// total.
func TestSummaryLive_CategoryBreakdown(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-categories")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	ids := make([]string, 0, 4)
	for _, name := range []string{"Rent", "Food", "Transport", "Books"} {
		id := createTestCategory(t, env.accountsTestEnv, token, name)
		ids = append(ids, id)
	}

	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", -90000, &ids[0])
	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", -50000, &ids[1])
	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", -10000, &ids[2])
	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", -4000, &ids[3])
	// Excluded from the breakdown but still counted in expensesAmount.
	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", -1000, nil)
	// Income is never in the breakdown at all.
	seedSummaryTransaction(t, env, token, accountID, "2026-07-20", 300000, &ids[0])

	got := getSummary(t, env, token, "")

	if len(got.Categories) != 4 {
		t.Fatalf("expected 3 categories plus Other, got %d: %+v", len(got.Categories), got.Categories)
	}
	if got.Categories[0].Name != "Rent" || got.Categories[0].Value != 90000 {
		t.Errorf("expected Rent=90000 first (ordered by value), got %+v", got.Categories[0])
	}
	last := got.Categories[len(got.Categories)-1]
	if last.Name != "Other" || last.Value != 4000 {
		t.Errorf("expected Other=4000, got %+v", last)
	}

	// The uncategorized 1000 is in the total but not the chart, and the income
	// on a category does not appear either. This is legacy behavior, recorded
	// as improvement 0017.
	if got.ExpensesAmount != 155000 {
		t.Errorf("expected expenses to include the uncategorized 1000 (155000), got %d", got.ExpensesAmount)
	}
	var charted int64
	for _, c := range got.Categories {
		charted += c.Value
	}
	if charted != 154000 {
		t.Errorf("expected the chart to sum to 154000, excluding uncategorized, got %d", charted)
	}
}

func TestSummaryLive_DailySeries(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-days")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	seedSummaryTransaction(t, env, token, accountID, "2026-07-11", 100000, nil)
	seedSummaryTransaction(t, env, token, accountID, "2026-07-13", -25000, nil)

	got := getSummary(t, env, token, "?from=2026-07-11&to=2026-07-15")

	if len(got.Days) != 5 {
		t.Fatalf("expected 5 inclusive days, got %d", len(got.Days))
	}
	byDay := make(map[string]openapi.SummaryDay, len(got.Days))
	for _, day := range got.Days {
		byDay[day.Date.Format(time.DateOnly)] = day
	}
	if d := byDay["2026-07-11"]; d.Income != 100000 || d.Expenses != 0 {
		t.Errorf("2026-07-11: expected income 100000, got %+v", d)
	}
	if d := byDay["2026-07-13"]; d.Expenses != 25000 || d.Income != 0 {
		t.Errorf("2026-07-13: expected expenses 25000, got %+v", d)
	}
	for _, gap := range []string{"2026-07-12", "2026-07-14", "2026-07-15"} {
		if d, ok := byDay[gap]; !ok {
			t.Errorf("%s missing from the series", gap)
		} else if d.Income != 0 || d.Expenses != 0 {
			t.Errorf("%s: expected a zero-filled day, got %+v", gap, d)
		}
	}
}

func TestSummaryLive_AccountFilter(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-accfilter")
	cash := createTestAccount(t, env.accountsTestEnv, token, "Cash")
	savings := createTestAccount(t, env.accountsTestEnv, token, "Savings")

	seedSummaryTransaction(t, env, token, cash, "2026-07-20", 100000, nil)
	seedSummaryTransaction(t, env, token, savings, "2026-07-20", 400000, nil)

	t.Run("narrows to the named account", func(t *testing.T) {
		got := getSummary(t, env, token, "?accountId="+cash)
		if got.IncomeAmount != 100000 {
			t.Errorf("expected only Cash income 100000, got %d", got.IncomeAmount)
		}
	})

	// An account the caller does not own yields zeros rather than a 404: the
	// join simply matches nothing, and confirming the account exists would leak
	// another user's data.
	t.Run("an unowned account yields zeros, not 404", func(t *testing.T) {
		got := getSummary(t, env, token, "?accountId=does-not-exist")
		if got.IncomeAmount != 0 || got.ExpensesAmount != 0 || got.RemainingAmount != 0 {
			t.Errorf("expected zeros for an unowned account, got %+v", got)
		}
	})
}

func TestSummaryLive_CrossUserIsolation(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "summary-iso-a")
	_, tokenB := env.createAccountsTestUser(t, "summary-iso-b")

	accountA := createTestAccount(t, env.accountsTestEnv, tokenA, "A Cash")
	accountB := createTestAccount(t, env.accountsTestEnv, tokenB, "B Cash")
	categoryB := createTestCategory(t, env.accountsTestEnv, tokenB, "B Rent")

	seedSummaryTransaction(t, env, tokenA, accountA, "2026-07-20", 100000, nil)
	seedSummaryTransaction(t, env, tokenB, accountB, "2026-07-20", 999999, nil)
	seedSummaryTransaction(t, env, tokenB, accountB, "2026-07-20", -55555, &categoryB)

	got := getSummary(t, env, tokenA, "")

	if got.IncomeAmount != 100000 {
		t.Errorf("A's income must exclude B's rows, got %d", got.IncomeAmount)
	}
	if got.ExpensesAmount != 0 {
		t.Errorf("A has no expenses; B's must not leak, got %d", got.ExpensesAmount)
	}
	if len(got.Categories) != 0 {
		t.Errorf("B's category must not appear in A's breakdown, got %+v", got.Categories)
	}

	// Filtering by another user's account must not expose its totals either.
	filtered := getSummary(t, env, tokenA, "?accountId="+accountB)
	if filtered.IncomeAmount != 0 || filtered.ExpensesAmount != 0 {
		t.Errorf("filtering by B's account must yield zeros for A, got %+v", filtered)
	}
}

func TestSummaryLive_DateQueryErrors(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "summary-dates")

	cases := []struct {
		query   string
		message string
	}{
		{"?from=2026/07/01", "from and to must use yyyy-MM-dd dates."},
		{"?from=", "from and to must use yyyy-MM-dd dates."},
		{"?from=2026-07-30&to=2026-07-01", "from must be less than or equal to to."},
		{"?from=2024-01-01&to=2026-07-01", "Date range cannot exceed 366 days."},
		{"?accountId=", "accountId must not be empty."},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			rec := env.do(t, http.MethodGet, "/api/summary"+tc.query, token, nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			// Decoded once: the recorder's body is a reader, so a second decode
			// would see EOF and assert nothing.
			body := decodeAccountsAPIError(t, rec)
			if body.Error.Code != "INVALID_QUERY" {
				t.Errorf("expected code INVALID_QUERY, got %q", body.Error.Code)
			}
			if body.Error.Message != tc.message {
				t.Errorf("expected %q, got %q", tc.message, body.Error.Message)
			}
		})
	}
}

func TestSummaryLive_Unauthorized(t *testing.T) {
	env := newTransactionsTestEnv(t)

	wrongSecretToken, _, err := auth.IssueAccessToken([]byte("a-totally-different-secret-32-bytes!!"), uuid.New(), 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken wrong secret: %v", err)
	}
	expiredToken, _, err := auth.IssueAccessToken(env.secret, uuid.New(), -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken expired: %v", err)
	}

	for _, tc := range []struct {
		name     string
		token    string
		wantCode string
	}{
		{"no-header", "", "UNAUTHORIZED"},
		{"malformed-header", "not-a-jwt", "UNAUTHORIZED"},
		{"bad-signature", wrongSecretToken, "UNAUTHORIZED"},
		{"expired", expiredToken, "ACCESS_TOKEN_EXPIRED"},
	} {
		t.Run(fmt.Sprintf("GET /api/summary/%s", tc.name), func(t *testing.T) {
			rec := env.do(t, http.MethodGet, "/api/summary", tc.token, nil)
			assertTransactionAPIError(t, rec, http.StatusUnauthorized, tc.wantCode)
		})
	}
}
