package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Categories reuse accountsTestEnv: it already wires a live pool, the real
// AuthServer + ResourceServer, and the same TEST_DATABASE_URL skip
// convention. Only the routes exercised differ.

func assertCategoryNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	body := decodeAccountsAPIError(t, rec)
	if body.Error.Code != "CATEGORY_NOT_FOUND" {
		t.Errorf("expected code CATEGORY_NOT_FOUND, got %q", body.Error.Code)
	}
}

func assertDuplicateCategoryName(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	body := decodeAccountsAPIError(t, rec)
	if body.Error.Code != "DUPLICATE_CATEGORY_NAME" {
		t.Errorf("expected code DUPLICATE_CATEGORY_NAME, got %q", body.Error.Code)
	}
	if body.Error.Details == nil {
		t.Fatalf("expected details.constraint, got no details (body: %s)", rec.Body.String())
	}
	constraint, ok := (*body.Error.Details)["constraint"]
	if !ok || constraint != "categories_user_id_name_uniq" {
		t.Errorf("expected details.constraint %q, got %v", "categories_user_id_name_uniq", constraint)
	}
}

func createTestCategory(t *testing.T, env accountsTestEnv, token, name string) string {
	t.Helper()
	rec := env.do(t, http.MethodPost, "/api/categories", token, categoryInputBody{Name: name})
	if rec.Code != http.StatusOK {
		t.Fatalf("create category %q: expected 200, got %d (body: %s)", name, rec.Code, rec.Body.String())
	}
	var created openapi.CategoryResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create category: %v", err)
	}
	return created.Data.Id
}

// --- happy path -----------------------------------------------------------

func TestCategoriesLive_HappyPath_AllOperations(t *testing.T) {
	env := newAccountsTestEnv(t)
	userID, token := env.createAccountsTestUser(t, "cat-happy")

	// Create (full row shape).
	createRec := env.do(t, http.MethodPost, "/api/categories", token, categoryInputBody{Name: "Groceries"})
	if createRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d (body: %s)", createRec.Code, createRec.Body.String())
	}
	createRawBody := createRec.Body.String()
	var created openapi.CategoryResponse
	if err := json.NewDecoder(strings.NewReader(createRawBody)).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Data.Name != "Groceries" {
		t.Errorf("create: expected name %q, got %q", "Groceries", created.Data.Name)
	}
	if created.Data.PlaidId != nil {
		t.Errorf("create: expected plaidId nil, got %v", *created.Data.PlaidId)
	}
	if created.Data.UserId != userID.String() {
		t.Errorf("create: expected userId %q, got %q", userID.String(), created.Data.UserId)
	}
	// Ids keep the legacy cuid2 format (post-migration improvement 0002
	// defers UUIDv7 convergence); a UUID here fails this pattern.
	if !legacyResourceIDPattern.MatchString(created.Data.Id) {
		t.Errorf("create: expected a cuid2-format id matching %s, got %q", legacyResourceIDPattern, created.Data.Id)
	}
	if !strings.Contains(createRawBody, `"plaidId":null`) {
		t.Errorf("create: expected raw body to contain \"plaidId\":null, got %s", createRawBody)
	}
	categoryID := created.Data.Id

	// List (summary shape).
	listRec := env.do(t, http.MethodGet, "/api/categories", token, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d (body: %s)", listRec.Code, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "userId") || strings.Contains(listRec.Body.String(), "plaidId") {
		t.Errorf("list: expected summary shape ({id,name} only), got %s", listRec.Body.String())
	}
	var listed openapi.CategoryListResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Id != categoryID || listed.Data[0].Name != "Groceries" {
		t.Errorf("list: expected [{%q,Groceries}], got %+v", categoryID, listed.Data)
	}

	// Get (summary shape).
	getRec := env.do(t, http.MethodGet, "/api/categories/"+categoryID, token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d (body: %s)", getRec.Code, getRec.Body.String())
	}
	if strings.Contains(getRec.Body.String(), "userId") || strings.Contains(getRec.Body.String(), "plaidId") {
		t.Errorf("get: expected summary shape, got %s", getRec.Body.String())
	}
	var got openapi.CategorySummaryResponse
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.Data.Id != categoryID || got.Data.Name != "Groceries" {
		t.Errorf("get: expected {%q,Groceries}, got %+v", categoryID, got.Data)
	}

	// Update (full row shape).
	updateRec := env.do(t, http.MethodPatch, "/api/categories/"+categoryID, token, categoryInputBody{Name: "Food"})
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (body: %s)", updateRec.Code, updateRec.Body.String())
	}
	var updated openapi.CategoryResponse
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Data.Id != categoryID || updated.Data.Name != "Food" || updated.Data.UserId != userID.String() {
		t.Errorf("update: expected {%q,Food,%q}, got %+v", categoryID, userID.String(), updated.Data)
	}
	if updated.Data.PlaidId != nil {
		t.Errorf("update: expected plaidId nil, got %v", *updated.Data.PlaidId)
	}

	// Delete.
	deleteRec := env.do(t, http.MethodDelete, "/api/categories/"+categoryID, token, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d (body: %s)", deleteRec.Code, deleteRec.Body.String())
	}
	var deleted openapi.DeletedResourceResponse
	if err := json.NewDecoder(deleteRec.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if deleted.Data.Id != categoryID {
		t.Errorf("delete: expected id %q, got %q", categoryID, deleted.Data.Id)
	}

	// Bulk-delete (on a freshly created category).
	bulkID := createTestCategory(t, env, token, "Transport")
	bulkRec := env.do(t, http.MethodPost, "/api/categories/bulk-delete", token, bulkDeleteBody{Ids: []string{bulkID}})
	if bulkRec.Code != http.StatusOK {
		t.Fatalf("bulk-delete: expected 200, got %d (body: %s)", bulkRec.Code, bulkRec.Body.String())
	}
	var bulked openapi.DeletedResourceListResponse
	if err := json.NewDecoder(bulkRec.Body).Decode(&bulked); err != nil {
		t.Fatalf("decode bulk-delete response: %v", err)
	}
	if len(bulked.Data) != 1 || bulked.Data[0].Id != bulkID {
		t.Errorf("bulk-delete: expected [{%q}], got %+v", bulkID, bulked.Data)
	}
}

func TestCategoriesLive_ListEmpty_ReturnsEmptyArrayNotNull(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "cat-empty")

	rec := env.do(t, http.MethodGet, "/api/categories", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	// Asserted on the raw body: a decoded []T hides the null-vs-[] difference
	// that breaks the frontend's .map.
	if body := strings.TrimSpace(rec.Body.String()); body != `{"data":[]}` {
		t.Errorf("expected {\"data\":[]}, got %s", body)
	}
}

// --- duplicates -----------------------------------------------------------

func TestCategoriesLive_DuplicateName_OnCreate(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "cat-dup-create")

	createTestCategory(t, env, token, "Groceries")

	t.Run("exact duplicate", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories", token, categoryInputBody{Name: "Groceries"})
		assertDuplicateCategoryName(t, rec)
	})

	// name is citext, so duplicate detection is case-insensitive per user
	// without any Go-side normalization.
	t.Run("case-differing duplicate", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories", token, categoryInputBody{Name: "groceries"})
		assertDuplicateCategoryName(t, rec)
	})
}

// TestCategoriesLive_DuplicateName_OnUpdate covers the one deliberate
// divergence from legacy in this ticket. Legacy categories.ts wraps its
// UPDATE in a bare catch with no 23505 branch, so renaming a category onto
// an existing name returns 500 (fixtures line 321). spec.md line 104
// requires that mismatch to be decided in the contract instead of inherited,
// and the contract declares 409 on updateCategory. A regression to 500 fails
// here.
func TestCategoriesLive_DuplicateName_OnUpdate(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "cat-dup-update")

	createTestCategory(t, env, token, "Groceries")
	otherID := createTestCategory(t, env, token, "Transport")

	t.Run("exact duplicate", func(t *testing.T) {
		rec := env.do(t, http.MethodPatch, "/api/categories/"+otherID, token, categoryInputBody{Name: "Groceries"})
		assertDuplicateCategoryName(t, rec)
	})

	t.Run("case-differing duplicate", func(t *testing.T) {
		rec := env.do(t, http.MethodPatch, "/api/categories/"+otherID, token, categoryInputBody{Name: "GROCERIES"})
		assertDuplicateCategoryName(t, rec)
	})
}

// --- not found ------------------------------------------------------------

func TestCategoriesLive_NotFound_Nonexistent(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "cat-404")

	missingID := "does-not-exist-" + uuid.NewString()

	t.Run("GET", func(t *testing.T) {
		assertCategoryNotFound(t, env.do(t, http.MethodGet, "/api/categories/"+missingID, token, nil))
	})
	t.Run("PATCH", func(t *testing.T) {
		assertCategoryNotFound(t, env.do(t, http.MethodPatch, "/api/categories/"+missingID, token, categoryInputBody{Name: "Nope"}))
	})
	t.Run("DELETE", func(t *testing.T) {
		assertCategoryNotFound(t, env.do(t, http.MethodDelete, "/api/categories/"+missingID, token, nil))
	})
}

// --- cross-user isolation -------------------------------------------------

func TestCategoriesLive_CrossUserIsolation(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "cat-isolation-a")
	_, tokenB := env.createAccountsTestUser(t, "cat-isolation-b")

	catA := createTestCategory(t, env, tokenA, "A's Groceries")
	catB := createTestCategory(t, env, tokenB, "B's Groceries")

	// A cannot GET/PATCH/DELETE B's category.
	assertCategoryNotFound(t, env.do(t, http.MethodGet, "/api/categories/"+catB, tokenA, nil))
	assertCategoryNotFound(t, env.do(t, http.MethodPatch, "/api/categories/"+catB, tokenA, categoryInputBody{Name: "Hijacked"}))
	assertCategoryNotFound(t, env.do(t, http.MethodDelete, "/api/categories/"+catB, tokenA, nil))

	// B's row survives A's failed attempts, unmodified.
	bCheckRec := env.do(t, http.MethodGet, "/api/categories/"+catB, tokenB, nil)
	if bCheckRec.Code != http.StatusOK {
		t.Fatalf("B re-fetch after A's failed attempts: expected 200, got %d (body: %s)", bCheckRec.Code, bCheckRec.Body.String())
	}
	var bCheck openapi.CategorySummaryResponse
	if err := json.NewDecoder(bCheckRec.Body).Decode(&bCheck); err != nil {
		t.Fatalf("decode B re-fetch: %v", err)
	}
	if bCheck.Data.Name != "B's Groceries" {
		t.Errorf("B's category name changed by A's failed patch: expected %q, got %q", "B's Groceries", bCheck.Data.Name)
	}

	// A's list never contains B's category.
	listRec := env.do(t, http.MethodGet, "/api/categories", tokenA, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("A list: expected 200, got %d", listRec.Code)
	}
	var listed openapi.CategoryListResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode A list: %v", err)
	}
	for _, item := range listed.Data {
		if item.Id == catB {
			t.Fatalf("A's list contains B's category id %q", catB)
		}
	}

	// Bulk-delete mixing A's id, B's id and garbage deletes only A's.
	garbageID := "garbage-" + uuid.NewString()
	bulkRec := env.do(t, http.MethodPost, "/api/categories/bulk-delete", tokenA,
		bulkDeleteBody{Ids: []string{catA, catB, garbageID}})
	if bulkRec.Code != http.StatusOK {
		t.Fatalf("bulk-delete: expected 200, got %d (body: %s)", bulkRec.Code, bulkRec.Body.String())
	}
	var bulked openapi.DeletedResourceListResponse
	if err := json.NewDecoder(bulkRec.Body).Decode(&bulked); err != nil {
		t.Fatalf("decode bulk-delete: %v", err)
	}
	if len(bulked.Data) != 1 || bulked.Data[0].Id != catA {
		t.Errorf("bulk-delete: expected only A's id [{%q}], got %+v", catA, bulked.Data)
	}

	// B's category still exists after A's bulk-delete named it.
	if rec := env.do(t, http.MethodGet, "/api/categories/"+catB, tokenB, nil); rec.Code != http.StatusOK {
		t.Errorf("B's category was deleted by A's bulk-delete: expected 200, got %d", rec.Code)
	}
}

// --- unauthorized ---------------------------------------------------------

func TestCategoriesLive_Unauthorized(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, validToken := env.createAccountsTestUser(t, "cat-unauth-seed")

	// A real category id, so the by-id routes' 401-before-404 ordering is
	// exercised against something that would otherwise be visible.
	categoryID := createTestCategory(t, env, validToken, "Seed")

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
		{http.MethodGet, "/api/categories", nil},
		{http.MethodPost, "/api/categories", categoryInputBody{Name: "X"}},
		{http.MethodGet, "/api/categories/" + categoryID, nil},
		{http.MethodPatch, "/api/categories/" + categoryID, categoryInputBody{Name: "X"}},
		{http.MethodDelete, "/api/categories/" + categoryID, nil},
		{http.MethodPost, "/api/categories/bulk-delete", bulkDeleteBody{Ids: []string{categoryID}}},
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
				if rec.Code != http.StatusUnauthorized {
					t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
				}
				apiErr := decodeAccountsAPIError(t, rec)
				if apiErr.Error.Code != tc.wantCode {
					t.Errorf("expected code %q, got %q", tc.wantCode, apiErr.Error.Code)
				}
			})
		}
	}
}

// --- validation -----------------------------------------------------------

func TestCategoriesLive_Validation(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "cat-validation")

	t.Run("create: unknown field", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories", token, map[string]string{"name": "X", "extra": "nope"})
		assertValidationError(t, rec)
	})
	t.Run("create: empty name", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories", token, categoryInputBody{Name: ""})
		assertValidationError(t, rec)
	})
	t.Run("create: missing name", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories", token, map[string]string{})
		assertValidationError(t, rec)
	})
	t.Run("update: unknown field", func(t *testing.T) {
		catID := createTestCategory(t, env, token, "update-unknown")
		rec := env.do(t, http.MethodPatch, "/api/categories/"+catID, token, map[string]string{"name": "New", "extra": "nope"})
		assertValidationError(t, rec)
	})
	t.Run("update: empty name", func(t *testing.T) {
		catID := createTestCategory(t, env, token, "update-empty")
		rec := env.do(t, http.MethodPatch, "/api/categories/"+catID, token, categoryInputBody{Name: ""})
		assertValidationError(t, rec)
	})
	t.Run("bulk-delete: missing ids", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories/bulk-delete", token, map[string]string{})
		assertValidationError(t, rec)
	})
	t.Run("bulk-delete: empty ids array", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories/bulk-delete", token, bulkDeleteBody{Ids: []string{}})
		assertValidationError(t, rec)
	})
	t.Run("bulk-delete: empty string id", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/categories/bulk-delete", token, bulkDeleteBody{Ids: []string{"ok", ""}})
		assertValidationError(t, rec)
	})
	// No byte cap on resource bodies (post-migration improvement 0013): a
	// large contract-valid body must be accepted, matching Hono.
	t.Run("create: large body is accepted, not capped", func(t *testing.T) {
		largeName := strings.Repeat("y", 128*1024)
		rec := env.do(t, http.MethodPost, "/api/categories", token, categoryInputBody{Name: largeName})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for a large contract-valid name, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

// --- FK behavior ----------------------------------------------------------

// TestCategoriesLive_DeleteSetsTransactionCategoryNull covers the acceptance
// criterion "Category deletion clears category_id on transactions", on BOTH
// delete paths.
//
// This is the inverse of accounts, where deleting cascades the transaction
// away entirely (transactions.account_id is ON DELETE CASCADE, category_id
// is ON DELETE SET NULL — migration 00002). A test that only asserted "the
// transaction is gone" would pass for the wrong reason here, so it asserts
// the transaction SURVIVES with a null category.
func TestCategoriesLive_DeleteSetsTransactionCategoryNull(t *testing.T) {
	env := newAccountsTestEnv(t)
	userID, token := env.createAccountsTestUser(t, "cat-fk")

	t.Run("single delete", func(t *testing.T) {
		categoryID := createTestCategory(t, env, token, "FK Single")
		accountID := createTestAccount(t, env, token, "FK Single Account")
		transactionID := seedTransaction(t, env.pool, userID, accountID, categoryID)

		rec := env.do(t, http.MethodDelete, "/api/categories/"+categoryID, token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete category: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		assertTransactionSurvivesWithNullCategory(t, env.pool, userID, transactionID)
	})

	t.Run("bulk delete", func(t *testing.T) {
		categoryID := createTestCategory(t, env, token, "FK Bulk")
		accountID := createTestAccount(t, env, token, "FK Bulk Account")
		transactionID := seedTransaction(t, env.pool, userID, accountID, categoryID)

		rec := env.do(t, http.MethodPost, "/api/categories/bulk-delete", token, bulkDeleteBody{Ids: []string{categoryID}})
		if rec.Code != http.StatusOK {
			t.Fatalf("bulk-delete category: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		assertTransactionSurvivesWithNullCategory(t, env.pool, userID, transactionID)
	})
}

// seedTransaction inserts a transaction referencing accountID and
// categoryID, bypassing the (not yet migrated) transactions API.
//
// It must go through db.WithUserTx: transactions_owner's WITH CHECK
// requires a bound app.user_id, so a bare pool insert is rejected with
// SQLSTATE 42501 rather than silently succeeding.
func seedTransaction(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, accountID, categoryID string) string {
	t.Helper()

	ctx := context.Background()
	transactionID := "txn-" + uuid.NewString()
	err := db.WithUserTx(ctx, pool, userID, func(q *dbgen.Queries) error {
		_, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
			ID:         transactionID,
			Amount:     1000,
			Payee:      "FK payee",
			Date:       pgtype.Timestamp{Time: time.Now(), Valid: true},
			AccountID:  accountID,
			CategoryID: pgtype.Text{String: categoryID, Valid: true},
			Currency:   "ARS",
		})
		return err
	})
	if err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
	return transactionID
}

// assertTransactionSurvivesWithNullCategory verifies ON DELETE SET NULL:
// the transaction row still exists and its category_id is NULL. Read
// through db.WithUserTx so the RLS policies apply exactly as they do in the
// request path (a bare pool read would see zero rows regardless).
func assertTransactionSurvivesWithNullCategory(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, transactionID string) {
	t.Helper()

	ctx := context.Background()
	var transaction dbgen.Transaction
	var found bool
	err := db.WithUserTx(ctx, pool, userID, func(q *dbgen.Queries) error {
		row, err := q.GetTransaction(ctx, dbgen.GetTransactionParams{ID: transactionID, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		transaction = row
		found = true
		return nil
	})
	if err != nil {
		t.Fatalf("read transaction after category delete: %v", err)
	}

	if !found {
		t.Fatalf("transaction %q was deleted; category_id is ON DELETE SET NULL, so the transaction must survive", transactionID)
	}
	if transaction.CategoryID.Valid {
		t.Errorf("expected category_id NULL after category delete, got %q", transaction.CategoryID.String)
	}
}
