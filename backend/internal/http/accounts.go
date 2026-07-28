package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"unicode/utf8"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	// Aliased: the bare package name `id` would read as shadowed inside the
	// by-id handlers, whose route parameter is also named id.
	idgen "github.com/GonzaloSecades/nuchi/backend/internal/id"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// errAccountNotFound is the sentinel db.WithUserTx's fn returns when a
// by-id account query (get/update/delete) matches no row. GetAccount,
// UpdateAccountName, and DeleteAccount all carry a WHERE id = $ AND
// user_id = $ predicate, so a foreign account and a genuinely missing one
// are indistinguishable to the caller — both roll the transaction back and
// map to 404 (design decision 6.8). Returning it from fn (rather than
// writing the response inside the closure) rolls the transaction back
// cleanly, per 6.10.
var errAccountNotFound = errors.New("httpapi: account not found")

// accountInputBody is the lenient decode target for AccountInput
// ({name}), used by both CreateAccount and UpdateAccount.
type accountInputBody struct {
	Name string `json:"name"`
}

// bulkDeleteBody is the lenient decode target for BulkDeleteRequest
// ({ids}).
type bulkDeleteBody struct {
	Ids []string `json:"ids"`
}

// ListAccounts implements GET /api/accounts. Returns the summary shape
// ({id, name}) for every account owned by the authenticated user, per the
// fixture's list/get-vs-create/update asymmetry (briefing §3).
func (s *ResourceServer) ListAccounts(w http.ResponseWriter, r *http.Request) {
	var rows []dbgen.Account
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		rows, err = q.ListAccounts(r.Context(), pgUserID(userID))
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logAccountsError(r, "list accounts", err)
		resp := openapi.ListAccounts500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch accounts")}
		_ = resp.VisitListAccountsResponse(w)
		return
	}

	// Nil slice from zero rows would serialize as JSON null; the contract
	// (and the frontend's .map) requires [] (design decision 6.13).
	summaries := make([]openapi.AccountSummary, 0, len(rows))
	for _, row := range rows {
		summaries = append(summaries, toAccountSummary(row))
	}
	resp := openapi.ListAccounts200JSONResponse{Data: summaries}
	_ = resp.VisitListAccountsResponse(w)
}

// GetAccount implements GET /api/accounts/{id}. Returns the summary shape;
// 404 for a missing or foreign id (6.8).
func (s *ResourceServer) GetAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var account dbgen.Account
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.GetAccount(r.Context(), dbgen.GetAccountParams{ID: id, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return errAccountNotFound
		}
		if err != nil {
			return err
		}
		account = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.GetAccount200JSONResponse{Data: toAccountSummary(account)}
		_ = resp.VisitGetAccountResponse(w)
	case errors.Is(err, errAccountNotFound):
		resp := openapi.GetAccount404JSONResponse{AccountNotFoundErrorJSONResponse: accountNotFoundError()}
		_ = resp.VisitGetAccountResponse(w)
	default:
		logAccountsError(r, "fetch account", err)
		resp := openapi.GetAccount500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch account")}
		_ = resp.VisitGetAccountResponse(w)
	}
}

// CreateAccount implements POST /api/accounts. Returns the full row
// (including plaidId: null, always, since legacy never sets it on create)
// on success; 409 on a duplicate (user_id, name).
func (s *ResourceServer) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var body accountInputBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.CreateAccount400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitCreateAccountResponse(w)
		return
	}

	if fields := validateAccountName(body.Name); len(fields) > 0 {
		resp := openapi.CreateAccount400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitCreateAccountResponse(w)
		return
	}

	// Legacy accounts.ts generates ids with @paralleldrive/cuid2's
	// createId(), and the finance tables keep text primary keys holding that
	// format. Converging finance ids on UUIDv7 is an observable change that
	// post-migration improvement 0002 defers until after parity, so this
	// migration keeps producing cuid2-shaped ids (internal/id).
	accountID := idgen.New()

	var account dbgen.Account
	txErr, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.CreateAccount(r.Context(), dbgen.CreateAccountParams{
			ID:     accountID,
			Name:   body.Name,
			UserID: pgUserID(userID),
			// PlaidID stays NULL: legacy accounts.ts never sets it on
			// create.
		})
		if err != nil {
			return err
		}
		account = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case txErr == nil:
		resp := openapi.CreateAccount200JSONResponse{Data: toAccount(account)}
		_ = resp.VisitCreateAccountResponse(w)
	case isUniqueViolation(txErr):
		resp := openapi.CreateAccount409JSONResponse{DuplicateAccountNameErrorJSONResponse: duplicateAccountNameError(txErr)}
		_ = resp.VisitCreateAccountResponse(w)
	default:
		logAccountsError(r, "create account", txErr)
		resp := openapi.CreateAccount500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to create account")}
		_ = resp.VisitCreateAccountResponse(w)
	}
}

// UpdateAccount implements PATCH /api/accounts/{id}. The single UPDATE
// statement (WHERE id = $ AND user_id = $) is simultaneously the ownership
// check and the race-free update: a foreign or missing row matches nothing
// -> pgx.ErrNoRows -> 404; a real name collision on an owned row raises
// 23505 -> 409. Do not reorder into an existence pre-check (design decision
// 6.8).
func (s *ResourceServer) UpdateAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var body accountInputBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.UpdateAccount400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitUpdateAccountResponse(w)
		return
	}

	if fields := validateAccountName(body.Name); len(fields) > 0 {
		resp := openapi.UpdateAccount400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitUpdateAccountResponse(w)
		return
	}

	var account dbgen.Account
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.UpdateAccountName(r.Context(), dbgen.UpdateAccountNameParams{
			Name:   body.Name,
			ID:     id,
			UserID: pgUserID(userID),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errAccountNotFound
		}
		if err != nil {
			return err
		}
		account = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.UpdateAccount200JSONResponse{Data: toAccount(account)}
		_ = resp.VisitUpdateAccountResponse(w)
	case errors.Is(err, errAccountNotFound):
		resp := openapi.UpdateAccount404JSONResponse{AccountNotFoundErrorJSONResponse: accountNotFoundError()}
		_ = resp.VisitUpdateAccountResponse(w)
	case isUniqueViolation(err):
		resp := openapi.UpdateAccount409JSONResponse{DuplicateAccountNameErrorJSONResponse: duplicateAccountNameError(err)}
		_ = resp.VisitUpdateAccountResponse(w)
	default:
		logAccountsError(r, "update account", err)
		resp := openapi.UpdateAccount500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to update account")}
		_ = resp.VisitUpdateAccountResponse(w)
	}
}

// DeleteAccount implements DELETE /api/accounts/{id}. 404 for a missing or
// foreign id (6.8); a successful delete cascades the account's transactions
// through the database foreign key (00002 migration), not application code.
func (s *ResourceServer) DeleteAccount(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var deletedID string
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.DeleteAccount(r.Context(), dbgen.DeleteAccountParams{ID: id, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return errAccountNotFound
		}
		if err != nil {
			return err
		}
		deletedID = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.DeleteAccount200JSONResponse{Data: openapi.DeletedResource{Id: deletedID}}
		_ = resp.VisitDeleteAccountResponse(w)
	case errors.Is(err, errAccountNotFound):
		resp := openapi.DeleteAccount404JSONResponse{AccountNotFoundErrorJSONResponse: accountNotFoundError()}
		_ = resp.VisitDeleteAccountResponse(w)
	default:
		logAccountsError(r, "delete account", err)
		resp := openapi.DeleteAccount500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to delete account")}
		_ = resp.VisitDeleteAccountResponse(w)
	}
}

// BulkDeleteAccounts implements POST /api/accounts/bulk-delete. Missing or
// unowned ids are silently ignored by the DELETE ... WHERE id = ANY($1) AND
// user_id = $2 query; this operation never 404s (fixtures line 825).
func (s *ResourceServer) BulkDeleteAccounts(w http.ResponseWriter, r *http.Request) {
	var body bulkDeleteBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.BulkDeleteAccounts400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitBulkDeleteAccountsResponse(w)
		return
	}

	if fields := validateBulkDeleteIds(body.Ids); len(fields) > 0 {
		resp := openapi.BulkDeleteAccounts400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitBulkDeleteAccountsResponse(w)
		return
	}

	var deletedIDs []string
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		deletedIDs, err = q.BulkDeleteAccounts(r.Context(), dbgen.BulkDeleteAccountsParams{
			Ids:    body.Ids,
			UserID: pgUserID(userID),
		})
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logAccountsError(r, "delete accounts", err)
		resp := openapi.BulkDeleteAccounts500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to delete accounts")}
		_ = resp.VisitBulkDeleteAccountsResponse(w)
		return
	}

	// Nil slice from zero deletions would serialize as JSON null (6.13).
	deleted := make([]openapi.DeletedResource, 0, len(deletedIDs))
	for _, id := range deletedIDs {
		deleted = append(deleted, openapi.DeletedResource{Id: id})
	}
	resp := openapi.BulkDeleteAccounts200JSONResponse{Data: deleted}
	_ = resp.VisitBulkDeleteAccountsResponse(w)
}

// --- validation -----------------------------------------------------------

// validateAccountName enforces the contract's AccountInput.name: minLength
// 1, counted in runes (not bytes) to match #41's convention. No trimming —
// trimming would change stored values relative to legacy.
func validateAccountName(name string) []apiFieldError {
	if utf8.RuneCountInString(name) < 1 {
		return []apiFieldError{{Path: "name", Message: "Name is required."}}
	}
	return nil
}

// validateBulkDeleteIds enforces the contract's BulkDeleteRequest.ids:
// minItems 1, and rejects any empty-string id (the contract declares no
// minLength on individual items, but an empty id can never match a real
// text primary key and legacy's Zod schema required min(1) per item).
func validateBulkDeleteIds(ids []string) []apiFieldError {
	if len(ids) == 0 {
		return []apiFieldError{{Path: "ids", Message: "At least one id is required."}}
	}
	for _, id := range ids {
		if id == "" {
			return []apiFieldError{{Path: "ids", Message: "Ids must not be empty."}}
		}
	}
	return nil
}

// --- projections ------------------------------------------------------

// toAccountSummary projects a dbgen.Account row into the contract's
// AccountSummary shape ({id, name}) used by list/get (fixture asymmetry,
// briefing §3).
func toAccountSummary(a dbgen.Account) openapi.AccountSummary {
	return openapi.AccountSummary{Id: a.ID, Name: a.Name}
}

// toAccount projects a dbgen.Account row into the contract's full Account
// shape used by create/update. plaidId is always serialized, never
// omitted (nil renders as JSON null, satisfying the Account schema's
// required plaidId field) since legacy never sets it.
func toAccount(a dbgen.Account) openapi.Account {
	var plaidID *string
	if a.PlaidID.Valid {
		plaidID = &a.PlaidID.String
	}
	return openapi.Account{
		Id:      a.ID,
		Name:    a.Name,
		PlaidId: plaidID,
		UserId:  uuid.UUID(a.UserID.Bytes).String(),
	}
}

// --- errors -------------------------------------------------------------

// accountNotFoundError mirrors the AccountNotFoundError response example.
func accountNotFoundError() openapi.AccountNotFoundErrorJSONResponse {
	return openapi.AccountNotFoundErrorJSONResponse{
		Error: openapi.ApiError{Code: "ACCOUNT_NOT_FOUND", Message: "Account not found."},
	}
}

// duplicateAccountNameError mirrors the DuplicateAccountNameError response
// example, carrying the violated constraint name under details.constraint
// (design decision 6.7 — the contract's ApiError is additionalProperties:
// false, so unlike the legacy Hono flat shape the constraint goes in
// details, not at the top level).
func duplicateAccountNameError(err error) openapi.DuplicateAccountNameErrorJSONResponse {
	details := map[string]any{"constraint": constraintName(err)}
	return openapi.DuplicateAccountNameErrorJSONResponse{
		Error: openapi.ApiError{
			Code:    "DUPLICATE_ACCOUNT_NAME",
			Message: "You already have an account with this name.",
			Details: &details,
		},
	}
}

// isUniqueViolation reports whether err is a Postgres 23505 unique
// violation (design decision 6.7).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// constraintName extracts the violated constraint's name from a Postgres
// unique-violation error. Empty string if err is not a *pgconn.PgError
// (should not happen given isUniqueViolation gates every call site, but
// this stays defensive rather than panicking).
func constraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// logAccountsError logs a database failure on an accounts operation.
// Follows auth.go's convention (see writeInternalError / sendAsync): the
// raw driver error is logged server-side only, never placed in the
// response body.
func logAccountsError(r *http.Request, operation string, err error) {
	slog.Default().ErrorContext(r.Context(), "accounts operation failed", "operation", operation, "error", err)
}
