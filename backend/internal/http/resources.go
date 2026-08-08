package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
	"unicode/utf8"

	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/GonzaloSecades/nuchi/backend/internal/ratelimit"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// decodeResourceBody decodes a request body on the authenticated owned-
// resource endpoints, with the same strict discipline as the auth endpoints
// (unknown fields rejected, exactly one JSON value required) but no byte
// cap.
//
// The cap is deliberately absent. Neither the legacy Hono validators nor
// the contract's AccountInput/BulkDeleteRequest declare maxLength, maxItems,
// or any byte-size limit, so a large-but-contract-valid body that Hono
// accepts and processes must not become a 400 here — the frontend's
// 500-item chunk size (lib/chunk-items.ts) is a client implementation
// detail, not an API constraint. Introducing a real limit is a contract
// change, scoped separately (post-migration improvement 0013).
//
// These endpoints sit behind RequireAuth, and the server's ReadTimeout
// (cmd/api/main.go) still bounds how long any single request may spend
// streaming a body.
func decodeResourceBody[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	return decodeJSONBody(w, r, dst, noBodyLimit)
}

// bodyDecodeOutcome distinguishes the two ways a size-limited decode can fail,
// because they map to different documented responses: a malformed body is 400
// VALIDATION_ERROR, while an oversized one is 413 REQUEST_BODY_TOO_LARGE.
// Collapsing them (as a bare bool would) answers the one condition the
// contract gives a dedicated status for with the wrong code.
type bodyDecodeOutcome int

const (
	bodyDecodeOK bodyDecodeOutcome = iota
	bodyDecodeMalformed
	bodyDecodeTooLarge
)

// decodeBulkBody decodes a bulk request body under a real byte limit.
//
// The limit is enforced twice, deliberately. r.ContentLength is checked first
// as a cheap rejection for a client that declares an oversized body, mirroring
// legacy's Content-Length guard. http.MaxBytesReader then enforces it against
// the actual stream, which is what closes the hole legacy has: a chunked
// request carries no Content-Length, so legacy's guard passes it through and
// streams an unbounded body into the decoder. See post-migration improvement
// 0016 for why that counts as a deliberate divergence.
//
// Note legacy's other carve-out -- a non-numeric Content-Length being ignored
// -- is unreachable here: net/http parses the header and rejects a malformed
// value with its own 400 before any handler runs.
func decodeBulkBody[T any](w http.ResponseWriter, r *http.Request, dst *T, limit int64) bodyDecodeOutcome {
	if r.ContentLength > limit {
		return bodyDecodeTooLarge
	}

	r.Body = http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return bodyDecodeTooLarge
		}
		return bodyDecodeMalformed
	}
	// Exactly one JSON value, same discipline as every other request body.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return bodyDecodeTooLarge
		}
		return bodyDecodeMalformed
	}
	return bodyDecodeOK
}

// ResourceServer implements the generated openapi.ServerInterface methods
// for the owned-resource routes (accounts in #44; categories, transactions,
// and summary follow in #45-#48 as additional methods on the same type).
// It holds a concrete *pgxpool.Pool, not AuthServer's dbHandle interface,
// because every handler routes through db.WithUserTx (internal/db/rls.go),
// which requires the pool directly to begin its own transaction and bind
// the RLS session user for that transaction's lifetime.
type ResourceServer struct {
	pool *pgxpool.Pool

	// limiter enforces the per-user, per-action transaction mutation limit.
	// Held on the server rather than in a package global so tests get an
	// isolated budget and cannot leak state into each other.
	limiter *ratelimit.Limiter

	// clock supplies "now" for the transaction list's date defaults. nil means
	// time.Now; tests pin it so a range assertion does not depend on when the
	// suite runs.
	clock func() time.Time
}

// NewResourceServer builds a ResourceServer backed by pool, with a fresh
// mutation rate limiter on the real clock.
func NewResourceServer(pool *pgxpool.Pool) *ResourceServer {
	return &ResourceServer{pool: pool, limiter: ratelimit.New(nil)}
}

// newResourceServerWithClock builds a ResourceServer whose limiter and date
// defaults both read from clock. Test-only: it exists so the rate-limit and
// date-range tests can advance time instead of sleeping.
func newResourceServerWithClock(pool *pgxpool.Pool, clock func() time.Time) *ResourceServer {
	return &ResourceServer{pool: pool, limiter: ratelimit.New(clock), clock: clock}
}

// resourceServerMethods documents that ResourceServer's handler methods
// have the exact signatures openapi.ServerInterface declares for the same
// operation names. It is not used for dispatch (chi routing stays
// hand-wired, same as auth), only as a compile-time signature check, and it
// grows as #46-#48 add transactions and summary.
type resourceServerMethods interface {
	ListAccounts(w http.ResponseWriter, r *http.Request)
	CreateAccount(w http.ResponseWriter, r *http.Request)
	BulkDeleteAccounts(w http.ResponseWriter, r *http.Request)
	GetAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	UpdateAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	DeleteAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)

	ListCategories(w http.ResponseWriter, r *http.Request)
	CreateCategory(w http.ResponseWriter, r *http.Request)
	BulkDeleteCategories(w http.ResponseWriter, r *http.Request)
	GetCategory(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	UpdateCategory(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	DeleteCategory(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)

	ListTransactions(w http.ResponseWriter, r *http.Request, params openapi.ListTransactionsParams)
	CreateTransaction(w http.ResponseWriter, r *http.Request)
	GetTransaction(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	UpdateTransaction(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	DeleteTransaction(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	BulkCreateTransactions(w http.ResponseWriter, r *http.Request)
	BulkDeleteTransactions(w http.ResponseWriter, r *http.Request)

	GetSummary(w http.ResponseWriter, r *http.Request, params openapi.GetSummaryParams)
}

var _ resourceServerMethods = (*ResourceServer)(nil)

// withResourceID adapts a handler with the generated (w, r, id) signature
// into a plain chi handler that reads the {id} URL parameter. Shared by
// every by-id resource route across #44-#48.
func withResourceID(fn func(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, chi.URLParam(r, "id"))
	}
}

// withListTransactionsParams adapts the generated (w, r, params) signature to
// a plain chi handler.
//
// It populates AccountId only. from/to are deliberately left nil and read
// raw from the query string by the handler: the generated FromDate/ToDate are
// openapi_types.Date — a struct that cannot hold a malformed value — so
// binding them here would discard precisely the inputs the contract requires
// distinct INVALID_QUERY messages for, and would answer with a generic binder
// error instead.
func withListTransactionsParams(fn func(w http.ResponseWriter, r *http.Request, params openapi.ListTransactionsParams)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params openapi.ListTransactionsParams
		// Presence, not emptiness: an explicitly empty accountId must reach the
		// handler as a present-but-invalid value so it can be rejected, rather
		// than being flattened into "omitted".
		params.AccountId = optionalQueryParam(r.URL.Query(), "accountId")
		fn(w, r, params)
	}
}

// optionalQueryParam returns a pointer to the raw value of key when the
// parameter is present, and nil when it is absent.
//
// url.Values.Get cannot express the difference: it returns "" for both an
// omitted key and `?key=`. Every optional query parameter in the contract
// references a schema with a minimum length and none sets allowEmptyValue
// (false by default in OpenAPI 3.0.3), so the distinction is the difference
// between defaulting and a 400.
func optionalQueryParam(query url.Values, key string) *string {
	if !query.Has(key) {
		return nil
	}
	value := query.Get(key)
	return &value
}

// accountIDFilter converts the optional accountId query parameter into the
// nullable value the transaction and summary queries take.
//
// nil means the parameter was omitted, so no filter applies. A present but
// empty value is rejected: accountId references ResourceId (minLength 1), so
// `?accountId=` is malformed for the same reason `?from=` is, rather than
// silently meaning "no filter". Shared by ListTransactions and GetSummary so
// the two cannot drift.
func accountIDFilter(accountID *string) (pgtype.Text, error) {
	if accountID == nil {
		return pgtype.Text{}, nil
	}
	if *accountID == "" {
		return pgtype.Text{}, dateRangeError{"accountId must not be empty."}
	}
	return pgtype.Text{String: *accountID, Valid: true}, nil
}

// withSummaryParams adapts the generated (w, r, params) signature to a plain
// chi handler. Like the transaction list, from/to are left for the handler to
// read raw — openapi_types.Date cannot carry a malformed value, and the three
// INVALID_QUERY messages exist precisely for those inputs.
func withSummaryParams(fn func(w http.ResponseWriter, r *http.Request, params openapi.GetSummaryParams)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params openapi.GetSummaryParams
		if accountID := optionalQueryParam(r.URL.Query(), "accountId"); accountID != nil {
			params.AccountId = accountID
		}
		fn(w, r, params)
	}
}

// withUser resolves the authenticated user id from r's context and, if
// present, runs fn inside db.WithUserTx bound to that user (internal/db/
// rls.go — the only sanctioned way to touch accounts/categories/
// transactions; see its doc comment on why a bare dbgen.New(pool) silently
// yields zero rows outside a transaction). The authenticated id is passed
// to fn, which is the only place a handler needs it — building query
// params — so it is deliberately not also returned.
//
// If the context carries no authenticated user, withUser writes 401
// UNAUTHORIZED itself and returns ok=false; this is unreachable behind
// RequireAuth today and is deliberate defense in depth (design decision
// 6.4) against ever binding app.user_id to the zero UUID. Callers must
// check ok before using err.
func (s *ResourceServer) withUser(w http.ResponseWriter, r *http.Request, fn func(userID uuid.UUID, q *dbgen.Queries) error) (err error, ok bool) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorizedError(w)
		return nil, false
	}

	err = db.WithUserTx(r.Context(), s.pool, userID, func(q *dbgen.Queries) error {
		return fn(userID, q)
	})
	return err, true
}

// pgUserID converts an authenticated user id into the pgtype.UUID shape the
// sqlc-generated query params expect.
func pgUserID(userID uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(userID), Valid: true}
}

// validateResourceName enforces the minLength 1 that every owned-resource
// name field declares (AccountInput.name, CategoryInput.name). Counted in
// runes rather than bytes, matching #41's convention, so a multi-byte
// single-character name is not rejected. No trimming — trimming would
// change stored values relative to legacy.
func validateResourceName(name string) []apiFieldError {
	if utf8.RuneCountInString(name) < 1 {
		return []apiFieldError{{Path: "name", Message: "Name is required."}}
	}
	return nil
}

// logResourceError logs a database failure on an owned-resource operation.
// Follows auth.go's convention (see writeInternalError / sendAsync): the
// raw driver error is logged server-side only, never placed in the
// response body.
func logResourceError(r *http.Request, operation string, err error) {
	slog.Default().ErrorContext(r.Context(), "resource operation failed", "operation", operation, "error", err)
}

// dbError builds the contract's DatabaseErrorJSONResponse embedding message,
// matching the legacy per-operation "DatabaseError - Failed to ..." wording
// (design decision 6.14). Never carries the underlying driver error.
func dbError(message string) openapi.DatabaseErrorJSONResponse {
	return openapi.DatabaseErrorJSONResponse{
		Error: openapi.ApiError{Code: "DB_ERROR", Message: message},
	}
}
