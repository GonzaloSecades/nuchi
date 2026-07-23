package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
)

// userIDContextKey is the unexported context key type RequireAuth stores
// the authenticated user id under. Being a distinct, unexported type (not a
// plain string) is the standard context-key-collision guard: no other
// package can read or overwrite it by accident.
type userIDContextKey struct{}

// UserIDFromContext returns the authenticated user id RequireAuth placed on
// ctx, and true. It returns the zero UUID and false if ctx carries no
// authenticated user (RequireAuth never ran, or ran and rejected the
// request before calling next). Callers must check the boolean: a caller
// that used the zero UUID without checking ok would bind app.user_id to an
// all-zeros value that matches no real user's rows, silently returning
// empty results instead of failing loudly — the boolean makes "no user in
// context" unmistakable.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(uuid.UUID)
	return userID, ok
}

// RequireAuth returns middleware that validates a Bearer access token,
// signed with secret, on every request it wraps. On success it places the
// token's authenticated user id on the request context (read back with
// UserIDFromContext) and calls next. On failure it writes a 401 and never
// calls next.
//
// Identity comes ONLY from the Authorization header — never a body field,
// a query parameter, or a header like X-User-Id — so nothing else in the
// request can override which user a downstream handler believes it is
// acting as.
//
// A missing/malformed Authorization header, or any verification failure
// other than expiry (bad signature, wrong secret, disallowed algorithm,
// missing/invalid subject, garbage input), responds 401 UNAUTHORIZED. An
// expired-but-otherwise-valid token responds 401 ACCESS_TOKEN_EXPIRED
// instead, per the contract's documented carve-out (components.responses
// .UnauthorizedError in openapi/nuchi.openapi.json) — the client needs
// that distinction to know whether to call /api/auth/refresh instead of
// re-prompting for credentials. Every other rejection reason is
// deliberately collapsed so it leaks no information about token internals
// to a caller who should not have had a valid token to begin with.
func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorizedError(w)
				return
			}

			userID, err := auth.VerifyAccessToken(secret, token)
			if err != nil {
				if errors.Is(err, auth.ErrAccessTokenExpired) {
					writeAccessTokenExpiredError(w)
					return
				}
				writeUnauthorizedError(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extracts the token from a "Bearer <token>" Authorization
// header value. It rejects anything that is not exactly that shape: a
// missing header, a different scheme (Basic, Digest, ...), a bare "Bearer"
// with nothing after it, or a Bearer prefix followed by an empty token.
func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimPrefix(header, prefix)
	if token == "" {
		return "", false
	}
	return token, true
}

// writeAPIError writes status and apiErr as the contract's ApiErrorResponse
// envelope ({"error": {"code", "message", ...}}). It is the single
// low-level JSON writer for call sites with no generated Visit*Response
// method to call: writeInternalError's 500 (the contract declares no 500
// for the auth operations) and RequireAuth's 401s, which run ahead of
// whichever specific operation's generated response type would otherwise
// apply.
func writeAPIError(w http.ResponseWriter, status int, apiErr openapi.ApiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(openapi.ApiErrorResponse{Error: apiErr})
}

func writeUnauthorizedError(w http.ResponseWriter) {
	writeAPIError(w, http.StatusUnauthorized, openapi.ApiError{
		Code:    "UNAUTHORIZED",
		Message: "Authentication required.",
	})
}

func writeAccessTokenExpiredError(w http.ResponseWriter) {
	writeAPIError(w, http.StatusUnauthorized, openapi.ApiError{
		Code:    "ACCESS_TOKEN_EXPIRED",
		Message: "Access token expired.",
	})
}
