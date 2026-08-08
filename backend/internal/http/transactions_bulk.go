package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	idgen "github.com/GonzaloSecades/nuchi/backend/internal/id"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/GonzaloSecades/nuchi/backend/internal/ratelimit"
	"github.com/google/uuid"
)

// Bulk operation limits, from lib/transaction-limits.ts. The row counts are
// also declared in the contract as maxItems; the byte limits are the documented
// bulk body caps behind the 413.
//
// These are the only endpoints in the migration with a real body cap. The
// single-resource bodies are deliberately uncapped (improvement 0013) because
// neither legacy nor the contract bounds them — here the contract does.
const (
	maxBulkCreateTransactions = 500
	maxBulkDeleteTransactions = 500
	maxBulkCreateBodyBytes    = 1_000_000
	maxBulkDeleteBodyBytes    = 100_000
)

// bulkTransactionRow is the JSON shape jsonb_to_recordset destructures in
// BulkCreateTransactions. The field names are the SQL column names in that
// query's column list, not the API's camelCase — a mismatch here silently
// yields NULLs rather than an error, so the two must be read together.
type bulkTransactionRow struct {
	ID         string  `json:"id"`
	Amount     int64   `json:"amount"`
	Payee      string  `json:"payee"`
	Notes      *string `json:"notes"`
	Date       string  `json:"date"`
	AccountID  string  `json:"account_id"`
	CategoryID *string `json:"category_id"`
	Currency   string  `json:"currency"`
}

// BulkCreateTransactions implements POST /api/transactions/bulk-create.
//
// The request body is a bare JSON array of TransactionInput, not a wrapped
// object — TransactionBulkCreateRequest is `type: array` with minItems 1 and
// maxItems 500.
//
// Every row is validated before anything is written, and the insert is a single
// statement inside one transaction, so "no partial insert" is structural rather
// than something unwound on failure. The returned id set is verified against the
// requested one before that transaction commits, so a batch that did not land
// exactly as asked rolls back instead of returning a successful-looking short
// response.
func (s *ResourceServer) BulkCreateTransactions(w http.ResponseWriter, r *http.Request) {
	var body []transactionInputBody
	switch decodeBulkBody(w, r, &body, maxBulkCreateBodyBytes) {
	case bodyDecodeTooLarge:
		resp := openapi.BulkCreateTransactions413JSONResponse{RequestBodyTooLargeErrorJSONResponse: requestBodyTooLargeError()}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
		return
	case bodyDecodeMalformed:
		resp := openapi.BulkCreateTransactions400JSONResponse{BulkCreateValidationErrorJSONResponse: bulkCreateValidationError()}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
		return
	}

	inputs, fields := validateBulkTransactionInputs(body)
	if len(fields) > 0 {
		resp := openapi.BulkCreateTransactions400JSONResponse{BulkCreateValidationErrorJSONResponse: bulkCreateValidationError(fields...)}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
		return
	}

	userID, authed := UserIDFromContext(r.Context())
	if !authed {
		writeUnauthorizedError(w)
		return
	}
	if allowed, retryAfter := s.allowMutation(userID, ratelimit.ActionBulkCreate); !allowed {
		resp := openapi.BulkCreateTransactions429JSONResponse{TransactionMutationRateLimitErrorJSONResponse: rateLimitedError(retryAfter)}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
		return
	}

	rows := make([]bulkTransactionRow, 0, len(inputs))
	for _, input := range inputs {
		rows = append(rows, bulkTransactionRow{
			ID:         idgen.New(),
			Amount:     input.amount,
			Payee:      input.payee,
			Notes:      textPointer(input.notes),
			Date:       input.date.Time.Format(time.RFC3339),
			AccountID:  input.accountID,
			CategoryID: textPointer(input.categoryID),
			Currency:   input.currency,
		})
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		logResourceError(r, "create transactions", err)
		resp := openapi.BulkCreateTransactions500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to create transactions")}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
		return
	}

	var ordered []openapi.Transaction
	txErr, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		if err := validateBulkTransactionReferences(r, q, userID, inputs); err != nil {
			return err
		}
		out, err := q.BulkCreateTransactions(r.Context(), payload)
		if err != nil {
			// A referenced row may have been deleted since the checks above.
			return referenceErrorFromForeignKeyViolation(err)
		}
		// Checked before the transaction commits, not after: a mismatch here
		// rolls the batch back, whereas discovering it afterwards could only
		// produce a successful-looking response that silently dropped rows,
		// with the partial insert already durable.
		//
		// The check is on the id *set*, not the row count. Equal counts do not
		// prove the right rows came back — a duplicate, an unexpected id, or a
		// trigger that rewrote one would all keep the length intact while
		// breaking the request-to-response correspondence.
		ordered, err = matchCreatedToRequested(rows, out)
		if err != nil {
			return err
		}
		return nil
	})
	if !ok {
		return
	}

	switch {
	case txErr == nil:
		resp := openapi.BulkCreateTransactions200JSONResponse{Data: ordered}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
	case errors.Is(txErr, errAccountReferenceNotFound):
		resp := openapi.BulkCreateTransactions404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: accountReferenceNotFoundError()}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
	case errors.Is(txErr, errCategoryReferenceNotFound):
		resp := openapi.BulkCreateTransactions404JSONResponse{TransactionReferenceNotFoundErrorJSONResponse: categoryReferenceNotFoundError()}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
	default:
		logResourceError(r, "create transactions", txErr)
		resp := openapi.BulkCreateTransactions500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to create transactions")}
		_ = resp.VisitBulkCreateTransactionsResponse(w)
	}
}

// BulkDeleteTransactions implements POST /api/transactions/bulk-delete. Missing,
// duplicate, garbage and unowned ids are all silently ignored — the query scopes
// through the owned account, so this operation never 404s.
func (s *ResourceServer) BulkDeleteTransactions(w http.ResponseWriter, r *http.Request) {
	var body bulkDeleteBody
	switch decodeBulkBody(w, r, &body, maxBulkDeleteBodyBytes) {
	case bodyDecodeTooLarge:
		resp := openapi.BulkDeleteTransactions413JSONResponse{RequestBodyTooLargeErrorJSONResponse: requestBodyTooLargeError()}
		_ = resp.VisitBulkDeleteTransactionsResponse(w)
		return
	case bodyDecodeMalformed:
		resp := openapi.BulkDeleteTransactions400JSONResponse{BulkDeleteValidationErrorJSONResponse: bulkDeleteValidationError()}
		_ = resp.VisitBulkDeleteTransactionsResponse(w)
		return
	}

	if fields := validateTransactionBulkDeleteIDs(body.Ids); len(fields) > 0 {
		resp := openapi.BulkDeleteTransactions400JSONResponse{BulkDeleteValidationErrorJSONResponse: bulkDeleteValidationError(fields...)}
		_ = resp.VisitBulkDeleteTransactionsResponse(w)
		return
	}

	userID, authed := UserIDFromContext(r.Context())
	if !authed {
		writeUnauthorizedError(w)
		return
	}
	if allowed, retryAfter := s.allowMutation(userID, ratelimit.ActionBulkDelete); !allowed {
		resp := openapi.BulkDeleteTransactions429JSONResponse{TransactionMutationRateLimitErrorJSONResponse: rateLimitedError(retryAfter)}
		_ = resp.VisitBulkDeleteTransactionsResponse(w)
		return
	}

	var deletedIDs []string
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		deletedIDs, err = q.BulkDeleteTransactions(r.Context(), dbgen.BulkDeleteTransactionsParams{
			Ids:    body.Ids,
			UserID: pgUserID(userID),
		})
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logResourceError(r, "delete transactions", err)
		resp := openapi.BulkDeleteTransactions500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to delete transactions")}
		_ = resp.VisitBulkDeleteTransactionsResponse(w)
		return
	}

	deleted := make([]openapi.DeletedResource, 0, len(deletedIDs))
	for _, id := range deletedIDs {
		deleted = append(deleted, openapi.DeletedResource{Id: id})
	}
	resp := openapi.BulkDeleteTransactions200JSONResponse{Data: deleted}
	_ = resp.VisitBulkDeleteTransactionsResponse(w)
}

// matchCreatedToRequested pairs the rows the database returned back to the rows
// that were requested, in request order, and fails if the two sets do not match
// exactly.
//
// Ordering and verification are one function on purpose. PostgreSQL does not
// promise that RETURNING emits rows in the source SELECT's order, so the
// correspondence has to be rebuilt in Go — and rebuilding it is precisely when a
// mismatch becomes visible. Splitting them would allow ordering without
// verifying, which is the bug this replaced: a length-only check passed an
// equal-sized but wrong id set, and the ordering step then silently skipped the
// unmatched rows and returned a short 200.
//
// Since the handler generates the ids itself, "the right rows" is exactly "the
// ids I sent, each once". Anything else — a missing id, an unexpected one, a
// duplicate — is a broken invariant rather than something to paper over, and the
// caller runs this inside the transaction so returning an error rolls the batch
// back.
func matchCreatedToRequested(requested []bulkTransactionRow, created []dbgen.Transaction) ([]openapi.Transaction, error) {
	byID := make(map[string]dbgen.Transaction, len(created))
	for _, row := range created {
		if _, duplicate := byID[row.ID]; duplicate {
			return nil, fmt.Errorf("httpapi: bulk create returned id %q more than once", row.ID)
		}
		byID[row.ID] = row
	}

	ordered := make([]openapi.Transaction, 0, len(requested))
	for _, row := range requested {
		found, ok := byID[row.ID]
		if !ok {
			return nil, fmt.Errorf("httpapi: bulk create did not return requested id %q", row.ID)
		}
		delete(byID, row.ID)
		ordered = append(ordered, toTransaction(found))
	}

	// Whatever is left was never asked for.
	for id := range byID {
		return nil, fmt.Errorf("httpapi: bulk create returned unrequested id %q", id)
	}

	return ordered, nil
}

// validateBulkTransactionReferences proves the caller owns every distinct
// account, and every distinct non-null category, referenced by the batch.
//
// Set-based on purpose: one query per resource type for the whole batch rather
// than a lookup per row, which at the 500-row maximum would be 1,000 round
// trips inside a single transaction. Accounts are checked before categories, so
// a batch with both a foreign account and a foreign category reports the
// account — the same precedence as the single-row path.
func validateBulkTransactionReferences(r *http.Request, q *dbgen.Queries, userID uuid.UUID, inputs []transactionInput) error {
	accountIDs := make([]string, 0, len(inputs))
	categoryIDs := make([]string, 0, len(inputs))
	seenAccount := make(map[string]struct{}, len(inputs))
	seenCategory := make(map[string]struct{}, len(inputs))

	for _, input := range inputs {
		if _, dup := seenAccount[input.accountID]; !dup {
			seenAccount[input.accountID] = struct{}{}
			accountIDs = append(accountIDs, input.accountID)
		}
		if !input.categoryID.Valid {
			continue
		}
		if _, dup := seenCategory[input.categoryID.String]; !dup {
			seenCategory[input.categoryID.String] = struct{}{}
			categoryIDs = append(categoryIDs, input.categoryID.String)
		}
	}

	ownedAccounts, err := q.ListOwnedAccountIDs(r.Context(), dbgen.ListOwnedAccountIDsParams{
		UserID: pgUserID(userID),
		Ids:    accountIDs,
	})
	if err != nil {
		return err
	}
	if len(ownedAccounts) != len(accountIDs) {
		return errAccountReferenceNotFound
	}

	if len(categoryIDs) == 0 {
		return nil
	}

	ownedCategories, err := q.ListOwnedCategoryIDs(r.Context(), dbgen.ListOwnedCategoryIDsParams{
		UserID: pgUserID(userID),
		Ids:    categoryIDs,
	})
	if err != nil {
		return err
	}
	if len(ownedCategories) != len(categoryIDs) {
		return errCategoryReferenceNotFound
	}
	return nil
}

// --- validation -----------------------------------------------------------

// validateBulkTransactionInputs validates the whole batch and returns every
// failure at once, so a client fixing a 500-row CSV import sees all the bad
// rows rather than one per round trip.
//
// Field paths are indexed — `[3].amount` — with `$` for failures about the
// array itself. The contract's ValidationError carries details.fields[].path
// and says nothing about indices, so this is the deliberate shape choice the
// ticket asks to be explicit about.
func validateBulkTransactionInputs(body []transactionInputBody) ([]transactionInput, []apiFieldError) {
	if len(body) == 0 {
		return nil, []apiFieldError{{Path: "$", Message: "At least one transaction is required."}}
	}
	if len(body) > maxBulkCreateTransactions {
		return nil, []apiFieldError{{
			Path:    "$",
			Message: fmt.Sprintf("At most %d transactions can be created at once.", maxBulkCreateTransactions),
		}}
	}

	inputs := make([]transactionInput, 0, len(body))
	var fields []apiFieldError
	for i, row := range body {
		input, rowFields := validateTransactionInput(row)
		for _, field := range rowFields {
			fields = append(fields, apiFieldError{
				Path:    fmt.Sprintf("[%d].%s", i, field.Path),
				Message: field.Message,
			})
		}
		inputs = append(inputs, input)
	}
	if len(fields) > 0 {
		return nil, fields
	}
	return inputs, nil
}

// validateTransactionBulkDeleteIDs enforces TransactionBulkDeleteRequest.ids:
// minItems 1, maxItems 500, no empty ids.
//
// Deliberately separate from validateBulkDeleteIds, which accounts and
// categories use: their BulkDeleteRequest declares no maxItems, so sharing one
// validator would silently impose a 500-id cap on two resources that
// intentionally do not have one.
func validateTransactionBulkDeleteIDs(ids []string) []apiFieldError {
	if len(ids) == 0 {
		return []apiFieldError{{Path: "ids", Message: "At least one id is required."}}
	}
	if len(ids) > maxBulkDeleteTransactions {
		return []apiFieldError{{
			Path:    "ids",
			Message: fmt.Sprintf("At most %d ids can be deleted at once.", maxBulkDeleteTransactions),
		}}
	}
	for i, id := range ids {
		if id == "" {
			return []apiFieldError{{Path: fmt.Sprintf("ids[%d]", i), Message: "Ids must not be empty."}}
		}
	}
	return nil
}

// --- errors ---------------------------------------------------------------

// bulkCreateValidationError and bulkDeleteValidationError build the two
// operation-specific bulk validation bodies.
//
// The envelope is the same as the single-resource ValidationError in every case;
// what differs is what the contract documents, and the two operations genuinely
// differ there. bulk-create reports `[i].field` per failing row and `$` for
// whole-array problems, accumulating every failure. bulk-delete reports `ids`
// when the array itself is unusable and `ids[i]` for the first empty id, and
// stops at the first. A single shared component described one of those and
// advertised it for both, so generated docs for bulk-delete promised a shape
// this code never produces.
//
// Called with no fields for an unparseable body, where there is nothing to
// index and `details` is omitted — also now documented on both.
func bulkCreateValidationError(fields ...apiFieldError) openapi.BulkCreateValidationErrorJSONResponse {
	return openapi.BulkCreateValidationErrorJSONResponse(validationError(fields...))
}

func bulkDeleteValidationError(fields ...apiFieldError) openapi.BulkDeleteValidationErrorJSONResponse {
	return openapi.BulkDeleteValidationErrorJSONResponse(validationError(fields...))
}

func requestBodyTooLargeError() openapi.RequestBodyTooLargeErrorJSONResponse {
	return openapi.RequestBodyTooLargeErrorJSONResponse{
		Error: openapi.ApiError{Code: "REQUEST_BODY_TOO_LARGE", Message: "Request body is too large."},
	}
}
