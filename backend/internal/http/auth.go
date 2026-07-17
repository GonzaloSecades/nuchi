package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"time"
	"unicode/utf8"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/config"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// refreshCookieName and refreshCookiePath must match the contract's
// refreshTokenCookie security scheme and the RefreshTokenSetCookie /
// RefreshTokenClearCookie header component examples exactly
// (openapi/nuchi.openapi.json).
const (
	refreshCookieName = "nuchi_refresh_token"
	refreshCookiePath = "/api/auth"
)

// maxAuthBodyBytes caps the request body on the unauthenticated auth
// endpoints. Register/login bodies are an email and a password — a few
// hundred bytes at most — so 4 KiB is generous while denying an anonymous
// client the ability to stream an arbitrarily large body into the JSON
// decoder (and from there into Argon2). Enforced with http.MaxBytesReader,
// which also covers chunked bodies that carry no Content-Length.
const maxAuthBodyBytes = 4 * 1024

// decodeAuthBody decodes an auth request body under maxAuthBodyBytes. The
// boolean result reports whether decoding succeeded; on failure the caller
// responds with its operation's 400 ValidationError (an oversized body or an
// unknown field is a malformed request, same as invalid JSON — the response
// shape stays within the contract).
//
// Unknown fields are rejected because the contract declares
// additionalProperties: false on RegisterRequest/LoginRequest — the contract
// is the oracle, and silent tolerance would let it drift. Future fields
// (profiles, households, roles) arrive as contract changes first, at which
// point they are known fields and pass untouched (#63).
func decodeAuthBody(w http.ResponseWriter, r *http.Request, dst *credentialsBody) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(dst) != nil {
		return false
	}
	// A valid body is exactly one JSON value: Decode stops after the first,
	// so require EOF to reject trailing values ({...}{"x":1}) the contract's
	// request-body schema would never accept.
	return dec.Decode(&struct{}{}) == io.EOF
}

// AuthServer implements the four in-scope generated openapi.ServerInterface
// auth methods (RegisterUser, LoginUser, RefreshSession, LogoutUser) with
// the exact signatures oapi-codegen generated, so wiring the remaining
// interface methods on other handlers later is additive. It intentionally
// does not implement the full ServerInterface: resource routes, email
// verification, and password reset are out of scope for #41 (see #42/#43).
type AuthServer struct {
	pool            *pgxpool.Pool
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	cookieSecure    bool
}

// authServerMethods documents that AuthServer's four methods below have the
// exact signatures openapi.ServerInterface declares for the same operation
// names. It is not used for dispatch (AuthServer does not implement the
// full ServerInterface), only as a compile-time signature check.
type authServerMethods interface {
	LoginUser(w http.ResponseWriter, r *http.Request)
	LogoutUser(w http.ResponseWriter, r *http.Request)
	RefreshSession(w http.ResponseWriter, r *http.Request)
	RegisterUser(w http.ResponseWriter, r *http.Request)
}

var _ authServerMethods = (*AuthServer)(nil)

// NewAuthServer builds an AuthServer backed by pool and the auth-related
// settings in cfg (JWTSecret, AccessTokenTTL, RefreshTokenTTL, CookieSecure).
func NewAuthServer(pool *pgxpool.Pool, cfg config.Config) *AuthServer {
	return &AuthServer{
		pool:            pool,
		jwtSecret:       cfg.JWTSecret,
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		cookieSecure:    cfg.CookieSecure,
	}
}

// credentialsBody is a lenient decode target for both RegisterRequest and
// LoginRequest bodies. Decoding into plain strings (rather than the
// generated types, which embed openapi_types.Email's own JSON-time
// validation) keeps error reporting under this handler's control so 400
// responses always carry the contract's ValidationError shape.
type credentialsBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// apiFieldError is one entry in a ValidationError's details.fields array,
// matching the shape shown in the contract's ValidationError example.
type apiFieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// RegisterUser implements POST /api/auth/register. It creates an unverified
// user with an Argon2id password hash. It intentionally does not create an
// email verification token or send email — that is #42's scope. Tests and
// operators mark a user verified via the existing MarkUserEmailVerified
// query until #42 lands.
func (s *AuthServer) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var body credentialsBody
	if !decodeAuthBody(w, r, &body) {
		writeRegisterValidationError(w)
		return
	}

	var fields []apiFieldError
	email, emailOK := normalizeEmail(body.Email)
	if !emailOK {
		fields = append(fields, apiFieldError{Path: "email", Message: "Invalid email address."})
	}
	// The contract's minLength counts characters; len(string) counts UTF-8
	// bytes and would let e.g. a two-emoji password (8 bytes, 2 characters)
	// through.
	if utf8.RuneCountInString(body.Password) < 8 {
		fields = append(fields, apiFieldError{Path: "password", Message: "Password must be at least 8 characters."})
	}
	if len(fields) > 0 {
		writeRegisterValidationError(w, fields...)
		return
	}

	passwordHash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeInternalError(w)
		return
	}

	userID, err := uuid.NewV7()
	if err != nil {
		writeInternalError(w)
		return
	}

	ctx := r.Context()
	q := dbgen.New(s.pool)

	if _, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		ID:           pgtype.UUID{Bytes: [16]byte(userID), Valid: true},
		Email:        email,
		PasswordHash: passwordHash,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			resp := openapi.RegisterUser409JSONResponse{ConflictErrorJSONResponse: conflictEmailError()}
			_ = resp.VisitRegisterUserResponse(w)
			return
		}
		writeInternalError(w)
		return
	}

	resp := openapi.RegisterUser201JSONResponse{
		Message: "Account created. Verify your email before logging in.",
	}
	_ = resp.VisitRegisterUserResponse(w)
}

// LoginUser implements POST /api/auth/login. On an unknown email it performs a
// dummy Argon2id verification so response timing does not distinguish
// "no such user" from "wrong password" (see auth.DummyVerify).
func (s *AuthServer) LoginUser(w http.ResponseWriter, r *http.Request) {
	var body credentialsBody
	if !decodeAuthBody(w, r, &body) {
		resp := openapi.LoginUser400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitLoginUserResponse(w)
		return
	}

	var fields []apiFieldError
	email, emailOK := normalizeEmail(body.Email)
	if !emailOK {
		fields = append(fields, apiFieldError{Path: "email", Message: "Invalid email address."})
	}
	if body.Password == "" {
		fields = append(fields, apiFieldError{Path: "password", Message: "Password is required."})
	}
	if len(fields) > 0 {
		resp := openapi.LoginUser400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitLoginUserResponse(w)
		return
	}

	ctx := r.Context()
	q := dbgen.New(s.pool)

	user, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Enumeration safety: spend the same Argon2id wall-clock time
			// on an unknown email as on a known one with a wrong password.
			auth.DummyVerify(body.Password)
			resp := openapi.LoginUser401JSONResponse{UnauthorizedErrorJSONResponse: invalidCredentialsError()}
			_ = resp.VisitLoginUserResponse(w)
			return
		}
		writeInternalError(w)
		return
	}

	ok, err := auth.VerifyPassword(body.Password, user.PasswordHash)
	if err != nil {
		writeInternalError(w)
		return
	}
	if !ok {
		resp := openapi.LoginUser401JSONResponse{UnauthorizedErrorJSONResponse: invalidCredentialsError()}
		_ = resp.VisitLoginUserResponse(w)
		return
	}

	if !user.EmailVerifiedAt.Valid {
		resp := openapi.LoginUser403JSONResponse{EmailNotVerifiedErrorJSONResponse: emailNotVerifiedError()}
		_ = resp.VisitLoginUserResponse(w)
		return
	}

	userID := uuid.UUID(user.ID.Bytes)
	accessToken, _, err := auth.IssueAccessToken(s.jwtSecret, userID, s.accessTokenTTL)
	if err != nil {
		writeInternalError(w)
		return
	}

	rawRefresh, refreshHash, err := auth.GenerateToken()
	if err != nil {
		writeInternalError(w)
		return
	}
	refreshID, err := uuid.NewV7()
	if err != nil {
		writeInternalError(w)
		return
	}
	if _, err := q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		ID:        pgtype.UUID{Bytes: [16]byte(refreshID), Valid: true},
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.refreshTokenTTL), Valid: true},
	}); err != nil {
		writeInternalError(w)
		return
	}

	setCookie := s.buildRefreshCookie(rawRefresh)
	resp := openapi.LoginUser200JSONResponse{
		Body: openapi.AuthSessionResponse{
			AccessToken: accessToken,
			ExpiresIn:   int(s.accessTokenTTL.Seconds()),
			TokenType:   openapi.Bearer,
			User:        toAuthUser(user),
		},
		Headers: openapi.LoginUser200ResponseHeaders{SetCookie: &setCookie},
	}
	_ = resp.VisitLoginUserResponse(w)
}

// RefreshSession implements POST /api/auth/refresh. It reads the refresh
// cookie, atomically consumes it (ConsumeRefreshToken — a single UPDATE
// that only one concurrent caller can win, see #40/#61), and on success
// creates a successor token and issues a new access token, all inside one
// transaction. A missing, unknown, expired, or already-consumed cookie
// always yields 401 and clears the cookie.
func (s *AuthServer) RefreshSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		s.respondRefreshInvalid(w)
		return
	}
	tokenHash := auth.HashToken(cookie.Value)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeInternalError(w)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	q := dbgen.New(tx)

	consumed, err := q.ConsumeRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.respondRefreshInvalid(w)
			return
		}
		writeInternalError(w)
		return
	}

	user, err := q.GetUserByID(ctx, consumed.UserID)
	if err != nil {
		// The user backing a still-valid refresh token should always
		// exist (refresh_tokens.user_id cascades on user delete); if the
		// row is genuinely gone, the session is invalid. Any other error
		// (transient DB failure, cancelled context) is a real 500 — mapping
		// it to 401 would silently log users out and hide the fault from
		// monitoring.
		if errors.Is(err, pgx.ErrNoRows) {
			s.respondRefreshInvalid(w)
			return
		}
		writeInternalError(w)
		return
	}

	newRaw, newHash, err := auth.GenerateToken()
	if err != nil {
		writeInternalError(w)
		return
	}
	newTokenID, err := uuid.NewV7()
	if err != nil {
		writeInternalError(w)
		return
	}
	if _, err := q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		ID:        pgtype.UUID{Bytes: [16]byte(newTokenID), Valid: true},
		UserID:    consumed.UserID,
		TokenHash: newHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.refreshTokenTTL), Valid: true},
	}); err != nil {
		writeInternalError(w)
		return
	}

	accessToken, _, err := auth.IssueAccessToken(s.jwtSecret, uuid.UUID(consumed.UserID.Bytes), s.accessTokenTTL)
	if err != nil {
		writeInternalError(w)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeInternalError(w)
		return
	}
	committed = true

	setCookie := s.buildRefreshCookie(newRaw)
	resp := openapi.RefreshSession200JSONResponse{
		Body: openapi.AuthSessionResponse{
			AccessToken: accessToken,
			ExpiresIn:   int(s.accessTokenTTL.Seconds()),
			TokenType:   openapi.Bearer,
			User:        toAuthUser(user),
		},
		Headers: openapi.RefreshSession200ResponseHeaders{SetCookie: &setCookie},
	}
	_ = resp.VisitRefreshSessionResponse(w)
}

func (s *AuthServer) respondRefreshInvalid(w http.ResponseWriter) {
	clearCookie := s.buildClearCookie()
	// RefreshSession401JSONResponse carries no Set-Cookie header field (the
	// contract only documents Set-Cookie on the 200 response), so the clear
	// cookie is set directly on the response writer before the generated
	// Visit method writes status/body; Visit only sets Content-Type and the
	// status code, so it does not disturb this header.
	w.Header().Set("Set-Cookie", clearCookie)
	resp := openapi.RefreshSession401JSONResponse{InvalidRefreshTokenErrorJSONResponse: invalidRefreshTokenError()}
	_ = resp.VisitRefreshSessionResponse(w)
}

// LogoutUser implements POST /api/auth/logout. Per the contract, a missing,
// unknown, expired, or already-revoked refresh cookie is a 401
// InvalidRefreshTokenError (not a silent no-op 200) — logout is not
// unconditionally idempotent at the HTTP layer, even though the underlying
// RevokeRefreshToken query is.
func (s *AuthServer) LogoutUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		resp := openapi.LogoutUser401JSONResponse{InvalidRefreshTokenErrorJSONResponse: invalidRefreshTokenError()}
		_ = resp.VisitLogoutUserResponse(w)
		return
	}
	tokenHash := auth.HashToken(cookie.Value)

	q := dbgen.New(s.pool)

	// Single atomic check-and-revoke: a separate validity read followed by a
	// revoke would let a concurrent refresh/logout revoke the token between
	// the two statements, turning what the contract defines as a 401 into a
	// false 200. ConsumeRefreshToken revokes only if still valid and reports
	// ErrNoRows otherwise — same one-winner semantics as the refresh path.
	if _, err := q.ConsumeRefreshToken(ctx, tokenHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			resp := openapi.LogoutUser401JSONResponse{InvalidRefreshTokenErrorJSONResponse: invalidRefreshTokenError()}
			_ = resp.VisitLogoutUserResponse(w)
			return
		}
		writeInternalError(w)
		return
	}

	clearCookie := s.buildClearCookie()
	resp := openapi.LogoutUser200JSONResponse{
		Body:    openapi.AuthMessageResponse{Message: "Logged out."},
		Headers: openapi.LogoutUser200ResponseHeaders{SetCookie: &clearCookie},
	}
	_ = resp.VisitLogoutUserResponse(w)
}

// buildRefreshCookie renders the Set-Cookie header value for a fresh
// refresh token: HttpOnly always, Secure from config, SameSite=Lax and
// Path=/api/auth per the contract's RefreshTokenSetCookie example, with a
// Max-Age matching the configured refresh token TTL.
func (s *AuthServer) buildRefreshCookie(rawToken string) string {
	c := &http.Cookie{
		Name:     refreshCookieName,
		Value:    rawToken,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.refreshTokenTTL.Seconds()),
	}
	return c.String()
}

// buildClearCookie renders the Set-Cookie header value that clears the
// refresh cookie, matching the contract's RefreshTokenClearCookie example
// (Max-Age=0).
func (s *AuthServer) buildClearCookie() string {
	c := &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	return c.String()
}

// normalizeEmail validates raw as an RFC 5322 mailbox and returns its
// normalized address form. Empty or unparsable input is rejected.
func normalizeEmail(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return "", false
	}
	return addr.Address, true
}

// toAuthUser projects a dbgen.User row into the contract's AuthUser shape.
func toAuthUser(u dbgen.User) openapi.AuthUser {
	return openapi.AuthUser{
		Id:            uuid.UUID(u.ID.Bytes),
		Email:         openapi_types.Email(u.Email),
		EmailVerified: u.EmailVerifiedAt.Valid,
	}
}

// --- Typed error constructors -----------------------------------------
//
// Each mirrors one of the ApiErrorResponse examples documented on the
// corresponding component in openapi/nuchi.openapi.json.

func validationError(fields ...apiFieldError) openapi.ValidationErrorJSONResponse {
	var details *map[string]interface{}
	if len(fields) > 0 {
		d := map[string]interface{}{"fields": fields}
		details = &d
	}
	return openapi.ValidationErrorJSONResponse{
		Error: openapi.ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Request validation failed.",
			Details: details,
		},
	}
}

func writeRegisterValidationError(w http.ResponseWriter, fields ...apiFieldError) {
	resp := openapi.RegisterUser400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
	_ = resp.VisitRegisterUserResponse(w)
}

// invalidCredentialsError is used for both "unknown email" and "wrong
// password" on login, deliberately identical so the response body cannot
// be used to enumerate registered emails.
func invalidCredentialsError() openapi.UnauthorizedErrorJSONResponse {
	return openapi.UnauthorizedErrorJSONResponse{
		Error: openapi.ApiError{Code: "UNAUTHORIZED", Message: "Invalid email or password."},
	}
}

func emailNotVerifiedError() openapi.EmailNotVerifiedErrorJSONResponse {
	return openapi.EmailNotVerifiedErrorJSONResponse{
		Error: openapi.ApiError{Code: "EMAIL_NOT_VERIFIED", Message: "Email address must be verified before login."},
	}
}

func conflictEmailError() openapi.ConflictErrorJSONResponse {
	return openapi.ConflictErrorJSONResponse{
		Error: openapi.ApiError{Code: "EMAIL_ALREADY_REGISTERED", Message: "Email is already registered."},
	}
}

func invalidRefreshTokenError() openapi.InvalidRefreshTokenErrorJSONResponse {
	return openapi.InvalidRefreshTokenErrorJSONResponse{
		Error: openapi.ApiError{Code: "INVALID_REFRESH_TOKEN", Message: "Refresh token is invalid or expired."},
	}
}

// writeInternalError handles failures with no dedicated response in the
// auth operations' documented response set (e.g. a database error talking
// to Postgres). It is a plain ApiErrorResponse write rather than a
// generated Visit*Response method because the contract does not declare a
// 500 for these four operations.
func writeInternalError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(openapi.ApiErrorResponse{
		Error: openapi.ApiError{Code: "INTERNAL_ERROR", Message: "Something went wrong. Please try again."},
	})
}
