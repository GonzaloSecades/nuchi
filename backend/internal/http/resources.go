package httpapi

import (
	"net/http"

	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
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

// ResourceServer implements the generated openapi.ServerInterface methods
// for the owned-resource routes (accounts in #44; categories, transactions,
// and summary follow in #45-#48 as additional methods on the same type).
// It holds a concrete *pgxpool.Pool, not AuthServer's dbHandle interface,
// because every handler routes through db.WithUserTx (internal/db/rls.go),
// which requires the pool directly to begin its own transaction and bind
// the RLS session user for that transaction's lifetime.
type ResourceServer struct {
	pool *pgxpool.Pool
}

// NewResourceServer builds a ResourceServer backed by pool.
func NewResourceServer(pool *pgxpool.Pool) *ResourceServer {
	return &ResourceServer{pool: pool}
}

// accountsServerMethods documents that ResourceServer's account methods
// have the exact signatures openapi.ServerInterface declares for the same
// operation names. It is not used for dispatch (chi routing stays
// hand-wired, same as auth), only as a compile-time signature check so
// #45-#48 adding categories/transactions/summary methods stay additive.
type accountsServerMethods interface {
	ListAccounts(w http.ResponseWriter, r *http.Request)
	CreateAccount(w http.ResponseWriter, r *http.Request)
	BulkDeleteAccounts(w http.ResponseWriter, r *http.Request)
	GetAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	UpdateAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
	DeleteAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)
}

var _ accountsServerMethods = (*ResourceServer)(nil)

// withResourceID adapts a handler with the generated (w, r, id) signature
// into a plain chi handler that reads the {id} URL parameter. Shared by
// every by-id resource route across #44-#48.
func withResourceID(fn func(w http.ResponseWriter, r *http.Request, id openapi.ResourceId)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, chi.URLParam(r, "id"))
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

// dbError builds the contract's DatabaseErrorJSONResponse embedding message,
// matching the legacy per-operation "DatabaseError - Failed to ..." wording
// (design decision 6.14). Never carries the underlying driver error.
func dbError(message string) openapi.DatabaseErrorJSONResponse {
	return openapi.DatabaseErrorJSONResponse{
		Error: openapi.ApiError{Code: "DB_ERROR", Message: message},
	}
}
