package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
)

func bulkCreateRows(accountID string, count int) []transactionBody {
	rows := make([]transactionBody, 0, count)
	for i := range count {
		body := validTransactionBody(accountID)
		body.Payee = fmt.Sprintf("Payee %d", i)
		rows = append(rows, body)
	}
	return rows
}

// countUserTransactions counts the caller's transactions through the list
// endpoint, so the read goes through the same RLS-bound path as a real request.
//
// Uses the default range rather than an explicit wide one: the list caps a range
// at 366 days, so "from 2020 to 2026" is a 400, not a bigger window. Every row
// these tests seed is dated inside the default 30 days ending at the pinned
// clock.
func countUserTransactions(t *testing.T, env transactionsTestEnv, token string) int {
	t.Helper()
	rec := env.do(t, http.MethodGet, "/api/transactions", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("count transactions: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var listed openapi.TransactionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return len(listed.Data)
}

// --- bulk create ----------------------------------------------------------

func TestTransactionsBulkLive_CreateHappyPath(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-happy")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")
	categoryID := createTestCategory(t, env.accountsTestEnv, token, "Groceries")

	notes := "weekly shop"
	first := validTransactionBody(accountID)
	first.Payee = "Market"
	first.Notes = &notes
	first.CategoryID = &categoryID
	second := validTransactionBody(accountID)
	second.Payee = "Salary"
	second.Amount = 500000

	rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, []transactionBody{first, second})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var created openapi.TransactionBulkCreateResponse
	raw := rec.Body.String()
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(created.Data) != 2 {
		t.Fatalf("expected 2 created rows, got %d", len(created.Data))
	}

	// Response order must match request order. The handler rebuilds the pairing
	// from the ids it generated (matchCreatedToRequested), so this is a guarantee
	// by construction — not an assumption that RETURNING preserves the source
	// SELECT's order, which PostgreSQL does not promise.
	if created.Data[0].Payee != "Market" || created.Data[1].Payee != "Salary" {
		t.Errorf("expected response order to match request order, got %q then %q",
			created.Data[0].Payee, created.Data[1].Payee)
	}
	if created.Data[1].Amount != 500000 {
		t.Errorf("expected amount 500000 on the second row, got %d", created.Data[1].Amount)
	}
	if created.Data[0].CategoryId == nil || *created.Data[0].CategoryId != categoryID {
		t.Errorf("expected the first row to keep its category, got %v", created.Data[0].CategoryId)
	}
	if created.Data[1].CategoryId != nil {
		t.Errorf("expected the second row to have no category, got %v", *created.Data[1].CategoryId)
	}
	if !strings.Contains(raw, `"categoryId":null`) {
		t.Error("expected a null categoryId to serialize as null, not be omitted")
	}
	for _, row := range created.Data {
		if !legacyResourceIDPattern.MatchString(row.Id) {
			t.Errorf("expected cuid2 ids, got %q", row.Id)
		}
		if row.Currency != openapi.ARS {
			t.Errorf("expected currency ARS, got %q", row.Currency)
		}
	}
}

// TestTransactionsBulkLive_CreateIsAllOrNothing is the ticket's central
// requirement: one bad row must leave the database untouched.
func TestTransactionsBulkLive_CreateIsAllOrNothing(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-atomic")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	if before := countUserTransactions(t, env, token); before != 0 {
		t.Fatalf("expected a clean slate, found %d transactions", before)
	}

	t.Run("invalid row rejects the whole batch", func(t *testing.T) {
		rows := bulkCreateRows(accountID, 3)
		rows[1].Amount = 10.5 // fractional: invalid milliunits
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, rows)
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")

		if after := countUserTransactions(t, env, token); after != 0 {
			t.Errorf("a rejected batch must insert nothing, found %d transactions", after)
		}
	})

	t.Run("unowned reference rejects the whole batch", func(t *testing.T) {
		rows := bulkCreateRows(accountID, 3)
		rows[2].AccountID = "does-not-exist"
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, rows)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "ACCOUNT_NOT_FOUND")

		if after := countUserTransactions(t, env, token); after != 0 {
			t.Errorf("a rejected batch must insert nothing, found %d transactions", after)
		}
	})
}

func TestTransactionsBulkLive_CreateRowCountLimits(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-limits")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	t.Run("empty array is rejected", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, []transactionBody{})
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
	})

	t.Run("exactly the maximum is accepted", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token,
			bulkCreateRows(accountID, maxBulkCreateTransactions))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %d rows, got %d (body: %s)",
				maxBulkCreateTransactions, rec.Code, rec.Body.String())
		}
		var created openapi.TransactionBulkCreateResponse
		if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(created.Data) != maxBulkCreateTransactions {
			t.Errorf("expected %d rows, got %d", maxBulkCreateTransactions, len(created.Data))
		}
	})

	// One past the maximum is a schema violation, so 400 rather than 413: the
	// 413 is about bytes, not row count.
	t.Run("one past the maximum is 400, not 413", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token,
			bulkCreateRows(accountID, maxBulkCreateTransactions+1))
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
	})
}

// TestTransactionsBulkLive_CreateRowErrorsAreIndexed pins the deliberate error
// shape: every failure is reported at once, with the offending row's index in
// the field path, so a client fixing a large CSV import does not have to
// discover its bad rows one request at a time.
func TestTransactionsBulkLive_CreateRowErrorsAreIndexed(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-rowerrors")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	rows := bulkCreateRows(accountID, 4)
	rows[1].Amount = maxSafeMilliunits + 1
	rows[3].Currency = "USD"

	rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, rows)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	body := decodeAccountsAPIError(t, rec)
	if body.Error.Details == nil {
		t.Fatal("expected details.fields")
	}
	details := fmt.Sprint(*body.Error.Details)
	for _, want := range []string{"[1].amount", "[3].currency"} {
		if !strings.Contains(details, want) {
			t.Errorf("expected a field error at %q, got %v", want, details)
		}
	}
	// Both failures in one response, not just the first.
	if strings.Contains(details, "[0].") || strings.Contains(details, "[2].") {
		t.Errorf("valid rows must not produce field errors, got %v", details)
	}
}

func TestTransactionsBulkLive_CreateReferenceOwnership(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "bulk-ref-a")
	_, tokenB := env.createAccountsTestUser(t, "bulk-ref-b")

	accountA := createTestAccount(t, env.accountsTestEnv, tokenA, "A Cash")
	accountB := createTestAccount(t, env.accountsTestEnv, tokenB, "B Cash")
	categoryB := createTestCategory(t, env.accountsTestEnv, tokenB, "B Groceries")

	t.Run("foreign account in the batch is 404 ACCOUNT_NOT_FOUND", func(t *testing.T) {
		rows := bulkCreateRows(accountA, 2)
		rows[1].AccountID = accountB
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", tokenA, rows)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "ACCOUNT_NOT_FOUND")
	})

	t.Run("foreign category in the batch is 404 CATEGORY_NOT_FOUND", func(t *testing.T) {
		rows := bulkCreateRows(accountA, 2)
		rows[1].CategoryID = &categoryB
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", tokenA, rows)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "CATEGORY_NOT_FOUND")
	})

	// Account precedence, same as the single-row path.
	t.Run("a batch bad on both reports the account", func(t *testing.T) {
		rows := bulkCreateRows(accountA, 1)
		rows[0].AccountID = accountB
		rows[0].CategoryID = &categoryB
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", tokenA, rows)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "ACCOUNT_NOT_FOUND")
	})
}

// --- bulk delete ----------------------------------------------------------

func TestTransactionsBulkLive_DeleteOwnedOnly(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "bulk-del-a")
	_, tokenB := env.createAccountsTestUser(t, "bulk-del-b")

	accountA := createTestAccount(t, env.accountsTestEnv, tokenA, "A Cash")
	accountB := createTestAccount(t, env.accountsTestEnv, tokenB, "B Cash")
	txnA := createTestTransaction(t, env, tokenA, validTransactionBody(accountA))
	txnB := createTestTransaction(t, env, tokenB, validTransactionBody(accountB))

	// A's id, B's id, a garbage id, and a duplicate of A's — only A's own row
	// may be deleted, and it must be reported once.
	rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", tokenA, bulkDeleteBody{
		Ids: []string{txnA.Id, txnB.Id, "garbage-" + uuid.NewString(), txnA.Id},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var deleted openapi.DeletedResourceListResponse
	if err := json.NewDecoder(rec.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(deleted.Data) != 1 || deleted.Data[0].Id != txnA.Id {
		t.Errorf("expected only A's id once, got %+v", deleted.Data)
	}

	// B's transaction survives.
	if got := env.do(t, http.MethodGet, "/api/transactions/"+txnB.Id, tokenB, nil); got.Code != http.StatusOK {
		t.Errorf("B's transaction must survive A's bulk-delete, got %d", got.Code)
	}
}

func TestTransactionsBulkLive_DeleteNeverNotFound(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-del-404")

	rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token, bulkDeleteBody{
		Ids: []string{"missing-" + uuid.NewString()},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("bulk-delete must never 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"data":[]}` {
		t.Errorf(`expected {"data":[]}, got %s`, body)
	}
}

func TestTransactionsBulkLive_DeleteIDLimits(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-del-limits")

	ids := make([]string, 0, maxBulkDeleteTransactions+1)
	for i := range maxBulkDeleteTransactions + 1 {
		ids = append(ids, fmt.Sprintf("missing-%d", i))
	}

	t.Run("exactly the maximum is accepted", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token,
			bulkDeleteBody{Ids: ids[:maxBulkDeleteTransactions]})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("one past the maximum is rejected", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token, bulkDeleteBody{Ids: ids})
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
	})

	t.Run("empty and empty-string ids are rejected", func(t *testing.T) {
		assertTransactionAPIError(t,
			env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token, bulkDeleteBody{Ids: []string{}}),
			http.StatusBadRequest, "VALIDATION_ERROR")
		assertTransactionAPIError(t,
			env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token, bulkDeleteBody{Ids: []string{"ok", ""}}),
			http.StatusBadRequest, "VALIDATION_ERROR")
	})
}

// --- body size ------------------------------------------------------------

// TestTransactionsBulkLive_BodyTooLarge covers the 413, including the case
// legacy's guard misses.
//
// Legacy trusts Content-Length, so a request that does not declare one streams
// unbounded into the decoder. Here the limit is enforced against the actual
// stream too (improvement 0016), so an oversized body is rejected whether or not
// it announced its size.
//
// Legacy's other carve-out — a non-numeric Content-Length being ignored — is
// deliberately not tested: net/http rejects a malformed header with its own 400
// before any handler runs, so the branch is unreachable in this server and a
// test could only "pass" by constructing a request that bypasses the parser.
func TestTransactionsBulkLive_BodyTooLarge(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-toolarge")

	oversized := func(limit int) []byte {
		padding := strings.Repeat("x", limit+1024)
		return []byte(fmt.Sprintf(`{"ids":["%s"]}`, padding))
	}

	t.Run("bulk-delete with a declared oversized body", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token, oversized(maxBulkDeleteBodyBytes))
		assertTransactionAPIError(t, rec, http.StatusRequestEntityTooLarge, "REQUEST_BODY_TOO_LARGE")
	})

	t.Run("bulk-create with a declared oversized body", func(t *testing.T) {
		padding := strings.Repeat("x", maxBulkCreateBodyBytes+1024)
		raw := []byte(fmt.Sprintf(`[{"amount":-1,"payee":"%s","date":"2026-07-20","accountId":"a","currency":"ARS"}]`, padding))
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, raw)
		assertTransactionAPIError(t, rec, http.StatusRequestEntityTooLarge, "REQUEST_BODY_TOO_LARGE")
	})

	// A body within the limit still decodes normally, so the guard is not simply
	// rejecting everything.
	t.Run("a body within the limit is processed", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token,
			bulkDeleteBody{Ids: []string{"missing-id"}})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})

	// The case that motivates improvement 0016. With ContentLength -1 (what a
	// chunked request looks like to a handler) legacy's Content-Length guard has
	// nothing to inspect and streams the whole body into the decoder. Here
	// MaxBytesReader still stops it, so the 413 does not depend on the client
	// honestly declaring its size.
	t.Run("undeclared oversized body is still rejected", func(t *testing.T) {
		payload := oversized(maxBulkDeleteBodyBytes)
		req := httptest.NewRequest(http.MethodPost, "/api/transactions/bulk-delete", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.ContentLength = -1

		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		assertTransactionAPIError(t, rec, http.StatusRequestEntityTooLarge, "REQUEST_BODY_TOO_LARGE")
	})

	// The same shape within the limit must still succeed undeclared, so the
	// check above is enforcing the limit rather than rejecting every chunked
	// request.
	t.Run("undeclared body within the limit is processed", func(t *testing.T) {
		payload := []byte(`{"ids":["missing-id"]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/transactions/bulk-delete", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.ContentLength = -1

		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

// --- rate limiting --------------------------------------------------------

// TestTransactionsBulkLive_RateLimitBudgetsAreIndependent proves the two bulk
// actions have their own budgets, separate from each other and from the
// single-mutation actions.
func TestTransactionsBulkLive_RateLimitBudgetsAreIndependent(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-ratelimit")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	for i := range 60 {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, bulkCreateRows(accountID, 1))
		if rec.Code != http.StatusOK {
			t.Fatalf("bulk-create %d of 60: expected 200, got %d (body: %s)", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, bulkCreateRows(accountID, 1))
	assertTransactionAPIError(t, rec, http.StatusTooManyRequests, "TRANSACTION_MUTATION_RATE_LIMITED")
	if retryAfter := rec.Header().Get("Retry-After"); retryAfter != "60" {
		t.Errorf("expected Retry-After 60, got %q", retryAfter)
	}

	// bulk-delete is a different action and untouched.
	if got := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token,
		bulkDeleteBody{Ids: []string{"missing-id"}}); got.Code != http.StatusOK {
		t.Errorf("bulk-delete has an independent budget, got %d", got.Code)
	}
	// So is single create.
	if got := env.do(t, http.MethodPost, "/api/transactions", token,
		validTransactionBody(accountID)); got.Code != http.StatusOK {
		t.Errorf("single create has an independent budget, got %d", got.Code)
	}
}

// TestTransactionsBulkLive_BulkDeleteBudgetIsIndependent exhausts the other
// direction.
//
// Exhausting only bulk-create leaves a misconfigured bulk-delete key
// undetected: if bulk-delete shared another action's budget, or keyed itself
// wrongly, this is the test that catches it. Uses its own user so the budgets
// start clean.
func TestTransactionsBulkLive_BulkDeleteBudgetIsIndependent(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-del-ratelimit")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")
	existing := createTestTransaction(t, env, token, validTransactionBody(accountID))

	for i := range 60 {
		rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token,
			bulkDeleteBody{Ids: []string{"missing-id"}})
		if rec.Code != http.StatusOK {
			t.Fatalf("bulk-delete %d of 60: expected 200, got %d (body: %s)", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token,
		bulkDeleteBody{Ids: []string{"missing-id"}})
	assertTransactionAPIError(t, rec, http.StatusTooManyRequests, "TRANSACTION_MUTATION_RATE_LIMITED")

	// Every other action still has its own budget.
	if got := env.do(t, http.MethodPost, "/api/transactions/bulk-create", token,
		bulkCreateRows(accountID, 1)); got.Code != http.StatusOK {
		t.Errorf("bulk-create must be unaffected by an exhausted bulk-delete budget, got %d", got.Code)
	}
	if got := env.do(t, http.MethodDelete, "/api/transactions/"+existing.Id, token, nil); got.Code != http.StatusOK {
		t.Errorf("single delete must be unaffected by an exhausted bulk-delete budget, got %d", got.Code)
	}
}

// --- unauthorized ---------------------------------------------------------

func TestTransactionsBulkLive_Unauthorized(t *testing.T) {
	env := newTransactionsTestEnv(t)

	for _, path := range []string{"/api/transactions/bulk-create", "/api/transactions/bulk-delete"} {
		t.Run(path, func(t *testing.T) {
			rec := env.do(t, http.MethodPost, path, "", bulkDeleteBody{Ids: []string{"x"}})
			assertTransactionAPIError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
		})
	}
}

// TestTransactionsBulkLive_ValidationErrorPathsMatchTheContract pins the field
// paths each bulk operation actually emits, because the two differ and the
// contract now documents them separately.
//
// bulk-create reports `[i].field` per failing row and `$` for whole-array
// problems; bulk-delete reports `ids` and `ids[i]`; an unparseable body on either
// carries no details at all. A single shared response component previously
// advertised the create shape for both, so bulk-delete's generated docs promised
// paths this code never produces.
func TestTransactionsBulkLive_ValidationErrorPathsMatchTheContract(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "bulk-paths")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	fieldPaths := func(t *testing.T, rec *httptest.ResponseRecorder) []string {
		t.Helper()
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
		}
		var parsed struct {
			Error struct {
				Details *struct {
					Fields []struct{ Path string } `json:"fields"`
				} `json:"details"`
			} `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if parsed.Error.Details == nil {
			return nil
		}
		paths := make([]string, 0, len(parsed.Error.Details.Fields))
		for _, f := range parsed.Error.Details.Fields {
			paths = append(paths, f.Path)
		}
		return paths
	}

	assertPaths := func(t *testing.T, got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("expected paths %v, got %v", want, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("path %d: expected %q, got %q", i, want[i], got[i])
			}
		}
	}

	t.Run("bulk-create row failure uses [i].field", func(t *testing.T) {
		rows := bulkCreateRows(accountID, 2)
		rows[1].Currency = "USD"
		assertPaths(t, fieldPaths(t, env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, rows)),
			[]string{"[1].currency"})
	})

	t.Run("bulk-create whole-array failure uses $", func(t *testing.T) {
		assertPaths(t, fieldPaths(t, env.do(t, http.MethodPost, "/api/transactions/bulk-create", token, []transactionBody{})),
			[]string{"$"})
	})

	t.Run("bulk-delete array failure uses ids", func(t *testing.T) {
		assertPaths(t, fieldPaths(t, env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token, bulkDeleteBody{Ids: []string{}})),
			[]string{"ids"})
	})

	t.Run("bulk-delete empty id uses ids[i]", func(t *testing.T) {
		assertPaths(t, fieldPaths(t, env.do(t, http.MethodPost, "/api/transactions/bulk-delete", token,
			bulkDeleteBody{Ids: []string{"ok", ""}})), []string{"ids[1]"})
	})

	// An unparseable body has nothing to index, so details is omitted on both.
	t.Run("malformed body carries no details", func(t *testing.T) {
		for _, path := range []string{"/api/transactions/bulk-create", "/api/transactions/bulk-delete"} {
			if got := fieldPaths(t, env.do(t, http.MethodPost, path, token, []byte(`{"not":"valid here"`))); got != nil {
				t.Errorf("%s: expected no details for a malformed body, got %v", path, got)
			}
		}
	})
}
