package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

// authTestEnv wires a live pgxpool, a router mounting the real AuthServer,
// and a fixed JWT secret, following the TEST_DATABASE_URL skip convention
// used across internal/db's live tests. Every test using it registers its
// own users and cleans them up with t.Cleanup (cascades tokens via FK).
type authTestEnv struct {
	pool      *pgxpool.Pool
	router    http.Handler
	cfg       config.Config
	accessTTL time.Duration
	mailer    *mail.CapturingMailer
	// authServer is the same instance router dispatches to. Most tests only
	// need router; a few (the rollback-fault test) construct their own
	// AuthServer directly with a fault-injecting pool instead of using this
	// field.
	authServer *AuthServer
}

func newAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live auth HTTP test")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	appBaseURL, err := url.Parse("http://localhost:3000")
	if err != nil {
		t.Fatalf("parse test app base URL: %v", err)
	}
	cfg := config.Config{
		JWTSecret:       []byte("live-http-test-secret-at-least-32-bytes!!"),
		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
		CookieSecure:    false,

		AppBaseURL:           appBaseURL,
		VerificationTokenTTL: 48 * time.Hour,
		ResetTokenTTL:        30 * time.Minute,
	}

	mailer := mail.NewCapturingMailer()
	authServer := NewAuthServer(pool, cfg, mailer)
	router := NewRouter(authServer)

	return authTestEnv{pool: pool, router: router, cfg: cfg, accessTTL: cfg.AccessTokenTTL, mailer: mailer, authServer: authServer}
}

func (e authTestEnv) do(t *testing.T, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// refreshCookieFromResponse extracts the nuchi_refresh_token cookie set by
// rec, failing the test if it is absent.
func refreshCookieFromResponse(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			return c
		}
	}
	t.Fatalf("expected a %q cookie in response, headers: %v", refreshCookieName, rec.Header())
	return nil
}

func decodeAPIError(t *testing.T, rec *httptest.ResponseRecorder) openapi.ApiErrorResponse {
	t.Helper()
	var out openapi.ApiErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode error response: %v (body: %s)", err, rec.Body.String())
	}
	return out
}

func uniqueAuthTestEmail(label string) string {
	return fmt.Sprintf("auth-http-%s-%s@example.test", label, uuid.NewString())
}

// registerAndVerify registers a user through the real HTTP handler, then
// marks it verified directly via MarkUserEmailVerified (standing in for
// #42's email verification flow, which is out of scope here), and
// registers cleanup. Returns the email and password used.
func (e authTestEnv) registerAndVerify(t *testing.T, label, password string) (email string, cleanup func()) {
	t.Helper()

	email = uniqueAuthTestEmail(label)
	rec := e.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: password}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	ctx := context.Background()
	q := dbgen.New(e.pool)
	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail after register: unexpected error: %v", err)
	}
	if _, err := q.MarkUserEmailVerified(ctx, user.ID); err != nil {
		t.Fatalf("MarkUserEmailVerified: unexpected error: %v", err)
	}

	cleanup = func() {
		if _, err := e.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email); err != nil {
			t.Errorf("cleanup: failed to delete test user %q: %v", email, err)
		}
	}
	return email, cleanup
}

// --- register -----------------------------------------------------------

func TestAuthLive_Register_CreatesUnverifiedUser(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("register")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	rec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var body openapi.AuthMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if body.Message == "" {
		t.Error("expected a non-empty message")
	}

	user, err := dbgen.New(env.pool).GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("GetUserByEmail: unexpected error: %v", err)
	}
	if user.EmailVerifiedAt.Valid {
		t.Error("expected a freshly registered user to be unverified")
	}
	if user.PasswordHash == "" || user.PasswordHash == "correct-horse-battery" {
		t.Errorf("expected password to be hashed, got %q", user.PasswordHash)
	}
}

func TestAuthLive_Register_DuplicateEmailConflicts(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("dup")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	first := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d (body: %s)", first.Code, first.Body.String())
	}

	second := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "another-password"}, nil)
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d (body: %s)", second.Code, second.Body.String())
	}
	apiErr := decodeAPIError(t, second)
	if apiErr.Error.Code != "EMAIL_ALREADY_REGISTERED" {
		t.Errorf("expected code EMAIL_ALREADY_REGISTERED, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_Register_ValidationErrors(t *testing.T) {
	env := newAuthTestEnv(t)

	cases := []struct {
		name string
		body credentialsBody
	}{
		{"invalid email", credentialsBody{Email: "not-an-email", Password: "correct-horse-battery"}},
		{"short password", credentialsBody{Email: uniqueAuthTestEmail("short-pw"), Password: "short"}},
		// 8 UTF-8 bytes but only 2 characters: the contract's minLength
		// counts characters, so byte-length validation would wrongly accept
		// this.
		{"multibyte short password", credentialsBody{Email: uniqueAuthTestEmail("emoji-pw"), Password: "😀😀"}},
		// Far past maxAuthBodyBytes: MaxBytesReader must stop an anonymous
		// client from streaming an arbitrarily large body into the decoder
		// (or a giant accepted password into Argon2).
		{"oversized body", credentialsBody{Email: uniqueAuthTestEmail("oversized"), Password: strings.Repeat("a", 64*1024)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.do(t, http.MethodPost, "/api/auth/register", tc.body, nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
			}
			apiErr := decodeAPIError(t, rec)
			if apiErr.Error.Code != "VALIDATION_ERROR" {
				t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
			}
		})
	}
}

// TestAuthLive_Register_UnknownFieldRejected pins the contract's
// additionalProperties: false on RegisterRequest: a body carrying an
// undeclared field must 400, not be silently accepted with the field
// ignored (#63).
func TestAuthLive_Register_UnknownFieldRejected(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/register", map[string]any{
		"email":    uniqueAuthTestEmail("unknown-field"),
		"password": "correct-horse-battery",
		"isAdmin":  true,
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a body with an unknown field, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
	}
}

// --- login ---------------------------------------------------------------

func TestAuthLive_Login_UnverifiedUserForbidden(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("unverified")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	regRec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regRec.Code)
	}

	loginRec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if loginRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unverified login, got %d (body: %s)", loginRec.Code, loginRec.Body.String())
	}
	apiErr := decodeAPIError(t, loginRec)
	if apiErr.Error.Code != "EMAIL_NOT_VERIFIED" {
		t.Errorf("expected code EMAIL_NOT_VERIFIED, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_Login_VerifiedUserSucceeds(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "verified", password)
	t.Cleanup(cleanup)

	rec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: password}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var session openapi.AuthSessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&session); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if session.AccessToken == "" {
		t.Error("expected a non-empty access token")
	}
	if session.ExpiresIn != int(env.accessTTL.Seconds()) {
		t.Errorf("expected expiresIn %d, got %d", int(env.accessTTL.Seconds()), session.ExpiresIn)
	}
	if session.TokenType != openapi.Bearer {
		t.Errorf("expected tokenType Bearer, got %q", session.TokenType)
	}
	if string(session.User.Email) != email {
		t.Errorf("expected user email %q, got %q", email, session.User.Email)
	}
	if !session.User.EmailVerified {
		t.Error("expected user.emailVerified to be true")
	}

	userID, err := auth.VerifyAccessToken(env.cfg.JWTSecret, session.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %v", err)
	}
	if userID != session.User.Id {
		t.Errorf("expected access token sub %v to match user id %v", userID, session.User.Id)
	}

	cookie := refreshCookieFromResponse(t, rec)
	if !cookie.HttpOnly {
		t.Error("expected refresh cookie to be HttpOnly")
	}
	if cookie.Path != refreshCookiePath {
		t.Errorf("expected refresh cookie path %q, got %q", refreshCookiePath, cookie.Path)
	}
	if cookie.Value == "" {
		t.Error("expected a non-empty refresh cookie value")
	}
}

func TestAuthLive_Login_WrongPasswordUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)
	email, cleanup := env.registerAndVerify(t, "wrongpw", "correct-horse-battery")
	t.Cleanup(cleanup)

	rec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: "totally-wrong-password"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_Login_UnknownEmailUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{
		Email:    uniqueAuthTestEmail("never-registered"),
		Password: "whatever-password",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_Login_WrongPasswordAndUnknownEmailShareShape(t *testing.T) {
	env := newAuthTestEnv(t)
	email, cleanup := env.registerAndVerify(t, "shape", "correct-horse-battery")
	t.Cleanup(cleanup)

	wrongPassword := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: "nope"}, nil)
	unknownEmail := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: uniqueAuthTestEmail("nope"), Password: "nope"}, nil)

	if wrongPassword.Code != unknownEmail.Code {
		t.Fatalf("expected matching status codes, got %d vs %d", wrongPassword.Code, unknownEmail.Code)
	}
	wrongErr := decodeAPIError(t, wrongPassword)
	unknownErr := decodeAPIError(t, unknownEmail)
	if wrongErr.Error.Code != unknownErr.Error.Code || wrongErr.Error.Message != unknownErr.Error.Message {
		t.Errorf("expected identical error shape for enumeration safety, got %+v vs %+v", wrongErr, unknownErr)
	}
}

// TestAuthLive_Register_TrailingJSONRejected pins single-value bodies: a
// request carrying a second JSON value after the credentials object is
// malformed per the contract's request-body schema and must 400, not have
// its trailing value silently discarded (#63).
func TestAuthLive_Register_TrailingJSONRejected(t *testing.T) {
	env := newAuthTestEnv(t)

	// Built by hand: env.do round-trips bodies through json.Marshal, which
	// (correctly) refuses to produce this deliberately malformed payload.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(
		[]byte(`{"email":"trailing@example.test","password":"correct-horse-battery"}{"isAdmin":true}`),
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a body with trailing JSON, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
	}
}

// TestAuthLive_Login_UnknownFieldRejected mirrors the register case for
// LoginRequest's additionalProperties: false (#63).
func TestAuthLive_Login_UnknownFieldRejected(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/login", map[string]any{
		"email":    "someone@example.test",
		"password": "whatever-password",
		"remember": true,
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a body with an unknown field, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
	}
}

// --- refresh ---------------------------------------------------------------

func TestAuthLive_Refresh_RotatesAndInvalidatesOldCookie(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "refresh", password)
	t.Cleanup(cleanup)

	loginRec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: password}, nil)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", loginRec.Code)
	}
	oldCookie := refreshCookieFromResponse(t, loginRec)

	refreshRec := env.do(t, http.MethodPost, "/api/auth/refresh", nil, oldCookie)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d (body: %s)", refreshRec.Code, refreshRec.Body.String())
	}
	var session openapi.AuthSessionResponse
	if err := json.NewDecoder(refreshRec.Body).Decode(&session); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if session.AccessToken == "" {
		t.Error("expected a non-empty access token from refresh")
	}

	newCookie := refreshCookieFromResponse(t, refreshRec)
	if newCookie.Value == oldCookie.Value {
		t.Fatal("expected refresh to rotate to a new cookie value")
	}

	// Replaying the old (now-consumed) cookie must fail with 401 and clear
	// the cookie — the #61 atomic-consume race fix, observable at HTTP
	// level.
	replayRec := env.do(t, http.MethodPost, "/api/auth/refresh", nil, oldCookie)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("replaying old refresh cookie: expected 401, got %d (body: %s)", replayRec.Code, replayRec.Body.String())
	}
	apiErr := decodeAPIError(t, replayRec)
	if apiErr.Error.Code != "INVALID_REFRESH_TOKEN" {
		t.Errorf("expected code INVALID_REFRESH_TOKEN, got %q", apiErr.Error.Code)
	}
	clearedOnReplay := refreshCookieFromResponse(t, replayRec)
	if clearedOnReplay.MaxAge >= 0 {
		t.Errorf("expected the 401 response to clear the cookie (MaxAge<0), got MaxAge=%d", clearedOnReplay.MaxAge)
	}

	// The new cookie from the successful refresh still works.
	secondRefreshRec := env.do(t, http.MethodPost, "/api/auth/refresh", nil, newCookie)
	if secondRefreshRec.Code != http.StatusOK {
		t.Fatalf("refresh with rotated cookie: expected 200, got %d (body: %s)", secondRefreshRec.Code, secondRefreshRec.Body.String())
	}
}

func TestAuthLive_Refresh_MissingCookieUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/refresh", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "INVALID_REFRESH_TOKEN" {
		t.Errorf("expected code INVALID_REFRESH_TOKEN, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_Refresh_GarbageCookieUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/refresh", nil, &http.Cookie{Name: refreshCookieName, Value: "not-a-real-token"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// --- logout ------------------------------------------------------------

func TestAuthLive_Logout_ClearsCookieAndRevokesSession(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "logout", password)
	t.Cleanup(cleanup)

	loginRec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: password}, nil)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", loginRec.Code)
	}
	cookie := refreshCookieFromResponse(t, loginRec)

	logoutRec := env.do(t, http.MethodPost, "/api/auth/logout", nil, cookie)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d (body: %s)", logoutRec.Code, logoutRec.Body.String())
	}
	var msg openapi.AuthMessageResponse
	if err := json.NewDecoder(logoutRec.Body).Decode(&msg); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	if msg.Message == "" {
		t.Error("expected a non-empty logout message")
	}
	clearedCookie := refreshCookieFromResponse(t, logoutRec)
	if clearedCookie.MaxAge >= 0 {
		t.Errorf("expected logout to clear the cookie (MaxAge<0), got MaxAge=%d", clearedCookie.MaxAge)
	}

	// The revoked session can no longer refresh.
	refreshAfterLogout := env.do(t, http.MethodPost, "/api/auth/refresh", nil, cookie)
	if refreshAfterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: expected 401, got %d (body: %s)", refreshAfterLogout.Code, refreshAfterLogout.Body.String())
	}
}

func TestAuthLive_Logout_MissingCookieUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/logout", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "INVALID_REFRESH_TOKEN" {
		t.Errorf("expected code INVALID_REFRESH_TOKEN, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_Logout_AlreadyLoggedOutUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "double-logout", password)
	t.Cleanup(cleanup)

	loginRec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: password}, nil)
	cookie := refreshCookieFromResponse(t, loginRec)

	first := env.do(t, http.MethodPost, "/api/auth/logout", nil, cookie)
	if first.Code != http.StatusOK {
		t.Fatalf("first logout: expected 200, got %d", first.Code)
	}

	second := env.do(t, http.MethodPost, "/api/auth/logout", nil, cookie)
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("second logout with the same (revoked) cookie: expected 401, got %d (body: %s)", second.Code, second.Body.String())
	}
}

// --- cross-user isolation -------------------------------------------------

func TestAuthLive_Refresh_CookieIsPerUserAndDoesNotLeakIdentity(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	emailA, cleanupA := env.registerAndVerify(t, "isolation-a", password)
	t.Cleanup(cleanupA)
	emailB, cleanupB := env.registerAndVerify(t, "isolation-b", password)
	t.Cleanup(cleanupB)

	loginA := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: emailA, Password: password}, nil)
	loginB := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: emailB, Password: password}, nil)

	var sessionA, sessionB openapi.AuthSessionResponse
	if err := json.NewDecoder(loginA.Body).Decode(&sessionA); err != nil {
		t.Fatalf("decode session A: %v", err)
	}
	if err := json.NewDecoder(loginB.Body).Decode(&sessionB); err != nil {
		t.Fatalf("decode session B: %v", err)
	}
	if sessionA.User.Id == sessionB.User.Id {
		t.Fatal("expected two distinct users to get distinct ids")
	}

	cookieA := refreshCookieFromResponse(t, loginA)
	refreshA := env.do(t, http.MethodPost, "/api/auth/refresh", nil, cookieA)
	if refreshA.Code != http.StatusOK {
		t.Fatalf("refresh A: expected 200, got %d", refreshA.Code)
	}
	var refreshedA openapi.AuthSessionResponse
	if err := json.NewDecoder(refreshA.Body).Decode(&refreshedA); err != nil {
		t.Fatalf("decode refreshed session A: %v", err)
	}
	if refreshedA.User.Id != sessionA.User.Id {
		t.Errorf("expected refresh to preserve user A's id, got %v vs %v", refreshedA.User.Id, sessionA.User.Id)
	}
	if refreshedA.User.Id == sessionB.User.Id {
		t.Fatal("refreshing user A's cookie must never resolve to user B's identity")
	}
}
