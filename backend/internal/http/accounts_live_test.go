package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/config"
	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/mail"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// legacyResourceIDPattern is the cuid2 shape legacy finance ids have and
// that the Go replacement must keep producing for parity: 24 characters, a
// leading lowercase letter, the rest base36. A UUID fails this pattern,
// which is the regression it exists to catch.
var legacyResourceIDPattern = regexp.MustCompile(`^[a-z][0-9a-z]{23}$`)

// accountsTestEnv wires a live pgxpool and a router mounting the real
// AuthServer + ResourceServer, following the same TEST_DATABASE_URL skip
// convention as authTestEnv (auth_live_test.go). Only the ResourceServer's
// account handlers are exercised here; users are created and tokens issued
// directly (bypassing the /api/auth/* HTTP flow, already covered in
// auth_live_test.go) to keep these tests focused on #44.
type accountsTestEnv struct {
	pool   *pgxpool.Pool
	router http.Handler
	secret []byte
}

func newAccountsTestEnv(t *testing.T) accountsTestEnv {
	t.Helper()

	databaseURL := liveDatabaseURL(t, "live accounts HTTP test")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	secret := []byte("accounts-live-test-secret-at-least-32-bytes!!")
	cfg := config.Config{
		JWTSecret:       secret,
		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
		CookieSecure:    false,

		VerificationTokenTTL: 48 * time.Hour,
		ResetTokenTTL:        30 * time.Minute,
	}

	authServer := NewAuthServer(pool, cfg, mail.NewCapturingMailer())
	resourceServer := NewResourceServer(pool)
	router := NewRouter(authServer, resourceServer)

	return accountsTestEnv{pool: pool, router: router, secret: secret}
}

// do issues an HTTP request against the env's router. token is sent as a
// Bearer Authorization header when non-empty; empty means no header at all
// (the "unauthorized: no header" case).
func (e accountsTestEnv) do(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case []byte:
		reader = bytes.NewReader(v)
	default:
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// createAccountsTestUser inserts a user directly (mirrors
// createProbeTestUser in middleware_live_test.go) and registers its
// cleanup. Returns the user id and a valid, long-lived access token for it.
func (e accountsTestEnv) createAccountsTestUser(t *testing.T, label string) (userID uuid.UUID, token string) {
	t.Helper()

	ctx := context.Background()
	email := fmt.Sprintf("accounts-http-%s-%s@example.test", label, uuid.NewString())
	var id string
	row := e.pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, 'test-hash') RETURNING id`, email)
	if err := row.Scan(&id); err != nil {
		t.Fatalf("insert test user %q: %v", label, err)
	}
	userID = uuid.MustParse(id)

	t.Cleanup(func() {
		if _, err := e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleanup: failed to delete test user %q: %v", userID, err)
		}
	})

	token, _, err := auth.IssueAccessToken(e.secret, userID, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken for %q: %v", label, err)
	}
	return userID, token
}

func decodeAccountsAPIError(t *testing.T, rec *httptest.ResponseRecorder) openapi.ApiErrorResponse {
	t.Helper()
	var out openapi.ApiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode error response: %v (body: %s)", err, rec.Body.String())
	}
	return out
}

// --- happy path -----------------------------------------------------------

// TestAccountsLive_HappyPath_AllOperations covers acceptance criterion 1:
// every one of the six operations, asserting exact response bodies
// including the list/get summary vs create/update full-row asymmetry and
// plaidId: null on create (briefing §3, design decision 6.6).
func TestAccountsLive_HappyPath_AllOperations(t *testing.T) {
	env := newAccountsTestEnv(t)
	userID, token := env.createAccountsTestUser(t, "happy")

	// Create.
	createRec := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: "Cash"})
	if createRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d (body: %s)", createRec.Code, createRec.Body.String())
	}
	createRawBody := createRec.Body.String()
	var created openapi.AccountResponse
	if err := json.NewDecoder(strings.NewReader(createRawBody)).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Data.Name != "Cash" {
		t.Errorf("create: expected name %q, got %q", "Cash", created.Data.Name)
	}
	if created.Data.PlaidId != nil {
		t.Errorf("create: expected plaidId nil, got %v", *created.Data.PlaidId)
	}
	if created.Data.UserId != userID.String() {
		t.Errorf("create: expected userId %q, got %q", userID.String(), created.Data.UserId)
	}
	// Ids must keep the legacy cuid2 format: the finance tables hold
	// cuid-style text primary keys, and converging them on UUIDv7 is
	// deferred to post-migration improvement 0002. A UUID here would be an
	// observable change and would mix id formats within one table.
	if !legacyResourceIDPattern.MatchString(created.Data.Id) {
		t.Errorf("create: expected a cuid2-format id matching %s, got %q", legacyResourceIDPattern, created.Data.Id)
	}
	// plaidId must be present as JSON null, never omitted.
	if !strings.Contains(createRawBody, `"plaidId":null`) {
		t.Errorf("create: expected raw body to contain \"plaidId\":null, got %s", createRawBody)
	}
	accountID := created.Data.Id

	// List (summary shape).
	listRec := env.do(t, http.MethodGet, "/api/accounts", token, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d (body: %s)", listRec.Code, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "userId") || strings.Contains(listRec.Body.String(), "plaidId") {
		t.Errorf("list: expected summary shape ({id,name} only), got %s", listRec.Body.String())
	}
	var listed openapi.AccountListResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Id != accountID || listed.Data[0].Name != "Cash" {
		t.Errorf("list: expected [{%q,Cash}], got %+v", accountID, listed.Data)
	}

	// Get (summary shape).
	getRec := env.do(t, http.MethodGet, "/api/accounts/"+accountID, token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d (body: %s)", getRec.Code, getRec.Body.String())
	}
	if strings.Contains(getRec.Body.String(), "userId") || strings.Contains(getRec.Body.String(), "plaidId") {
		t.Errorf("get: expected summary shape, got %s", getRec.Body.String())
	}
	var got openapi.AccountSummaryResponse
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.Data.Id != accountID || got.Data.Name != "Cash" {
		t.Errorf("get: expected {%q,Cash}, got %+v", accountID, got.Data)
	}

	// Update (full row shape).
	updateRec := env.do(t, http.MethodPatch, "/api/accounts/"+accountID, token, accountInputBody{Name: "Checking"})
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (body: %s)", updateRec.Code, updateRec.Body.String())
	}
	var updated openapi.AccountResponse
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Data.Id != accountID || updated.Data.Name != "Checking" || updated.Data.UserId != userID.String() {
		t.Errorf("update: expected {%q,Checking,%q}, got %+v", accountID, userID.String(), updated.Data)
	}
	if updated.Data.PlaidId != nil {
		t.Errorf("update: expected plaidId nil, got %v", *updated.Data.PlaidId)
	}

	// Bulk-delete.
	bulkRec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", token, bulkDeleteBody{Ids: []string{accountID}})
	if bulkRec.Code != http.StatusOK {
		t.Fatalf("bulk-delete: expected 200, got %d (body: %s)", bulkRec.Code, bulkRec.Body.String())
	}
	var bulked openapi.DeletedResourceListResponse
	if err := json.NewDecoder(bulkRec.Body).Decode(&bulked); err != nil {
		t.Fatalf("decode bulk-delete response: %v", err)
	}
	if len(bulked.Data) != 1 || bulked.Data[0].Id != accountID {
		t.Errorf("bulk-delete: expected [{%q}], got %+v", accountID, bulked.Data)
	}

	// The account is gone now; recreate one to exercise single delete.
	create2Rec := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: "Savings"})
	if create2Rec.Code != http.StatusOK {
		t.Fatalf("create #2: expected 200, got %d (body: %s)", create2Rec.Code, create2Rec.Body.String())
	}
	var created2 openapi.AccountResponse
	if err := json.NewDecoder(create2Rec.Body).Decode(&created2); err != nil {
		t.Fatalf("decode create #2 response: %v", err)
	}

	deleteRec := env.do(t, http.MethodDelete, "/api/accounts/"+created2.Data.Id, token, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d (body: %s)", deleteRec.Code, deleteRec.Body.String())
	}
	var deleted openapi.DeletedResourceResponse
	if err := json.NewDecoder(deleteRec.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if deleted.Data.Id != created2.Data.Id {
		t.Errorf("delete: expected id %q, got %q", created2.Data.Id, deleted.Data.Id)
	}
}

// TestAccountsLive_ListEmpty_ReturnsEmptyArrayNotNull covers acceptance
// criterion 2, asserting the raw JSON (not a decoded slice, which hides
// null vs [] — design decision 6.13).
func TestAccountsLive_ListEmpty_ReturnsEmptyArrayNotNull(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "list-empty")

	rec := env.do(t, http.MethodGet, "/api/accounts", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	got := strings.TrimSpace(rec.Body.String())
	if got != `{"data":[]}` {
		t.Errorf("expected raw body %q, got %q", `{"data":[]}`, got)
	}
}

// --- duplicates -------------------------------------------------------------

// TestAccountsLive_DuplicateName_OnCreate covers acceptance criterion 3:
// duplicate name 409s with details.constraint, including a differing-case
// duplicate proving citext case-insensitivity (design decision 6.7).
func TestAccountsLive_DuplicateName_OnCreate(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "dup-create")

	rec1 := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: "cash"})
	if rec1.Code != http.StatusOK {
		t.Fatalf("first create: expected 200, got %d (body: %s)", rec1.Code, rec1.Body.String())
	}

	cases := []struct {
		name  string
		value string
	}{
		{"exact duplicate", "cash"},
		{"case-differing duplicate", "Cash"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: tc.value})
			if rec.Code != http.StatusConflict {
				t.Fatalf("expected 409, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			apiErr := decodeAccountsAPIError(t, rec)
			if apiErr.Error.Code != "DUPLICATE_ACCOUNT_NAME" {
				t.Errorf("expected code DUPLICATE_ACCOUNT_NAME, got %q", apiErr.Error.Code)
			}
			if apiErr.Error.Details == nil {
				t.Fatal("expected details.constraint, got nil details")
			}
			constraint, _ := (*apiErr.Error.Details)["constraint"].(string)
			if constraint != "accounts_user_id_name_uniq" {
				t.Errorf("expected details.constraint %q, got %q", "accounts_user_id_name_uniq", constraint)
			}
		})
	}
}

// TestAccountsLive_DuplicateName_OnUpdate covers acceptance criterion 4.
func TestAccountsLive_DuplicateName_OnUpdate(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "dup-update")

	rec1 := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: "Cash"})
	if rec1.Code != http.StatusOK {
		t.Fatalf("create #1: expected 200, got %d", rec1.Code)
	}

	var acc2Body openapi.AccountResponse
	rec2 := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: "Checking"})
	if rec2.Code != http.StatusOK {
		t.Fatalf("create #2: expected 200, got %d", rec2.Code)
	}
	if err := json.NewDecoder(rec2.Body).Decode(&acc2Body); err != nil {
		t.Fatalf("decode create #2: %v", err)
	}

	rec := env.do(t, http.MethodPatch, "/api/accounts/"+acc2Body.Data.Id, token, accountInputBody{Name: "Cash"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAccountsAPIError(t, rec)
	if apiErr.Error.Code != "DUPLICATE_ACCOUNT_NAME" {
		t.Errorf("expected code DUPLICATE_ACCOUNT_NAME, got %q", apiErr.Error.Code)
	}
}

// --- not found --------------------------------------------------------------

// TestAccountsLive_NotFound_Nonexistent covers acceptance criterion 5.
func TestAccountsLive_NotFound_Nonexistent(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "not-found")

	missingID := "does-not-exist-" + uuid.NewString()

	t.Run("GET", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/api/accounts/"+missingID, token, nil)
		assertAccountNotFound(t, rec)
	})
	t.Run("PATCH", func(t *testing.T) {
		rec := env.do(t, http.MethodPatch, "/api/accounts/"+missingID, token, accountInputBody{Name: "Whatever"})
		assertAccountNotFound(t, rec)
	})
	t.Run("DELETE", func(t *testing.T) {
		rec := env.do(t, http.MethodDelete, "/api/accounts/"+missingID, token, nil)
		assertAccountNotFound(t, rec)
	})
}

func assertAccountNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAccountsAPIError(t, rec)
	if apiErr.Error.Code != "ACCOUNT_NOT_FOUND" {
		t.Errorf("expected code ACCOUNT_NOT_FOUND, got %q", apiErr.Error.Code)
	}
}

// --- cross-user isolation ----------------------------------------------------

// TestAccountsLive_CrossUserIsolation covers acceptance criterion 6: user
// B's account id on GET/PATCH/DELETE 404s and leaves B's row untouched;
// A's list never contains B's rows; bulk-delete with a mix of A's ids, B's
// ids, and garbage deletes only A's and returns only A's ids.
func TestAccountsLive_CrossUserIsolation(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, tokenA := env.createAccountsTestUser(t, "isolation-a")
	_, tokenB := env.createAccountsTestUser(t, "isolation-b")

	// Seed one account each for A and B.
	recA := env.do(t, http.MethodPost, "/api/accounts", tokenA, accountInputBody{Name: "A's Cash"})
	if recA.Code != http.StatusOK {
		t.Fatalf("create A: expected 200, got %d (body: %s)", recA.Code, recA.Body.String())
	}
	var accA openapi.AccountResponse
	if err := json.NewDecoder(recA.Body).Decode(&accA); err != nil {
		t.Fatalf("decode create A: %v", err)
	}

	recB := env.do(t, http.MethodPost, "/api/accounts", tokenB, accountInputBody{Name: "B's Cash"})
	if recB.Code != http.StatusOK {
		t.Fatalf("create B: expected 200, got %d (body: %s)", recB.Code, recB.Body.String())
	}
	var accB openapi.AccountResponse
	if err := json.NewDecoder(recB.Body).Decode(&accB); err != nil {
		t.Fatalf("decode create B: %v", err)
	}

	// A cannot GET/PATCH/DELETE B's account.
	getRec := env.do(t, http.MethodGet, "/api/accounts/"+accB.Data.Id, tokenA, nil)
	assertAccountNotFound(t, getRec)

	patchRec := env.do(t, http.MethodPatch, "/api/accounts/"+accB.Data.Id, tokenA, accountInputBody{Name: "Hijacked"})
	assertAccountNotFound(t, patchRec)

	deleteRec := env.do(t, http.MethodDelete, "/api/accounts/"+accB.Data.Id, tokenA, nil)
	assertAccountNotFound(t, deleteRec)

	// B's row is untouched by A's failed attempts.
	bCheckRec := env.do(t, http.MethodGet, "/api/accounts/"+accB.Data.Id, tokenB, nil)
	if bCheckRec.Code != http.StatusOK {
		t.Fatalf("B re-fetch after A's failed attempts: expected 200, got %d (body: %s)", bCheckRec.Code, bCheckRec.Body.String())
	}
	var bCheck openapi.AccountSummaryResponse
	if err := json.NewDecoder(bCheckRec.Body).Decode(&bCheck); err != nil {
		t.Fatalf("decode B re-fetch: %v", err)
	}
	if bCheck.Data.Name != "B's Cash" {
		t.Errorf("B's account name changed by A's failed patch attempt: expected %q, got %q", "B's Cash", bCheck.Data.Name)
	}

	// A's list never contains B's account.
	listRec := env.do(t, http.MethodGet, "/api/accounts", tokenA, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("A list: expected 200, got %d", listRec.Code)
	}
	var listed openapi.AccountListResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode A list: %v", err)
	}
	for _, item := range listed.Data {
		if item.Id == accB.Data.Id {
			t.Fatalf("A's list contains B's account id %q", accB.Data.Id)
		}
	}

	// Bulk-delete with a mix of A's id, B's id, and a garbage id deletes
	// only A's and returns only A's id.
	garbageID := "garbage-" + uuid.NewString()
	bulkRec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", tokenA,
		bulkDeleteBody{Ids: []string{accA.Data.Id, accB.Data.Id, garbageID}})
	if bulkRec.Code != http.StatusOK {
		t.Fatalf("bulk-delete: expected 200, got %d (body: %s)", bulkRec.Code, bulkRec.Body.String())
	}
	var bulked openapi.DeletedResourceListResponse
	if err := json.NewDecoder(bulkRec.Body).Decode(&bulked); err != nil {
		t.Fatalf("decode bulk-delete: %v", err)
	}
	if len(bulked.Data) != 1 || bulked.Data[0].Id != accA.Data.Id {
		t.Errorf("bulk-delete: expected only [{%q}], got %+v", accA.Data.Id, bulked.Data)
	}

	// B's row survives A's bulk-delete.
	bSurviveRec := env.do(t, http.MethodGet, "/api/accounts/"+accB.Data.Id, tokenB, nil)
	if bSurviveRec.Code != http.StatusOK {
		t.Fatalf("B's account after A's bulk-delete: expected 200 (still present), got %d (body: %s)", bSurviveRec.Code, bSurviveRec.Body.String())
	}
}

// --- unauthorized -------------------------------------------------------------

// TestAccountsLive_Unauthorized covers acceptance criterion 7 across every
// route: no header, malformed header, bad signature, and an expired token.
func TestAccountsLive_Unauthorized(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, validToken := env.createAccountsTestUser(t, "unauth-seed")

	// A real account id so a by-id route's 404-vs-401 ordering is exercised
	// against something that would otherwise be visible.
	seedRec := env.do(t, http.MethodPost, "/api/accounts", validToken, accountInputBody{Name: "Seed"})
	if seedRec.Code != http.StatusOK {
		t.Fatalf("seed create: expected 200, got %d", seedRec.Code)
	}
	var seeded openapi.AccountResponse
	if err := json.NewDecoder(seedRec.Body).Decode(&seeded); err != nil {
		t.Fatalf("decode seed create: %v", err)
	}
	accountID := seeded.Data.Id

	wrongSecretToken, _, err := auth.IssueAccessToken([]byte("a-totally-different-secret-32-bytes!!"), uuid.New(), 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken wrong secret: %v", err)
	}
	expiredToken, _, err := auth.IssueAccessToken(env.secret, uuid.New(), -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken expired: %v", err)
	}

	type route struct {
		method string
		path   string
		body   any
	}
	routes := []route{
		{http.MethodGet, "/api/accounts", nil},
		{http.MethodPost, "/api/accounts", accountInputBody{Name: "X"}},
		{http.MethodGet, "/api/accounts/" + accountID, nil},
		{http.MethodPatch, "/api/accounts/" + accountID, accountInputBody{Name: "X"}},
		{http.MethodDelete, "/api/accounts/" + accountID, nil},
		{http.MethodPost, "/api/accounts/bulk-delete", bulkDeleteBody{Ids: []string{accountID}}},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path+"/no-header", func(t *testing.T) {
			rec := env.do(t, rt.method, rt.path, "", rt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			apiErr := decodeAccountsAPIError(t, rec)
			if apiErr.Error.Code != "UNAUTHORIZED" {
				t.Errorf("expected code UNAUTHORIZED, got %q", apiErr.Error.Code)
			}
		})

		t.Run(rt.method+" "+rt.path+"/malformed-header", func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, requestBodyReader(t, rt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Basic not-a-bearer-token")
			rec := httptest.NewRecorder()
			env.router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
			}
		})

		t.Run(rt.method+" "+rt.path+"/bad-signature", func(t *testing.T) {
			rec := env.do(t, rt.method, rt.path, wrongSecretToken, rt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			apiErr := decodeAccountsAPIError(t, rec)
			if apiErr.Error.Code != "UNAUTHORIZED" {
				t.Errorf("expected code UNAUTHORIZED, got %q", apiErr.Error.Code)
			}
		})

		t.Run(rt.method+" "+rt.path+"/expired", func(t *testing.T) {
			rec := env.do(t, rt.method, rt.path, expiredToken, rt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			apiErr := decodeAccountsAPIError(t, rec)
			if apiErr.Error.Code != "ACCESS_TOKEN_EXPIRED" {
				t.Errorf("expected code ACCESS_TOKEN_EXPIRED, got %q", apiErr.Error.Code)
			}
		})
	}
}

func requestBodyReader(t *testing.T, body any) *bytes.Reader {
	t.Helper()
	if body == nil {
		return bytes.NewReader(nil)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return bytes.NewReader(encoded)
}

// --- validation ---------------------------------------------------------------

// TestAccountsLive_Validation covers acceptance criterion 8.
func TestAccountsLive_Validation(t *testing.T) {
	env := newAccountsTestEnv(t)
	_, token := env.createAccountsTestUser(t, "validation")

	t.Run("create: unknown field", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts", token, map[string]string{"name": "Cash", "extra": "nope"})
		assertValidationError(t, rec)
	})
	t.Run("create: empty name", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: ""})
		assertValidationError(t, rec)
	})
	t.Run("create: missing name", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts", token, map[string]string{})
		assertValidationError(t, rec)
	})
	// A large-but-contract-valid body must be ACCEPTED, not rejected: neither
	// the legacy Hono validators nor AccountInput declare any maxLength or
	// byte cap, so Hono accepts and stores a name this size. Freezing that
	// here guards against an undocumented API limit creeping back in.
	t.Run("create: large body is accepted, not capped", func(t *testing.T) {
		largeName := strings.Repeat("x", 128*1024)
		rec := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: largeName})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for a large contract-valid name, got %d (body: %s)", rec.Code, rec.Body.String())
		}
		var created openapi.AccountResponse
		if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		if created.Data.Name != largeName {
			t.Errorf("expected the full %d-character name to round-trip, got %d characters", len(largeName), len(created.Data.Name))
		}
	})

	t.Run("update: unknown field", func(t *testing.T) {
		accID := createTestAccount(t, env, token, "update-unknown")
		rec := env.do(t, http.MethodPatch, "/api/accounts/"+accID, token, map[string]string{"name": "New", "extra": "nope"})
		assertValidationError(t, rec)
	})
	t.Run("update: empty name", func(t *testing.T) {
		accID := createTestAccount(t, env, token, "update-empty")
		rec := env.do(t, http.MethodPatch, "/api/accounts/"+accID, token, accountInputBody{Name: ""})
		assertValidationError(t, rec)
	})

	t.Run("bulk-delete: missing ids", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", token, map[string]string{})
		assertValidationError(t, rec)
	})
	t.Run("bulk-delete: empty ids array", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", token, bulkDeleteBody{Ids: []string{}})
		assertValidationError(t, rec)
	})
	t.Run("bulk-delete: empty string id", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", token, bulkDeleteBody{Ids: []string{"ok", ""}})
		assertValidationError(t, rec)
	})
	t.Run("bulk-delete: unknown field", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", token, map[string]any{"ids": []string{"x"}, "extra": "nope"})
		assertValidationError(t, rec)
	})
	// Same guard on the other side: a bulk-delete far larger than the
	// frontend's 500-item chunk (a client detail, not an API constraint) must
	// be processed, not rejected. The ids are unowned, so it deletes nothing
	// and returns an empty list.
	t.Run("bulk-delete: large id list is accepted, not capped", func(t *testing.T) {
		ids := make([]string, 0, 4000)
		for i := range 4000 {
			ids = append(ids, fmt.Sprintf("padding-id-%d-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", i))
		}
		rec := env.do(t, http.MethodPost, "/api/accounts/bulk-delete", token, bulkDeleteBody{Ids: ids})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for a large bulk-delete body, got %d (body: %s)", rec.Code, rec.Body.String())
		}
		if body := rec.Body.String(); !strings.Contains(body, `"data":[]`) {
			t.Errorf("expected an empty data array for unowned ids, got %s", body)
		}
	})
}

func createTestAccount(t *testing.T, env accountsTestEnv, token, name string) string {
	t.Helper()
	rec := env.do(t, http.MethodPost, "/api/accounts", token, accountInputBody{Name: name})
	if rec.Code != http.StatusOK {
		t.Fatalf("seed create %q: expected 200, got %d (body: %s)", name, rec.Code, rec.Body.String())
	}
	var created openapi.AccountResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode seed create %q: %v", name, err)
	}
	return created.Data.Id
}

func assertValidationError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAccountsAPIError(t, rec)
	if apiErr.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
	}
}

// --- cascade ------------------------------------------------------------------

// TestAccountsLive_DeleteCascadesTransactions covers acceptance criterion
// 9: deleting an account cascades its transactions through the database
// foreign key (00002_finance_base.sql), not application code.
func TestAccountsLive_DeleteCascadesTransactions(t *testing.T) {
	env := newAccountsTestEnv(t)
	userID, token := env.createAccountsTestUser(t, "cascade")

	accountID := createTestAccount(t, env, token, "Cascade Account")

	// Transactions carry FORCE RLS too, so seeding a row for the test needs
	// the same RLS-bound insert path production code uses (db.WithUserTx),
	// not a bare pool.Exec — matching the general rule this ticket enforces
	// for every owned-table query.
	transactionID := "txn-" + uuid.NewString()
	ctx := context.Background()
	if err := db.WithUserTx(ctx, env.pool, userID, func(q *dbgen.Queries) error {
		_, err := q.CreateTransaction(ctx, dbgen.CreateTransactionParams{
			ID:        transactionID,
			Amount:    1000,
			Payee:     "Test Payee",
			Date:      pgtype.Timestamp{Time: time.Now(), Valid: true},
			AccountID: accountID,
			Currency:  "ARS",
		})
		return err
	}); err != nil {
		t.Fatalf("insert test transaction: %v", err)
	}

	if !userTransactionExists(t, env.pool, userID, transactionID) {
		t.Fatal("expected the seeded transaction to exist before delete")
	}

	deleteRec := env.do(t, http.MethodDelete, "/api/accounts/"+accountID, token, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete account: expected 200, got %d (body: %s)", deleteRec.Code, deleteRec.Body.String())
	}

	if userTransactionExists(t, env.pool, userID, transactionID) {
		t.Error("expected the transaction to be cascade-deleted with its account, but it still exists")
	}
}

// userTransactionExists reports whether a transaction with the given id is
// visible to userID, through the same RLS-bound db.WithUserTx path
// production code uses (GetTransaction, joined through the owning
// account): a bare pool query would see nothing regardless of what exists,
// since transactions carries FORCE RLS and an unbound connection has no
// app.user_id to satisfy the policy.
func userTransactionExists(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, transactionID string) bool {
	t.Helper()
	ctx := context.Background()
	var exists bool
	err := db.WithUserTx(ctx, pool, userID, func(q *dbgen.Queries) error {
		_, err := q.GetTransaction(ctx, dbgen.GetTransactionParams{ID: transactionID, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			exists = false
			return nil
		}
		if err != nil {
			return err
		}
		exists = true
		return nil
	})
	if err != nil {
		t.Fatalf("check transaction %q existence: %v", transactionID, err)
	}
	return exists
}
