package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/config"
	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	"github.com/GonzaloSecades/nuchi/backend/internal/mail"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
)

// transactionsTestClock is a controllable clock shared by the router's date
// defaults and its rate limiter, so neither depends on when the suite runs and
// no test has to sleep.
type transactionsTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTransactionsTestClock() *transactionsTestClock {
	return &transactionsTestClock{now: time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)}
}

func (c *transactionsTestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *transactionsTestClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type transactionsTestEnv struct {
	accountsTestEnv
	clock *transactionsTestClock
}

func newTransactionsTestEnv(t *testing.T) transactionsTestEnv {
	t.Helper()

	databaseURL := liveDatabaseURL(t, "live transactions HTTP test")

	pool, err := db.NewPool(t.Context(), databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	secret := []byte("transactions-live-test-secret-at-least-32-bytes!!")
	cfg := config.Config{
		JWTSecret:            secret,
		AccessTokenTTL:       30 * time.Minute,
		RefreshTokenTTL:      720 * time.Hour,
		CookieSecure:         false,
		VerificationTokenTTL: 48 * time.Hour,
		ResetTokenTTL:        30 * time.Minute,
	}

	clock := newTransactionsTestClock()
	authServer := NewAuthServer(pool, cfg, mail.NewCapturingMailer())
	resourceServer := newResourceServerWithClock(pool, clock.Now)

	return transactionsTestEnv{
		accountsTestEnv: accountsTestEnv{pool: pool, router: NewRouter(authServer, resourceServer), secret: secret},
		clock:           clock,
	}
}

// transactionBody is the request shape, mirroring TransactionInput. Amount is
// json.Number so tests can send values Go's int64 would refuse to marshal.
type transactionBody struct {
	Amount     any     `json:"amount"`
	Payee      string  `json:"payee"`
	Notes      *string `json:"notes,omitempty"`
	Date       string  `json:"date"`
	AccountID  string  `json:"accountId"`
	CategoryID *string `json:"categoryId,omitempty"`
	Currency   string  `json:"currency"`
}

func validTransactionBody(accountID string) transactionBody {
	return transactionBody{
		Amount:    -12500,
		Payee:     "Market",
		Date:      "2026-07-20",
		AccountID: accountID,
		Currency:  "ARS",
	}
}

func createTestTransaction(t *testing.T, env transactionsTestEnv, token string, body transactionBody) openapi.Transaction {
	t.Helper()
	rec := env.do(t, http.MethodPost, "/api/transactions", token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("create transaction: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var created openapi.TransactionResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create transaction: %v", err)
	}
	return created.Data
}

func assertTransactionAPIError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected %d, got %d (body: %s)", status, rec.Code, rec.Body.String())
	}
	body := decodeAccountsAPIError(t, rec)
	if body.Error.Code != code {
		t.Errorf("expected code %q, got %q", code, body.Error.Code)
	}
}

// --- happy path -----------------------------------------------------------

func TestTransactionsLive_HappyPath_AllOperations(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-happy")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")
	categoryID := createTestCategory(t, env.accountsTestEnv, token, "Groceries")

	notes := "weekly shop"
	body := validTransactionBody(accountID)
	body.Notes = &notes
	body.CategoryID = &categoryID

	created := createTestTransaction(t, env, token, body)
	if !legacyResourceIDPattern.MatchString(created.Id) {
		t.Errorf("expected a cuid2-format id, got %q", created.Id)
	}
	if created.Amount != -12500 {
		t.Errorf("expected amount -12500, got %d", created.Amount)
	}
	if created.Currency != openapi.ARS {
		t.Errorf("expected currency ARS, got %q", created.Currency)
	}
	if created.CategoryId == nil || *created.CategoryId != categoryID {
		t.Errorf("expected categoryId %q, got %v", categoryID, created.CategoryId)
	}

	// Get: entity shape, no joined names.
	getRec := env.do(t, http.MethodGet, "/api/transactions/"+created.Id, token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d (body: %s)", getRec.Code, getRec.Body.String())
	}
	getBody := getRec.Body.String()
	if strings.Contains(getBody, `"account"`) || strings.Contains(getBody, `"category"`) {
		t.Errorf("get must not carry the joined account/category names, got %s", getBody)
	}

	// List: joined names present.
	listRec := env.do(t, http.MethodGet, "/api/transactions", token, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d (body: %s)", listRec.Code, listRec.Body.String())
	}
	var listed openapi.TransactionListResponse
	if err := json.NewDecoder(strings.NewReader(listRec.Body.String())).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Data) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(listed.Data))
	}
	item := listed.Data[0]
	if item.Account != "Cash" {
		t.Errorf("expected joined account name %q, got %q", "Cash", item.Account)
	}
	if item.Category == nil || *item.Category != "Groceries" {
		t.Errorf("expected joined category name %q, got %v", "Groceries", item.Category)
	}
	if item.Currency != openapi.ARS {
		t.Errorf("expected currency ARS on the list item, got %q", item.Currency)
	}

	// Update replaces every field.
	updateBody := validTransactionBody(accountID)
	updateBody.Amount = 25000
	updateBody.Payee = "Employer"
	rec := env.do(t, http.MethodPatch, "/api/transactions/"+created.Id, token, updateBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var updated openapi.TransactionResponse
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Data.Amount != 25000 || updated.Data.Payee != "Employer" {
		t.Errorf("update did not replace fields: %+v", updated.Data)
	}
	// categoryId was omitted on update, which means "no category".
	if updated.Data.CategoryId != nil {
		t.Errorf("expected categoryId cleared, got %v", *updated.Data.CategoryId)
	}

	// Delete.
	delRec := env.do(t, http.MethodDelete, "/api/transactions/"+created.Id, token, nil)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d (body: %s)", delRec.Code, delRec.Body.String())
	}
	assertTransactionAPIError(t, env.do(t, http.MethodGet, "/api/transactions/"+created.Id, token, nil),
		http.StatusNotFound, "TRANSACTION_NOT_FOUND")
}

func TestTransactionsLive_ListEmpty_ReturnsEmptyArrayNotNull(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-empty")

	rec := env.do(t, http.MethodGet, "/api/transactions", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"data":[]}` {
		t.Errorf(`expected {"data":[]}, got %s`, body)
	}
}

// --- amounts --------------------------------------------------------------

// TestTransactionsLive_AmountRange is the regression net for the bigint
// widening (migration 00005) and the checked conversion. Values past the old
// int4 cap must now round-trip exactly, and values past the safe-integer bound
// must be refused rather than silently wrapped.
func TestTransactionsLive_AmountRange(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-amounts")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	t.Run("values beyond the old int32 cap round-trip", func(t *testing.T) {
		for _, amount := range []int64{3_000_000_000, -3_000_000_000, 2147483648, -2147483649} {
			body := validTransactionBody(accountID)
			body.Amount = amount
			created := createTestTransaction(t, env, token, body)
			if created.Amount != amount {
				t.Errorf("amount %d did not round-trip, got %d", amount, created.Amount)
			}
		}
	})

	t.Run("safe-integer bounds are accepted", func(t *testing.T) {
		for _, amount := range []int64{maxSafeMilliunits, -maxSafeMilliunits, 0} {
			body := validTransactionBody(accountID)
			body.Amount = amount
			created := createTestTransaction(t, env, token, body)
			if created.Amount != amount {
				t.Errorf("amount %d did not round-trip, got %d", amount, created.Amount)
			}
		}
	})

	t.Run("beyond the safe-integer bound is rejected", func(t *testing.T) {
		for _, amount := range []int64{maxSafeMilliunits + 1, -(maxSafeMilliunits + 1)} {
			body := validTransactionBody(accountID)
			body.Amount = amount
			rec := env.do(t, http.MethodPost, "/api/transactions", token, body)
			assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
		}
	})

	t.Run("fractional amounts are rejected", func(t *testing.T) {
		body := validTransactionBody(accountID)
		body.Amount = 10.5
		rec := env.do(t, http.MethodPost, "/api/transactions", token, body)
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
	})

	t.Run("missing amount is rejected but explicit zero is valid", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions", token, map[string]any{
			"payee": "X", "date": "2026-07-20", "accountId": accountID, "currency": "ARS",
		})
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")

		body := validTransactionBody(accountID)
		body.Amount = 0
		if created := createTestTransaction(t, env, token, body); created.Amount != 0 {
			t.Errorf("expected zero to round-trip, got %d", created.Amount)
		}
	})
}

// --- currency and category ------------------------------------------------

func TestTransactionsLive_CurrencyIsRequiredAndMustBeARS(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-currency")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	t.Run("omitted currency is rejected", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/transactions", token, map[string]any{
			"amount": -100, "payee": "X", "date": "2026-07-20", "accountId": accountID,
		})
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
	})

	t.Run("unsupported currency is rejected", func(t *testing.T) {
		body := validTransactionBody(accountID)
		body.Currency = "USD"
		rec := env.do(t, http.MethodPost, "/api/transactions", token, body)
		assertTransactionAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
	})
}

// TestTransactionsLive_CategoryIdOmittedAndNullBothClear pins that the two
// syntactic ways of saying "no category" produce the same outcome.
func TestTransactionsLive_CategoryIdOmittedAndNullBothClear(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-category")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	omitted := createTestTransaction(t, env, token, validTransactionBody(accountID))
	if omitted.CategoryId != nil {
		t.Errorf("omitted categoryId should mean no category, got %v", *omitted.CategoryId)
	}

	rec := env.do(t, http.MethodPost, "/api/transactions", token, map[string]any{
		"amount": -100, "payee": "X", "date": "2026-07-20",
		"accountId": accountID, "currency": "ARS", "categoryId": nil,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("explicit null categoryId: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	// Capture the raw body before decoding: the decoder consumes it, and the
	// null-vs-omitted assertion below needs the wire form.
	rawBody := rec.Body.String()
	var explicit openapi.TransactionResponse
	if err := json.NewDecoder(strings.NewReader(rawBody)).Decode(&explicit); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if explicit.Data.CategoryId != nil {
		t.Errorf("explicit null categoryId should mean no category, got %v", *explicit.Data.CategoryId)
	}
	// Serialized as null, not omitted — the contract marks categoryId required
	// and nullable, so dropping the key would violate it.
	if !strings.Contains(rawBody, `"categoryId":null`) {
		t.Errorf("expected categoryId to serialize as null, got %s", rawBody)
	}
}

// --- reference ownership --------------------------------------------------

func TestTransactionsLive_ReferenceOwnership(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "txn-ref-a")
	_, tokenB := env.createAccountsTestUser(t, "txn-ref-b")

	accountA := createTestAccount(t, env.accountsTestEnv, tokenA, "A Cash")
	accountB := createTestAccount(t, env.accountsTestEnv, tokenB, "B Cash")
	categoryB := createTestCategory(t, env.accountsTestEnv, tokenB, "B Groceries")

	t.Run("create with a foreign account is 404 ACCOUNT_NOT_FOUND", func(t *testing.T) {
		body := validTransactionBody(accountB)
		rec := env.do(t, http.MethodPost, "/api/transactions", tokenA, body)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "ACCOUNT_NOT_FOUND")
	})

	// A foreign key would happily accept user B's category here: it proves the
	// row exists, not that user A owns it, and FK checks bypass RLS.
	t.Run("create with a foreign category is 404 CATEGORY_NOT_FOUND", func(t *testing.T) {
		body := validTransactionBody(accountA)
		body.CategoryID = &categoryB
		rec := env.do(t, http.MethodPost, "/api/transactions", tokenA, body)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "CATEGORY_NOT_FOUND")
	})

	t.Run("missing account beats missing transaction", func(t *testing.T) {
		body := validTransactionBody("does-not-exist")
		rec := env.do(t, http.MethodPatch, "/api/transactions/also-missing", tokenA, body)
		// Legacy validates references before matching the transaction, so the
		// account error wins even though the transaction is missing too.
		assertTransactionAPIError(t, rec, http.StatusNotFound, "ACCOUNT_NOT_FOUND")
	})

	t.Run("update to a foreign account is 404", func(t *testing.T) {
		existing := createTestTransaction(t, env, tokenA, validTransactionBody(accountA))
		body := validTransactionBody(accountB)
		rec := env.do(t, http.MethodPatch, "/api/transactions/"+existing.Id, tokenA, body)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "ACCOUNT_NOT_FOUND")
	})
}

func TestTransactionsLive_CrossUserIsolation(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "txn-iso-a")
	_, tokenB := env.createAccountsTestUser(t, "txn-iso-b")

	accountA := createTestAccount(t, env.accountsTestEnv, tokenA, "A Cash")
	accountB := createTestAccount(t, env.accountsTestEnv, tokenB, "B Cash")
	txnA := createTestTransaction(t, env, tokenA, validTransactionBody(accountA))
	txnB := createTestTransaction(t, env, tokenB, validTransactionBody(accountB))

	// A cannot see or touch B's transaction.
	assertTransactionAPIError(t, env.do(t, http.MethodGet, "/api/transactions/"+txnB.Id, tokenA, nil),
		http.StatusNotFound, "TRANSACTION_NOT_FOUND")
	assertTransactionAPIError(t, env.do(t, http.MethodDelete, "/api/transactions/"+txnB.Id, tokenA, nil),
		http.StatusNotFound, "TRANSACTION_NOT_FOUND")

	// B's row survives A's attempts.
	if rec := env.do(t, http.MethodGet, "/api/transactions/"+txnB.Id, tokenB, nil); rec.Code != http.StatusOK {
		t.Errorf("B's transaction should be untouched, got %d", rec.Code)
	}

	// A's list contains only A's transaction.
	rec := env.do(t, http.MethodGet, "/api/transactions", tokenA, nil)
	var listed openapi.TransactionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Id != txnA.Id {
		t.Errorf("A's list should contain only A's transaction, got %+v", listed.Data)
	}
}

// --- list filtering -------------------------------------------------------

func TestTransactionsLive_ListFiltering(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-filter")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")
	otherAccountID := createTestAccount(t, env.accountsTestEnv, token, "Savings")

	// Inserted deliberately out of order so the DESC assertion means something.
	for _, date := range []string{"2026-07-10", "2026-07-28", "2026-07-19"} {
		body := validTransactionBody(accountID)
		body.Date = date
		createTestTransaction(t, env, token, body)
	}
	otherBody := validTransactionBody(otherAccountID)
	otherBody.Date = "2026-07-15"
	createTestTransaction(t, env, token, otherBody)

	list := func(t *testing.T, query string) []openapi.TransactionListItem {
		t.Helper()
		rec := env.do(t, http.MethodGet, "/api/transactions"+query, token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list%s: expected 200, got %d (body: %s)", query, rec.Code, rec.Body.String())
		}
		var out openapi.TransactionListResponse
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		return out.Data
	}

	t.Run("sorted by date descending", func(t *testing.T) {
		items := list(t, "")
		for i := 1; i < len(items); i++ {
			if items[i-1].Date.Before(items[i].Date) {
				t.Fatalf("expected date DESC, got %v before %v", items[i-1].Date, items[i].Date)
			}
		}
	})

	t.Run("from and to are inclusive at both ends", func(t *testing.T) {
		// 07-10 and 07-19 are the boundaries and must be included; 07-15 falls
		// inside; 07-28 must be excluded.
		items := list(t, "?from=2026-07-10&to=2026-07-19")
		dates := make(map[string]bool, len(items))
		for _, item := range items {
			dates[item.Date.Format(time.DateOnly)] = true
		}
		for _, want := range []string{"2026-07-10", "2026-07-19", "2026-07-15"} {
			if !dates[want] {
				t.Errorf("expected %s to be inside the inclusive range, got %+v", want, dates)
			}
		}
		if dates["2026-07-28"] {
			t.Error("2026-07-28 is outside the range and must be excluded")
		}
	})

	t.Run("accountId filters", func(t *testing.T) {
		items := list(t, "?accountId="+otherAccountID)
		if len(items) != 1 || items[0].AccountId != otherAccountID {
			t.Errorf("expected only the other account's row, got %+v", items)
		}
	})

	// The join still requires an owned account, so an unowned filter yields an
	// empty list rather than a 404.
	t.Run("unowned accountId yields an empty list, not 404", func(t *testing.T) {
		items := list(t, "?accountId=does-not-exist")
		if len(items) != 0 {
			t.Errorf("expected an empty list, got %+v", items)
		}
	})

	t.Run("invalid dates return INVALID_QUERY", func(t *testing.T) {
		for _, query := range []string{"?from=2026/07/01", "?from=2026-07-30&to=2026-07-01", "?from=2024-01-01&to=2026-07-01"} {
			rec := env.do(t, http.MethodGet, "/api/transactions"+query, token, nil)
			assertTransactionAPIError(t, rec, http.StatusBadRequest, "INVALID_QUERY")
		}
	})
}

// --- rate limiting --------------------------------------------------------

func TestTransactionsLive_MutationRateLimit(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-ratelimit")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	// Exhaust the create budget.
	for i := range 60 {
		rec := env.do(t, http.MethodPost, "/api/transactions", token, validTransactionBody(accountID))
		if rec.Code != http.StatusOK {
			t.Fatalf("create %d of 60: expected 200, got %d (body: %s)", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := env.do(t, http.MethodPost, "/api/transactions", token, validTransactionBody(accountID))
	assertTransactionAPIError(t, rec, http.StatusTooManyRequests, "TRANSACTION_MUTATION_RATE_LIMITED")
	if retryAfter := rec.Header().Get("Retry-After"); retryAfter != "60" {
		t.Errorf("expected Retry-After 60, got %q", retryAfter)
	}

	// Delete has its own budget and is unaffected.
	existing := env.do(t, http.MethodGet, "/api/transactions", token, nil)
	var listed openapi.TransactionListResponse
	if err := json.NewDecoder(existing.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if delRec := env.do(t, http.MethodDelete, "/api/transactions/"+listed.Data[0].Id, token, nil); delRec.Code != http.StatusOK {
		t.Errorf("delete has an independent budget and should succeed, got %d", delRec.Code)
	}

	// Once the window passes, creates are allowed again.
	env.clock.Advance(61 * time.Second)
	if rec := env.do(t, http.MethodPost, "/api/transactions", token, validTransactionBody(accountID)); rec.Code != http.StatusOK {
		t.Errorf("expected the create budget to recover after the window, got %d", rec.Code)
	}
}

// --- unauthorized ---------------------------------------------------------

func TestTransactionsLive_Unauthorized(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, validToken := env.createAccountsTestUser(t, "txn-unauth")
	accountID := createTestAccount(t, env.accountsTestEnv, validToken, "Cash")
	txn := createTestTransaction(t, env, validToken, validTransactionBody(accountID))

	wrongSecretToken, _, err := auth.IssueAccessToken([]byte("a-totally-different-secret-32-bytes!!"), uuid.New(), 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken wrong secret: %v", err)
	}
	expiredToken, _, err := auth.IssueAccessToken(env.secret, uuid.New(), -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken expired: %v", err)
	}

	routes := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/api/transactions", nil},
		{http.MethodPost, "/api/transactions", validTransactionBody(accountID)},
		{http.MethodGet, "/api/transactions/" + txn.Id, nil},
		{http.MethodPatch, "/api/transactions/" + txn.Id, validTransactionBody(accountID)},
		{http.MethodDelete, "/api/transactions/" + txn.Id, nil},
	}
	tokenCases := []struct {
		name     string
		token    string
		wantCode string
	}{
		{"no-header", "", "UNAUTHORIZED"},
		{"malformed-header", "not-a-jwt", "UNAUTHORIZED"},
		{"bad-signature", wrongSecretToken, "UNAUTHORIZED"},
		{"expired", expiredToken, "ACCESS_TOKEN_EXPIRED"},
	}

	for _, rt := range routes {
		for _, tc := range tokenCases {
			t.Run(fmt.Sprintf("%s %s/%s", rt.method, rt.path, tc.name), func(t *testing.T) {
				rec := env.do(t, rt.method, rt.path, tc.token, rt.body)
				assertTransactionAPIError(t, rec, http.StatusUnauthorized, tc.wantCode)
			})
		}
	}
}

// TestTransactionsLive_EmptyCategoryIdIsARejectedReference pins that an empty
// categoryId is treated as a reference that does not exist, not as "clear the
// category". Legacy gates on `values.categoryId == null`, which catches null
// and undefined but not "", so an empty id falls through to its category
// lookup and 404s. Silently clearing it instead would turn an invalid client
// value into a successful write.
func TestTransactionsLive_EmptyCategoryIdIsARejectedReference(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-empty-cat")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	empty := ""

	t.Run("create", func(t *testing.T) {
		body := validTransactionBody(accountID)
		body.CategoryID = &empty
		rec := env.do(t, http.MethodPost, "/api/transactions", token, body)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "CATEGORY_NOT_FOUND")
	})

	t.Run("update", func(t *testing.T) {
		existing := createTestTransaction(t, env, token, validTransactionBody(accountID))
		body := validTransactionBody(accountID)
		body.CategoryID = &empty
		rec := env.do(t, http.MethodPatch, "/api/transactions/"+existing.Id, token, body)
		assertTransactionAPIError(t, rec, http.StatusNotFound, "CATEGORY_NOT_FOUND")
	})
}

// TestTransactionsLive_EmptyDateParamsDefaultRatherThanFail pins that `?from=`
// and `?to=` are treated as absent, not malformed.
//
// url.Values.Get returns "" for both an omitted key and an explicitly empty
// one, and legacy's parseStrictDate short-circuits on `if (!value)`, so an
// empty value takes the default range there too. Rejecting it would diverge
// from legacy and would break the existing client, whose transactions hook
// sends from/to as empty strings whenever no filter is set.
func TestTransactionsLive_EmptyDateParamsDefaultRatherThanFail(t *testing.T) {
	env := newTransactionsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "txn-empty-dates")
	accountID := createTestAccount(t, env.accountsTestEnv, token, "Cash")

	// Dated inside the default 30-day window that ends at the pinned clock.
	body := validTransactionBody(accountID)
	body.Date = "2026-07-20"
	created := createTestTransaction(t, env, token, body)

	for _, query := range []string{"?from=&to=", "?from=", "?to=", "?from=&to=&accountId="} {
		t.Run(query, func(t *testing.T) {
			rec := env.do(t, http.MethodGet, "/api/transactions"+query, token, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 for empty date params, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			var listed openapi.TransactionListResponse
			if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
				t.Fatalf("decode list: %v", err)
			}
			if len(listed.Data) != 1 || listed.Data[0].Id != created.Id {
				t.Errorf("expected the default range to include the seeded transaction, got %+v", listed.Data)
			}
		})
	}
}
