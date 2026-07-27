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

// maxResourceBodyBytes caps request bodies on the authenticated owned-
// resource endpoints (accounts, categories, transactions, summary). Larger
// than maxAuthBodyBytes because bulk-delete requests can legitimately carry
// up to 500 chunked ids (lib/chunk-items.ts): a 4 KiB limit would reject
// real client requests, not just abuse.
const maxResourceBodyBytes = 64 * 1024

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
// yields zero rows outside a transaction). The returned userID is the
// caller's own authenticated id, handed back so handlers do not need a
// second UserIDFromContext call to build query params.
//
// If the context carries no authenticated user, withUser writes 401
// UNAUTHORIZED itself and returns ok=false; this is unreachable behind
// RequireAuth today and is deliberate defense in depth (design decision
// 6.4) against ever binding app.user_id to the zero UUID. Callers must
// check ok before using err or userID.
func (s *ResourceServer) withUser(w http.ResponseWriter, r *http.Request, fn func(userID uuid.UUID, q *dbgen.Queries) error) (userID uuid.UUID, err error, ok bool) {
	userID, ok = UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorizedError(w)
		return uuid.UUID{}, nil, false
	}

	err = db.WithUserTx(r.Context(), s.pool, userID, func(q *dbgen.Queries) error {
		return fn(userID, q)
	})
	return userID, err, true
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
