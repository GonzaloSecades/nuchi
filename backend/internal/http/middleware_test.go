package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var middlewareTestSecret = []byte("middleware-test-secret-at-least-32-bytes!!")

// newProbeRouter mounts probe behind RequireAuth(secret) on both GET and
// POST /probe, standing in for the "minimal test-only protected probe
// route" the #43 briefing calls for: NewRouter's own RequireAuth group is
// intentionally empty until #44-#48 mount real resource routes into it, so
// exercising the middleware end-to-end needs its own tiny router here.
func newProbeRouter(secret []byte, probe http.HandlerFunc) http.Handler {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(RequireAuth(secret))
		r.Get("/probe", probe)
		r.Post("/probe", probe)
	})
	return router
}

// contextEchoProbe writes the context user id (or "none" if absent) as the
// response body, so tests can assert on exactly what RequireAuth put on the
// request context.
func contextEchoProbe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("none"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(userID.String()))
}

func decodeMiddlewareAPIError(t *testing.T, rec *httptest.ResponseRecorder) openapi.ApiErrorResponse {
	t.Helper()
	var out openapi.ApiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode error response: %v (body: %s)", err, rec.Body.String())
	}
	return out
}

func doProbeRequest(t *testing.T, router http.Handler, method, authHeader string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, "/probe", bytes.NewReader(body))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestRequireAuth_MissingHeader_Unauthorized covers acceptance criterion 1:
// no Authorization header at all.
func TestRequireAuth_MissingHeader_Unauthorized(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	rec := doProbeRequest(t, router, http.MethodGet, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	got := decodeMiddlewareAPIError(t, rec)
	if got.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got %q", got.Error.Code)
	}
	if got.Error.Message != "Authentication required." {
		t.Errorf("expected message %q, got %q", "Authentication required.", got.Error.Message)
	}
}

// TestRequireAuth_MalformedHeader_Unauthorized covers acceptance criterion
// 2: headers that are present but not a well-formed Bearer token.
func TestRequireAuth_MalformedHeader_Unauthorized(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	cases := []string{
		"Bearer",
		"Bearer ",
		"Basic xyz",
		"Bearer a.b.c",
		"bearer sometoken",
	}
	for _, header := range cases {
		t.Run(header, func(t *testing.T) {
			rec := doProbeRequest(t, router, http.MethodGet, header, nil)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("header %q: expected 401, got %d (body: %s)", header, rec.Code, rec.Body.String())
			}
			got := decodeMiddlewareAPIError(t, rec)
			if got.Error.Code != "UNAUTHORIZED" {
				t.Errorf("header %q: expected code UNAUTHORIZED, got %q", header, got.Error.Code)
			}
		})
	}
}

// TestRequireAuth_WrongSecret_Unauthorized covers acceptance criterion 3.
func TestRequireAuth_WrongSecret_Unauthorized(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	token, _, err := auth.IssueAccessToken([]byte("a-completely-different-secret-32bytes!!"), uuid.New(), 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	rec := doProbeRequest(t, router, http.MethodGet, "Bearer "+token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	got := decodeMiddlewareAPIError(t, rec)
	if got.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got %q", got.Error.Code)
	}
}

// TestRequireAuth_ExpiredToken_AccessTokenExpired covers acceptance
// criterion 4: the one deliberate carve-out (design decision 4e).
func TestRequireAuth_ExpiredToken_AccessTokenExpired(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	token, _, err := auth.IssueAccessToken(middlewareTestSecret, uuid.New(), -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	rec := doProbeRequest(t, router, http.MethodGet, "Bearer "+token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	got := decodeMiddlewareAPIError(t, rec)
	if got.Error.Code != "ACCESS_TOKEN_EXPIRED" {
		t.Errorf("expected code ACCESS_TOKEN_EXPIRED, got %q", got.Error.Code)
	}
	if got.Error.Message != "Access token expired." {
		t.Errorf("expected message %q, got %q", "Access token expired.", got.Error.Message)
	}
}

// TestRequireAuth_RejectsAlgPolicyViolations covers acceptance criterion 5:
// alg=none and HS512 tokens, an alg-policy regression guard riding on top
// of VerifyAccessToken's own coverage (internal/auth/jwt_test.go).
func TestRequireAuth_RejectsAlgPolicyViolations(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	userID := uuid.New()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build alg=none fixture: %v", err)
	}
	hs512Token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(middlewareTestSecret)
	if err != nil {
		t.Fatalf("failed to build HS512 fixture: %v", err)
	}

	for name, token := range map[string]string{"alg=none": noneToken, "HS512": hs512Token} {
		t.Run(name, func(t *testing.T) {
			rec := doProbeRequest(t, router, http.MethodGet, "Bearer "+token, nil)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s: expected 401, got %d (body: %s)", name, rec.Code, rec.Body.String())
			}
			got := decodeMiddlewareAPIError(t, rec)
			if got.Error.Code != "UNAUTHORIZED" {
				t.Errorf("%s: expected code UNAUTHORIZED, got %q", name, got.Error.Code)
			}
		})
	}
}

// TestRequireAuth_ValidToken_ExposesUserID covers acceptance criterion 6:
// a valid token reaches the handler, which observes the exact UUID the
// token carried.
func TestRequireAuth_ValidToken_ExposesUserID(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	userID := uuid.New()
	token, _, err := auth.IssueAccessToken(middlewareTestSecret, userID, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	rec := doProbeRequest(t, router, http.MethodGet, "Bearer "+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != userID.String() {
		t.Errorf("expected handler to observe user id %q, got %q", userID.String(), got)
	}
}

// TestRequireAuth_BodyIdentityIsIgnored covers acceptance criterion 9: a
// request body claiming to be a different user must not influence which
// user the handler believes is authenticated. contextEchoProbe never reads
// the body at all, which is itself the proof — RequireAuth's only identity
// source is the Authorization header — but this test pins the end-to-end
// behavior an attacker would actually attempt: send someone else's id in
// the body under your own valid token.
func TestRequireAuth_BodyIdentityIsIgnored(t *testing.T) {
	router := newProbeRouter(middlewareTestSecret, contextEchoProbe)

	realUser := uuid.New()
	impersonatedUser := uuid.New()
	token, _, err := auth.IssueAccessToken(middlewareTestSecret, realUser, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken: unexpected error: %v", err)
	}

	body, err := json.Marshal(map[string]string{"user_id": impersonatedUser.String(), "userId": impersonatedUser.String()})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	rec := doProbeRequest(t, router, http.MethodPost, "Bearer "+token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != realUser.String() {
		t.Errorf("expected handler to observe token's user id %q, got %q (impersonation via body succeeded)", realUser.String(), got)
	}
}

// TestRequireAuth_NoUserInContext_WhenUnauthenticated documents the
// UserIDFromContext contract itself (design decision 4f): calling it on a
// context RequireAuth never touched returns ok=false, not a zero UUID a
// caller might use without checking.
func TestRequireAuth_NoUserInContext_WhenUnauthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	userID, ok := UserIDFromContext(req.Context())
	if ok {
		t.Fatalf("expected ok=false for an untouched context, got ok=true, userID=%v", userID)
	}
	if userID != uuid.Nil {
		t.Errorf("expected zero UUID alongside ok=false, got %v", userID)
	}
}
