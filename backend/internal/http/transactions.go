package httpapi

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	idgen "github.com/GonzaloSecades/nuchi/backend/internal/id"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/GonzaloSecades/nuchi/backend/internal/ratelimit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// maxSafeMilliunits is the largest magnitude the API accepts for an amount.
//
// The column is bigint (migration 00005), whose range is far wider, but the
// API deliberately stops at JavaScript's Number.MAX_SAFE_INTEGER so every
// amount it returns is exact in the browser. The contract states the same
// bound as minimum/maximum, but oapi-codegen emits no range validation from
// those keywords, so this check is the only thing enforcing it.
//
// The check also prevents a subtler bug: the generated Amount field is a Go
// int64 while some paths narrow to the column type, and an unchecked
// conversion would wrap silently — turning a large income into a negative
// expense with no error anywhere. Legacy never had this failure mode, because
// a JavaScript number reached PostgreSQL and was rejected there.
const maxSafeMilliunits int64 = 9007199254740991

// errTransactionNotFound is the sentinel for a by-id transaction query that
// matched no row: either it does not exist or it hangs off an account the
// caller does not own, which are deliberately indistinguishable.
var errTransactionNotFound = errors.New("httpapi: transaction not found")

// errAccountReferenceNotFound and errCategoryReferenceNotFound report that a
// referenced account/category is missing or unowned. Legacy validates the
// account first and returns "Account not found" before it ever looks at the
// category, so both the order and the distinct messages are observable.
var (
	errAccountReferenceNotFound  = errors.New("httpapi: referenced account not found")
	errCategoryReferenceNotFound = errors.New("httpapi: referenced category not found")
)

// transactionInputBody is the lenient decode target for TransactionInput.
//
// Amount is json.Number rather than int64 so a fractional or oversized value
// produces this handler's own field-level 400 instead of an opaque decoder
// error.
type transactionInputBody struct {
	Amount     json.Number `json:"amount"`
	Payee      string      `json:"payee"`
	Notes      *string     `json:"notes"`
	Date       string      `json:"date"`
	AccountID  string      `json:"accountId"`
	CategoryID *string     `json:"categoryId"`
	Currency   string      `json:"currency"`
}

// transactionInput is a validated body: the shape the handlers actually write.
type transactionInput struct {
	amount     int64
	payee      string
	notes      pgtype.Text
	date       pgtype.Timestamp
	accountID  string
	categoryID pgtype.Text
	currency   string
}

// ListTransactions implements GET /api/transactions.
//
// params carries accountId only. from/to are read straight from the query
// string instead: the generated FromDate/ToDate are openapi_types.Date, a
// struct that cannot represent a malformed value, so binding through it would
// discard exactly the inputs the contract requires three specific
// INVALID_QUERY messages for.
//
// All three parameters distinguish "omitted" from "present but empty". None of
// them sets allowEmptyValue, which defaults to false in OpenAPI 3.0.3, and each
// references a schema with a minimum length, so an empty value is malformed
// rather than a request for the default.
func (s *ResourceServer) ListTransactions(w http.ResponseWriter, r *http.Request, params openapi.ListTransactionsParams) {
	query := r.URL.Query()
	start, end, dateErr := parseDateRange(optionalQueryParam(query, "from"), optionalQueryParam(query, "to"), s.now())
	if dateErr != nil {
		resp := openapi.ListTransactions400JSONResponse{InvalidQueryErrorJSONResponse: invalidDateQueryError(dateErr)}
		_ = resp.VisitListTransactionsResponse(w)
		return
	}

	// accountId references ResourceId (minLength 1), so an explicitly empty
	// value is rejected for the same reason as an empty date rather than
	// silently meaning "no filter".
	accountID := pgtype.Text{}
	if params.AccountId != nil {
		if *params.AccountId == "" {
			resp := openapi.ListTransactions400JSONResponse{InvalidQueryErrorJSONResponse: invalidQueryError("accountId must not be empty.")}
			_ = resp.VisitListTransactionsResponse(w)
			return
		}
		accountID = pgtype.Text{String: *params.AccountId, Valid: true}
	}

	var rows []dbgen.ListTransactionsRow
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		rows, err = q.ListTransactions(r.Context(), dbgen.ListTransactionsParams{
			UserID:    pgUserID(userID),
			StartDate: pgtype.Timestamp{Time: start, Valid: true},
			EndDate:   pgtype.Timestamp{Time: end, Valid: true},
			AccountID: accountID,
		})
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logResourceError(r, "list transactions", err)
		resp := openapi.ListTransactions500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch transactions")}
		_ = resp.VisitListTransactionsResponse(w)
		return
	}

	items := make([]openapi.TransactionListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toTransactionListItem(row))
	}
	resp := openapi.ListTransactions200JSONResponse{Data: items}
	_ = resp.VisitListTransactionsResponse(w)
}

// GetTransaction implements GET /api/transactions/{id}. Ownership runs
// through the joined account; the response is the entity shape, without the
// joined account/category names the list carries.
func (s *ResourceServer) GetTransaction(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var transaction dbgen.Transaction
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.GetTransaction(r.Context(), dbgen.GetTransactionParams{ID: id, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return errTransactionNotFound
		}
		if err != nil {
			return err
		}
		transaction = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.GetTransaction200JSONResponse{Data: toTransaction(transaction)}
		_ = resp.VisitGetTransactionResponse(w)
	case errors.Is(err, errTransactionNotFound):
		resp := openapi.GetTransaction404JSONResponse{TransactionNotFoundErrorJSONResponse: transactionNotFoundError()}
		_ = resp.VisitGetTransactionResponse(w)
	default:
		logResourceError(r, "fetch transaction", err)
		resp := openapi.GetTransaction500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch transaction")}
		_ = resp.VisitGetTransactionResponse(w)
	}
}

// CreateTransaction implements POST /api/transactions. Reference ownership is
// validated inside the same transaction as the insert, so the checks and the
// write share one RLS-bound identity and either both land or neither does.
//
// That is not the same as excluding a concurrent delete of the referenced
// account or category. The validation reads are plain SELECTs and take no row
// locks, so under READ COMMITTED another transaction may delete a referenced
// row after the check and before the insert; the insert then fails on the
// foreign key. referenceErrorFromForeignKeyViolation maps that outcome back to
// the same 404 the up-front check would have produced, so the race surfaces as
// "account not found" rather than a generic 500.
func (s *ResourceServer) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var body transactionInputBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.CreateTransaction400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitCreateTransactionResponse(w)
		return
	}

	input, fields := validateTransactionInput(body)
	if len(fields) > 0 {
		resp := openapi.CreateTransaction400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitCreateTransactionResponse(w)
		return
	}

	userID, authed := UserIDFromContext(r.Context())
	if !authed {
		writeUnauthorizedError(w)
		return
	}
	// Checked before any work, matching legacy: a request that goes on to fail
	// reference validation still consumes an attempt.
	if allowed, retryAfter := s.allowMutation(userID, ratelimit.ActionCreate); !allowed {
		resp := openapi.CreateTransaction429JSONResponse{TransactionMutationRateLimitErrorJSONResponse: rateLimitedError(retryAfter)}
		_ = resp.VisitCreateTransactionResponse(w)
		return
	}

	transactionID := idgen.New()

	var created dbgen.Transaction
	txErr, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		if err := validateTransactionReferences(r, q, userID, input); err != nil {
			return err
		}
		row, err := q.CreateTransaction(r.Context(), dbgen.CreateTransactionParams{
			ID:         transactionID,
			Amount:     input.amount,
			Payee:      input.payee,
			Notes:      input.notes,
			Date:       input.date,
			AccountID:  input.accountID,
			CategoryID: input.categoryID,
			Currency:   input.currency,
		})
		if err != nil {
			// A referenced row may have been deleted since the check above.
			return referenceErrorFromForeignKeyViolation(err)
		}
		created = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case txErr == nil:
		resp := openapi.CreateTransaction200JSONResponse{Data: toTransaction(created)}
		_ = resp.VisitCreateTransactionResponse(w)
	case errors.Is(txErr, errAccountReferenceNotFound):
		resp := openapi.CreateTransaction404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: accountReferenceNotFoundError()}
		_ = resp.VisitCreateTransactionResponse(w)
	case errors.Is(txErr, errCategoryReferenceNotFound):
		resp := openapi.CreateTransaction404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: categoryReferenceNotFoundError()}
		_ = resp.VisitCreateTransactionResponse(w)
	default:
		logResourceError(r, "create transaction", txErr)
		resp := openapi.CreateTransaction500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to create transaction")}
		_ = resp.VisitCreateTransactionResponse(w)
	}
}

// UpdateTransaction implements PATCH /api/transactions/{id}.
//
// Despite the verb this is a full replacement, matching legacy: every field is
// required and an omitted one is not "leave unchanged". Error precedence is
// also legacy's — the referenced account is validated, then the category, and
// only then is the transaction itself matched, so a request naming both a
// missing transaction and a missing account reports the account.
func (s *ResourceServer) UpdateTransaction(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var body transactionInputBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.UpdateTransaction400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitUpdateTransactionResponse(w)
		return
	}

	input, fields := validateTransactionInput(body)
	if len(fields) > 0 {
		resp := openapi.UpdateTransaction400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitUpdateTransactionResponse(w)
		return
	}

	userID, authed := UserIDFromContext(r.Context())
	if !authed {
		writeUnauthorizedError(w)
		return
	}
	if allowed, retryAfter := s.allowMutation(userID, ratelimit.ActionUpdate); !allowed {
		resp := openapi.UpdateTransaction429JSONResponse{TransactionMutationRateLimitErrorJSONResponse: rateLimitedError(retryAfter)}
		_ = resp.VisitUpdateTransactionResponse(w)
		return
	}

	var updated dbgen.Transaction
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		if err := validateTransactionReferences(r, q, userID, input); err != nil {
			return err
		}
		row, err := q.UpdateTransaction(r.Context(), dbgen.UpdateTransactionParams{
			Amount:     input.amount,
			Payee:      input.payee,
			Notes:      input.notes,
			Date:       input.date,
			AccountID:  input.accountID,
			CategoryID: input.categoryID,
			Currency:   input.currency,
			ID:         id,
			UserID:     pgUserID(userID),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errTransactionNotFound
		}
		if err != nil {
			// A referenced row may have been deleted since the check above.
			return referenceErrorFromForeignKeyViolation(err)
		}
		updated = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.UpdateTransaction200JSONResponse{Data: toTransaction(updated)}
		_ = resp.VisitUpdateTransactionResponse(w)
	case errors.Is(err, errAccountReferenceNotFound):
		resp := openapi.UpdateTransaction404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: accountReferenceNotFoundError()}
		_ = resp.VisitUpdateTransactionResponse(w)
	case errors.Is(err, errCategoryReferenceNotFound):
		resp := openapi.UpdateTransaction404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: categoryReferenceNotFoundError()}
		_ = resp.VisitUpdateTransactionResponse(w)
	case errors.Is(err, errTransactionNotFound):
		resp := openapi.UpdateTransaction404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: openapi.TransactionReferenceNotFoundErrorJSONResponse(transactionNotFoundError())}
		_ = resp.VisitUpdateTransactionResponse(w)
	default:
		logResourceError(r, "update transaction", err)
		resp := openapi.UpdateTransaction500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to update transaction")}
		_ = resp.VisitUpdateTransactionResponse(w)
	}
}

// DeleteTransaction implements DELETE /api/transactions/{id}.
func (s *ResourceServer) DeleteTransaction(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	userID, authed := UserIDFromContext(r.Context())
	if !authed {
		writeUnauthorizedError(w)
		return
	}
	if allowed, retryAfter := s.allowMutation(userID, ratelimit.ActionDelete); !allowed {
		resp := openapi.DeleteTransaction429JSONResponse{TransactionMutationRateLimitErrorJSONResponse: rateLimitedError(retryAfter)}
		_ = resp.VisitDeleteTransactionResponse(w)
		return
	}

	var deletedID string
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.DeleteTransaction(r.Context(), dbgen.DeleteTransactionParams{ID: id, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return errTransactionNotFound
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
		resp := openapi.DeleteTransaction200JSONResponse{Data: openapi.DeletedResource{Id: deletedID}}
		_ = resp.VisitDeleteTransactionResponse(w)
	case errors.Is(err, errTransactionNotFound):
		resp := openapi.DeleteTransaction404JSONResponse{TransactionNotFoundErrorJSONResponse: transactionNotFoundError()}
		_ = resp.VisitDeleteTransactionResponse(w)
	default:
		logResourceError(r, "delete transaction", err)
		resp := openapi.DeleteTransaction500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to delete transaction")}
		_ = resp.VisitDeleteTransactionResponse(w)
	}
}

// validateTransactionReferences proves the authenticated user owns the
// referenced account, and the category when one is supplied, before any write.
//
// A foreign key proves the referenced row exists; it does not prove the caller
// owns it, and FK checks bypass RLS entirely. Without these lookups a caller
// could attach another tenant's category to their own transaction — which is
// exactly the hole migration 00004 hardened at the policy level. RLS remains
// the backstop; these checks are what produce the user-facing 404s.
func validateTransactionReferences(r *http.Request, q *dbgen.Queries, userID uuid.UUID, input transactionInput) error {
	if _, err := q.GetAccount(r.Context(), dbgen.GetAccountParams{ID: input.accountID, UserID: pgUserID(userID)}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errAccountReferenceNotFound
		}
		return err
	}

	if !input.categoryID.Valid {
		return nil
	}

	if _, err := q.GetCategory(r.Context(), dbgen.GetCategoryParams{ID: input.categoryID.String, UserID: pgUserID(userID)}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errCategoryReferenceNotFound
		}
		return err
	}
	return nil
}

// referenceErrorFromForeignKeyViolation converts a foreign-key violation on
// the transactions table into the matching reference sentinel, or returns err
// unchanged when it is not one.
//
// This closes the gap between validateTransactionReferences and the write.
// Those checks are unlocked SELECTs, so a concurrent delete of the referenced
// account or category can commit in between and the write then fails with
// SQLSTATE 23503. Without this mapping the caller would get a 500 for what is
// really "that account no longer exists" — the same condition the up-front
// check reports as 404. The constraint name says which reference broke.
//
// The alternative was SELECT ... FOR KEY SHARE on the reference rows, which
// would serialize the delete behind the write. That is a stronger guarantee,
// but it takes locks on every mutation to prevent an outcome this mapping
// already reports correctly, so it is not worth the contention here. Legacy
// has the same unlocked race and answers it with a 500, so mapping to 404 is
// strictly better than what it did.
func referenceErrorFromForeignKeyViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		return err
	}
	switch pgErr.ConstraintName {
	case "transactions_account_id_fkey":
		return errAccountReferenceNotFound
	case "transactions_category_id_fkey":
		return errCategoryReferenceNotFound
	default:
		return err
	}
}

// allowMutation consults the limiter and converts its decision into the
// handler's (allowed, retryAfterSeconds) pair.
func (s *ResourceServer) allowMutation(userID uuid.UUID, action ratelimit.Action) (bool, int) {
	result := s.limiter.Allow(userID.String(), action)
	if result.Allowed {
		return true, 0
	}
	// Legacy sends Math.max(1, ceil(ms/1000)); the header is observable, so
	// the rounding must match rather than merely approximate.
	seconds := int(math.Ceil(result.RetryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return false, seconds
}

// --- validation -----------------------------------------------------------

func validateTransactionInput(body transactionInputBody) (transactionInput, []apiFieldError) {
	var fields []apiFieldError
	var out transactionInput

	amount, amountErr := parseAmount(body.Amount)
	if amountErr != nil {
		fields = append(fields, *amountErr)
	} else {
		out.amount = amount
	}

	if body.Payee == "" {
		fields = append(fields, apiFieldError{Path: "payee", Message: "Payee is required."})
	}
	out.payee = body.Payee

	if body.Date == "" {
		fields = append(fields, apiFieldError{Path: "date", Message: "Date is required."})
	} else if parsed, ok := parseCalendarDate(body.Date); !ok {
		fields = append(fields, apiFieldError{Path: "date", Message: "Date must use yyyy-MM-dd."})
	} else {
		out.date = pgtype.Timestamp{Time: parsed, Valid: true}
	}

	if body.AccountID == "" {
		fields = append(fields, apiFieldError{Path: "accountId", Message: "Account is required."})
	}
	out.accountID = body.AccountID

	// Required and single-valued in the contract. Rejecting an omitted or
	// unknown currency loudly beats defaulting it, which would silently accept
	// a client that believes it is sending USD.
	if body.Currency != string(openapi.ARS) {
		fields = append(fields, apiFieldError{Path: "currency", Message: `Currency must be "ARS".`})
	}
	out.currency = body.Currency

	if body.Notes != nil {
		out.notes = pgtype.Text{String: *body.Notes, Valid: true}
	}
	// Omitted and explicit null are the same outcome: no category. Only nil
	// means "no category".
	//
	// A present-but-empty categoryId is neither: the field is nullable, but its
	// non-null branch is ResourceId with minLength 1, so "" is a malformed
	// request value and fails validation here — before any lookup — rather than
	// being reported as a reference that was not found. A 404 would claim a
	// syntactically valid id was missing, which is a different thing and leaves
	// the response outside the contract.
	if body.CategoryID != nil {
		if *body.CategoryID == "" {
			fields = append(fields, apiFieldError{Path: "categoryId", Message: "Category must not be empty."})
		} else {
			out.categoryID = pgtype.Text{String: *body.CategoryID, Valid: true}
		}
	}

	return out, fields
}

// parseAmount converts the raw JSON number to signed integer milliunits,
// rejecting anything the API cannot represent exactly.
func parseAmount(raw json.Number) (int64, *apiFieldError) {
	if raw == "" {
		return 0, &apiFieldError{Path: "amount", Message: "Amount is required."}
	}
	amount, err := raw.Int64()
	if err != nil {
		// Either fractional (money is never a float here) or beyond int64.
		return 0, &apiFieldError{Path: "amount", Message: "Amount must be a whole number of milliunits."}
	}
	if amount > maxSafeMilliunits || amount < -maxSafeMilliunits {
		return 0, &apiFieldError{Path: "amount", Message: "Amount is out of range."}
	}
	return amount, nil
}

// --- projections ----------------------------------------------------------

// toTransactionListItem projects a joined list row, which carries the current
// account and category names. Those names are joined data, not transaction
// columns: renaming a category must change what the next list returns without
// rewriting any transaction.
func toTransactionListItem(row dbgen.ListTransactionsRow) openapi.TransactionListItem {
	return openapi.TransactionListItem{
		Id:         row.ID,
		Date:       row.Date.Time,
		Category:   textPointer(row.Category),
		CategoryId: textPointer(row.CategoryID),
		Payee:      row.Payee,
		Amount:     row.Amount,
		Notes:      textPointer(row.Notes),
		Account:    row.Account,
		AccountId:  row.AccountID,
		Currency:   openapi.CurrencyCode(row.Currency),
	}
}

// toTransaction projects the entity shape used by get/create/update: the
// transaction's own columns only, with no joined names.
func toTransaction(t dbgen.Transaction) openapi.Transaction {
	return openapi.Transaction{
		Id:         t.ID,
		Amount:     t.Amount,
		Payee:      t.Payee,
		Notes:      textPointer(t.Notes),
		Date:       t.Date.Time,
		AccountId:  t.AccountID,
		CategoryId: textPointer(t.CategoryID),
		Currency:   openapi.CurrencyCode(t.Currency),
	}
}

// textPointer converts a nullable column into the pointer the generated types
// use, so SQL NULL serializes as JSON null rather than an empty string.
func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

// --- errors ---------------------------------------------------------------

func invalidDateQueryError(err error) openapi.InvalidQueryErrorJSONResponse {
	return invalidQueryError(err.Error())
}

// The contract's response component is InvalidQueryError rather than
// InvalidDateQueryError: this ticket added a non-date use of it (an empty
// accountId), so a name and description scoped to dates would have
// misdescribed an observable response in the generated documentation.

// invalidQueryError builds the contract's INVALID_QUERY body for any malformed
// query parameter, date or otherwise.
func invalidQueryError(message string) openapi.InvalidQueryErrorJSONResponse {
	return openapi.InvalidQueryErrorJSONResponse{
		Error: openapi.ApiError{Code: "INVALID_QUERY", Message: message},
	}
}

func transactionNotFoundError() openapi.TransactionNotFoundErrorJSONResponse {
	return openapi.TransactionNotFoundErrorJSONResponse{
		Error: openapi.ApiError{Code: "TRANSACTION_NOT_FOUND", Message: "Transaction not found."},
	}
}

func accountReferenceNotFoundError() openapi.TransactionReferenceNotFoundErrorJSONResponse {
	return openapi.TransactionReferenceNotFoundErrorJSONResponse{
		Error: openapi.ApiError{Code: "ACCOUNT_NOT_FOUND", Message: "Account not found."},
	}
}

func categoryReferenceNotFoundError() openapi.TransactionReferenceNotFoundErrorJSONResponse {
	return openapi.TransactionReferenceNotFoundErrorJSONResponse{
		Error: openapi.ApiError{Code: "CATEGORY_NOT_FOUND", Message: "Category not found."},
	}
}

func rateLimitedError(retryAfterSeconds int) openapi.TransactionMutationRateLimitErrorJSONResponse {
	return openapi.TransactionMutationRateLimitErrorJSONResponse{
		Body: openapi.ApiErrorResponse{
			Error: openapi.ApiError{
				Code:    "TRANSACTION_MUTATION_RATE_LIMITED",
				Message: "Too many transaction mutations. Please try again later.",
			},
		},
		Headers: openapi.TransactionMutationRateLimitErrorResponseHeaders{RetryAfter: &retryAfterSeconds},
	}
}

// now returns the server clock the date defaults are computed from, injectable
// so tests can pin "today" instead of depending on when they run.
func (s *ResourceServer) now() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock().UTC()
}
