package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/mail"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- test-only helpers ------------------------------------------------------

// doRequestNoT is the concurrency-safe twin of authTestEnv.do: it never
// calls *testing.T methods, so it is safe to invoke from goroutines other
// than the one running the test (t.Fatal et al. must only be called from
// the test's own goroutine). Callers collect responses and assert on them
// back on the main goroutine.
func doRequestNoT(router http.Handler, method, path string, body any, cookie *http.Cookie) (*httptest.ResponseRecorder, error) {
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
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
	router.ServeHTTP(rec, req)
	return rec, nil
}

// waitForCapturedTokens polls get (a CapturingMailer snapshot getter) until
// at least want captures addressed to to are present, or timeout elapses.
// Emails send asynchronously after commit (#42's send semantics), so tests
// asserting on a captured token must wait for the background goroutine
// rather than reading the mailer immediately after the HTTP response.
func waitForCapturedTokens(t *testing.T, get func() []mail.CapturedEmail, to string, want int, timeout time.Duration) []string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		var toks []string
		for _, e := range get() {
			if e.To == to {
				toks = append(toks, e.Token)
			}
		}
		if len(toks) >= want {
			return toks
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d captured sends to %q, got %d", want, to, len(toks))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// countCapturedTo counts captures addressed to to in emails.
func countCapturedTo(emails []mail.CapturedEmail, to string) int {
	n := 0
	for _, e := range emails {
		if e.To == to {
			n++
		}
	}
	return n
}

// insertExpiredVerificationToken inserts an already-expired email
// verification token row directly (bypassing the handler, which never
// issues one) so tests can exercise VerifyEmail's expired-token path.
func insertExpiredVerificationToken(t *testing.T, pool *pgxpool.Pool, userID pgtype.UUID) (rawToken string) {
	t.Helper()
	ctx := context.Background()
	q := dbgen.New(pool)

	raw, hash, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	tokenID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid.NewV7: %v", err)
	}
	if _, err := q.CreateEmailVerificationToken(ctx, dbgen.CreateEmailVerificationTokenParams{
		ID:        pgtype.UUID{Bytes: [16]byte(tokenID), Valid: true},
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
	}); err != nil {
		t.Fatalf("CreateEmailVerificationToken: %v", err)
	}
	return raw
}

// --- fault injection seam for the rollback test -----------------------------

// errInjectedFault is returned by faultingTx in place of running the
// matched statement.
var errInjectedFault = errors.New("email_flows_live_test: injected fault")

// errorRow is a pgx.Row whose Scan always fails with a fixed error,
// standing in for a query that failed to execute.
type errorRow struct{ err error }

func (r errorRow) Scan(dest ...any) error { return r.err }

// faultingTx wraps a real pgx.Tx and forces QueryRow to fail whenever the
// SQL text contains matchSQL, while every other statement (including the
// preceding Consume*Token call) runs for real. This is the seam the
// rollback-fault test uses to prove that a failure in the mutation step
// after a successful consume rolls the whole transaction back, leaving the
// token unconsumed.
type faultingTx struct {
	pgx.Tx
	matchSQL string
	err      error
}

func (f *faultingTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if containsSQL(sql, f.matchSQL) {
		return errorRow{f.err}
	}
	return f.Tx.QueryRow(ctx, sql, args...)
}

func containsSQL(sql, substr string) bool {
	return substr != "" && strings.Contains(sql, substr)
}

// faultingPool wraps a real *pgxpool.Pool and hands out faultingTx values
// from Begin, so it satisfies dbHandle exactly like the pool it wraps
// (Exec/Query/QueryRow are promoted directly from the embedded Pool).
type faultingPool struct {
	*pgxpool.Pool
	matchSQL string
	err      error
}

func (f *faultingPool) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := f.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &faultingTx{Tx: tx, matchSQL: f.matchSQL, err: f.err}, nil
}

var _ dbHandle = (*faultingPool)(nil)

// faultyAuthServer builds an AuthServer identical to env's, except its
// pool is wrapped to fail the first QueryRow whose SQL contains matchSQL.
// mailer is shared with env so captured sends from both routers are
// visible on env.mailer.
func (e authTestEnv) faultyAuthServer(matchSQL string) *AuthServer {
	return &AuthServer{
		pool:            &faultingPool{Pool: e.pool, matchSQL: matchSQL, err: errInjectedFault},
		jwtSecret:       e.cfg.JWTSecret,
		accessTokenTTL:  e.cfg.AccessTokenTTL,
		refreshTokenTTL: e.cfg.RefreshTokenTTL,
		cookieSecure:    e.cfg.CookieSecure,

		mailer:               e.mailer,
		verificationTokenTTL: e.cfg.VerificationTokenTTL,
		resetTokenTTL:        e.cfg.ResetTokenTTL,
	}
}

// --- verify-email ------------------------------------------------------------

func TestAuthLive_Register_SendsVerificationEmail_ThenVerifyEmailUnlocksLogin(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("verify-flow")
	password := "correct-horse-battery"
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	regRec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: password}, nil)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (body: %s)", regRec.Code, regRec.Body.String())
	}

	// Before verification, login is 403 (existing #41 behavior).
	preLogin := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: password}, nil)
	if preLogin.Code != http.StatusForbidden {
		t.Fatalf("login before verification: expected 403, got %d", preLogin.Code)
	}

	tokens := waitForCapturedTokens(t, env.mailer.VerificationSends, email, 1, 3*time.Second)
	token := tokens[0]

	verifyRec := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: token}, nil)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify-email: expected 200, got %d (body: %s)", verifyRec.Code, verifyRec.Body.String())
	}
	var msg openapi.AuthMessageResponse
	if err := json.NewDecoder(verifyRec.Body).Decode(&msg); err != nil {
		t.Fatalf("decode verify-email response: %v", err)
	}
	if msg.Message == "" {
		t.Error("expected a non-empty verify-email message")
	}

	postLogin := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: password}, nil)
	if postLogin.Code != http.StatusOK {
		t.Fatalf("login after verification: expected 200, got %d (body: %s)", postLogin.Code, postLogin.Body.String())
	}
}

func TestAuthLive_VerifyEmail_ReplayUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("verify-replay")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	regRec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regRec.Code)
	}
	tokens := waitForCapturedTokens(t, env.mailer.VerificationSends, email, 1, 3*time.Second)
	token := tokens[0]

	first := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: token}, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("first verify-email: expected 200, got %d (body: %s)", first.Code, first.Body.String())
	}

	second := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: token}, nil)
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("replayed verify-email: expected 401, got %d (body: %s)", second.Code, second.Body.String())
	}
	apiErr := decodeAPIError(t, second)
	if apiErr.Error.Code != "INVALID_TOKEN" {
		t.Errorf("expected code INVALID_TOKEN, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_VerifyEmail_ExpiredTokenUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("verify-expired")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	regRec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regRec.Code)
	}

	user, err := dbgen.New(env.pool).GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	expiredToken := insertExpiredVerificationToken(t, env.pool, user.ID)

	rec := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: expiredToken}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired token: expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "INVALID_TOKEN" {
		t.Errorf("expected code INVALID_TOKEN, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_VerifyEmail_GarbageTokenUnauthorized(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: "not-a-real-token"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("garbage token: expected 401, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "INVALID_TOKEN" {
		t.Errorf("expected code INVALID_TOKEN, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_VerifyEmail_EmptyTokenBadRequest(t *testing.T) {
	env := newAuthTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty token: expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	apiErr := decodeAPIError(t, rec)
	if apiErr.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
	}
}

// TestAuthLive_VerifyEmail_ConcurrentSameToken_ExactlyOneSucceeds proves
// ConsumeEmailVerificationToken's single-UPDATE one-time-consume semantics
// hold at the HTTP layer: of two simultaneous verify-email calls carrying
// the same token, exactly one succeeds.
func TestAuthLive_VerifyEmail_ConcurrentSameToken_ExactlyOneSucceeds(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("verify-concurrent")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	regRec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regRec.Code)
	}
	tokens := waitForCapturedTokens(t, env.mailer.VerificationSends, email, 1, 3*time.Second)
	token := tokens[0]

	const n = 2
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec, err := doRequestNoT(env.router, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: token}, nil)
			if err != nil {
				codes[i] = -1
				return
			}
			codes[i] = rec.Code
		}(i)
	}
	wg.Wait()

	var okCount, unauthorizedCount int
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			okCount++
		case http.StatusUnauthorized:
			unauthorizedCount++
		}
	}
	if okCount != 1 || unauthorizedCount != 1 {
		t.Fatalf("expected exactly one 200 and one 401 among concurrent verify-email calls, got codes %v", codes)
	}
}

// --- password reset request --------------------------------------------------

func TestAuthLive_PasswordResetRequest_UnknownEmail_SameShapeAsKnown(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	knownEmail, cleanup := env.registerAndVerify(t, "reset-known-shape", password)
	t.Cleanup(cleanup)

	unknownEmail := uniqueAuthTestEmail("reset-unknown-shape")

	knownRec := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: knownEmail}, nil)
	unknownRec := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: unknownEmail}, nil)

	if knownRec.Code != http.StatusOK || unknownRec.Code != http.StatusOK {
		t.Fatalf("expected both requests to 200, got known=%d unknown=%d", knownRec.Code, unknownRec.Code)
	}
	if knownRec.Body.String() != unknownRec.Body.String() {
		t.Errorf("expected byte-identical response bodies for enumeration safety, got known=%q unknown=%q",
			knownRec.Body.String(), unknownRec.Body.String())
	}

	// Only the known account should ever get a reset email.
	time.Sleep(200 * time.Millisecond)
	if got := countCapturedTo(env.mailer.ResetSends(), unknownEmail); got != 0 {
		t.Errorf("expected no reset email for an unknown address, got %d", got)
	}
}

// --- full reset lifecycle -----------------------------------------------------

func TestAuthLive_PasswordReset_FullLifecycle(t *testing.T) {
	env := newAuthTestEnv(t)
	oldPassword := "correct-horse-battery"
	newPassword := "new-correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "reset-lifecycle", oldPassword)
	t.Cleanup(cleanup)

	// A session obtained before the reset must be revoked afterwards.
	preLogin := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: oldPassword}, nil)
	if preLogin.Code != http.StatusOK {
		t.Fatalf("pre-reset login: expected 200, got %d", preLogin.Code)
	}
	preResetCookie := refreshCookieFromResponse(t, preLogin)

	reqRec := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
	if reqRec.Code != http.StatusOK {
		t.Fatalf("password-reset/request: expected 200, got %d (body: %s)", reqRec.Code, reqRec.Body.String())
	}
	tokens := waitForCapturedTokens(t, env.mailer.ResetSends, email, 1, 3*time.Second)
	token := tokens[0]

	confirmRec := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: token, Password: newPassword}, nil)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("password-reset/confirm: expected 200, got %d (body: %s)", confirmRec.Code, confirmRec.Body.String())
	}
	var msg openapi.AuthMessageResponse
	if err := json.NewDecoder(confirmRec.Body).Decode(&msg); err != nil {
		t.Fatalf("decode confirm response: %v", err)
	}
	if msg.Message == "" {
		t.Error("expected a non-empty confirm message")
	}

	oldPwLogin := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: oldPassword}, nil)
	if oldPwLogin.Code != http.StatusUnauthorized {
		t.Fatalf("login with old password after reset: expected 401, got %d", oldPwLogin.Code)
	}

	newPwLogin := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: newPassword}, nil)
	if newPwLogin.Code != http.StatusOK {
		t.Fatalf("login with new password after reset: expected 200, got %d (body: %s)", newPwLogin.Code, newPwLogin.Body.String())
	}

	refreshAfterReset := env.do(t, http.MethodPost, "/api/auth/refresh", nil, preResetCookie)
	if refreshAfterReset.Code != http.StatusUnauthorized {
		t.Fatalf("refresh with pre-reset cookie: expected 401 (session revoked), got %d (body: %s)", refreshAfterReset.Code, refreshAfterReset.Body.String())
	}

	replayRec := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: token, Password: "yet-another-password"}, nil)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("reset token replay: expected 401, got %d (body: %s)", replayRec.Code, replayRec.Body.String())
	}
	apiErr := decodeAPIError(t, replayRec)
	if apiErr.Error.Code != "INVALID_TOKEN" {
		t.Errorf("expected code INVALID_TOKEN, got %q", apiErr.Error.Code)
	}
}

func TestAuthLive_PasswordReset_TwoConsecutiveRequests_FirstTokenInvalidated(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "reset-two-consecutive", password)
	t.Cleanup(cleanup)

	first := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", first.Code)
	}
	firstTokens := waitForCapturedTokens(t, env.mailer.ResetSends, email, 1, 3*time.Second)
	firstToken := firstTokens[0]

	second := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
	if second.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", second.Code)
	}
	secondTokens := waitForCapturedTokens(t, env.mailer.ResetSends, email, 2, 3*time.Second)
	secondToken := secondTokens[1]

	if firstToken == secondToken {
		t.Fatal("expected two distinct reset tokens from two separate requests")
	}

	firstConfirm := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: firstToken, Password: "some-new-password-1"}, nil)
	if firstConfirm.Code != http.StatusUnauthorized {
		t.Fatalf("confirming with the invalidated first token: expected 401, got %d (body: %s)", firstConfirm.Code, firstConfirm.Body.String())
	}

	secondConfirm := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: secondToken, Password: "some-new-password-2"}, nil)
	if secondConfirm.Code != http.StatusOK {
		t.Fatalf("confirming with the current second token: expected 200, got %d (body: %s)", secondConfirm.Code, secondConfirm.Body.String())
	}
}

func TestAuthLive_PasswordReset_FourthRequestWithinHour_StillOKButOnlyThreeSent(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "reset-cap-sequential", password)
	t.Cleanup(cleanup)

	for i := range 4 {
		rec := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d (body: %s)", i+1, rec.Code, rec.Body.String())
		}
	}

	// Wait for the (at most 3) async sends to land, then hold for a grace
	// period to prove a 4th never arrives.
	got := 0
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got = countCapturedTo(env.mailer.ResetSends(), email)
		if got >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got != 3 {
		t.Fatalf("expected exactly 3 captured reset sends after 4 requests, got %d", got)
	}
	time.Sleep(300 * time.Millisecond)
	if got := countCapturedTo(env.mailer.ResetSends(), email); got != 3 {
		t.Errorf("expected reset sends to stay capped at 3 (4th request suppressed), got %d", got)
	}
}

func TestAuthLive_PasswordResetConfirm_WeakPasswordDoesNotBurnToken(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "reset-weak-pw", password)
	t.Cleanup(cleanup)

	reqRec := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
	if reqRec.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d", reqRec.Code)
	}
	tokens := waitForCapturedTokens(t, env.mailer.ResetSends, email, 1, 3*time.Second)
	token := tokens[0]

	weakRec := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: token, Password: "short12"}, nil)
	if weakRec.Code != http.StatusBadRequest {
		t.Fatalf("weak password confirm: expected 400, got %d (body: %s)", weakRec.Code, weakRec.Body.String())
	}
	apiErr := decodeAPIError(t, weakRec)
	if apiErr.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Error.Code)
	}

	// The same token must still work: validation runs before any DB work,
	// so the token was never touched by the rejected attempt.
	strongRec := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: token, Password: "a-strong-enough-password"}, nil)
	if strongRec.Code != http.StatusOK {
		t.Fatalf("confirm with a valid password using the same token: expected 200, got %d (body: %s)", strongRec.Code, strongRec.Body.String())
	}
}

// TestAuthLive_PasswordReset_ConcurrentIssuance_LockSerializesAndCapsAtThree
// fires >= 5 simultaneous reset requests for one user and asserts exactly
// one unused token remains and the per-hour count never exceeds 3. Without
// the LockUser (FOR UPDATE) row lock serializing issuance, concurrent
// requests could all pass the cap check or leave more than one live token
// — this test is able to fail if that lock regresses.
func TestAuthLive_PasswordReset_ConcurrentIssuance_LockSerializesAndCapsAtThree(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "reset-concurrent", password)
	t.Cleanup(cleanup)

	const n = 5
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec, err := doRequestNoT(env.router, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
			if err != nil {
				codes[i] = -1
				return
			}
			codes[i] = rec.Code
		}(i)
	}
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("concurrent request %d: expected 200, got %d", i, code)
		}
	}

	ctx := context.Background()
	q := dbgen.New(env.pool)
	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}

	// Give the last async sends a moment to land, then inspect DB state
	// directly (source of truth for issuance count/uniqueness).
	time.Sleep(300 * time.Millisecond)

	since := pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true}
	total, err := q.CountRecentPasswordResetTokens(ctx, dbgen.CountRecentPasswordResetTokensParams{UserID: user.ID, Since: since})
	if err != nil {
		t.Fatalf("CountRecentPasswordResetTokens: %v", err)
	}
	if total > maxResetTokensPerHour {
		t.Errorf("expected the per-hour cap to never be exceeded, got %d tokens", total)
	}

	var unused int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL`, user.ID).Scan(&unused); err != nil {
		t.Fatalf("count unused reset tokens: %v", err)
	}
	if unused != 1 {
		t.Errorf("expected exactly one unused reset token after concurrent issuance, got %d", unused)
	}
}

// --- rollback fault -----------------------------------------------------------

// TestAuthLive_ConfirmPasswordReset_RollbackFault_TokenRemainsUsable injects
// a failure into the UpdateUserPassword statement that runs immediately
// after a successful ConsumePasswordResetToken, inside the same
// transaction (transactional invariant #1, #42). The whole transaction must
// roll back, leaving the token unconsumed — usable on a retry against the
// real (non-faulty) handler.
func TestAuthLive_ConfirmPasswordReset_RollbackFault_TokenRemainsUsable(t *testing.T) {
	env := newAuthTestEnv(t)
	password := "correct-horse-battery"
	email, cleanup := env.registerAndVerify(t, "reset-rollback-fault", password)
	t.Cleanup(cleanup)

	reqRec := env.do(t, http.MethodPost, "/api/auth/password-reset/request", resetRequestBody{Email: email}, nil)
	if reqRec.Code != http.StatusOK {
		t.Fatalf("request: expected 200, got %d", reqRec.Code)
	}
	tokens := waitForCapturedTokens(t, env.mailer.ResetSends, email, 1, 3*time.Second)
	token := tokens[0]

	faultyRouter := NewRouter(env.faultyAuthServer("SET password_hash"))

	faultyRec, err := doRequestNoT(faultyRouter, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: token, Password: "attempted-new-password"}, nil)
	if err != nil {
		t.Fatalf("faulty confirm request: %v", err)
	}
	if faultyRec.Code == http.StatusOK {
		t.Fatalf("expected the injected fault to prevent a 200, got 200 (body: %s)", faultyRec.Body.String())
	}

	// The token must still be usable against the real handler: the shared
	// transaction rolled back, so the earlier consume never took effect.
	retryRec := env.do(t, http.MethodPost, "/api/auth/password-reset/confirm", resetConfirmBody{Token: token, Password: "a-real-new-password"}, nil)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry after rollback: expected 200 (token should remain usable), got %d (body: %s)", retryRec.Code, retryRec.Body.String())
	}

	loginRec := env.do(t, http.MethodPost, "/api/auth/login", credentialsBody{Email: email, Password: "a-real-new-password"}, nil)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login with the password set by the retried confirm: expected 200, got %d", loginRec.Code)
	}
}

// TestAuthLive_VerifyEmail_RollbackFault_TokenRemainsUsable mirrors the
// password-reset rollback-fault test for the other mutation named in #42's
// binding transactional invariants: MarkUserEmailVerified.
func TestAuthLive_VerifyEmail_RollbackFault_TokenRemainsUsable(t *testing.T) {
	env := newAuthTestEnv(t)
	email := uniqueAuthTestEmail("verify-rollback-fault")
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	regRec := env.do(t, http.MethodPost, "/api/auth/register", credentialsBody{Email: email, Password: "correct-horse-battery"}, nil)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", regRec.Code)
	}
	tokens := waitForCapturedTokens(t, env.mailer.VerificationSends, email, 1, 3*time.Second)
	token := tokens[0]

	faultyRouter := NewRouter(env.faultyAuthServer("SET email_verified_at"))

	faultyRec, err := doRequestNoT(faultyRouter, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: token}, nil)
	if err != nil {
		t.Fatalf("faulty verify request: %v", err)
	}
	if faultyRec.Code == http.StatusOK {
		t.Fatalf("expected the injected fault to prevent a 200, got 200 (body: %s)", faultyRec.Body.String())
	}

	retryRec := env.do(t, http.MethodPost, "/api/auth/verify-email", tokenBody{Token: token}, nil)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry after rollback: expected 200 (token should remain usable), got %d (body: %s)", retryRec.Code, retryRec.Body.String())
	}
}
