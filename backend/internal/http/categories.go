package httpapi

import (
	"errors"
	"net/http"

	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	idgen "github.com/GonzaloSecades/nuchi/backend/internal/id"
	"github.com/GonzaloSecades/nuchi/backend/internal/openapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// errCategoryNotFound is the sentinel withUser's fn returns when a by-id
// category query (get/update/delete) matches no row. Every such query
// carries a WHERE id = $ AND user_id = $ predicate, so a foreign category
// and a genuinely missing one are indistinguishable to the caller — both
// roll the transaction back and map to 404, mirroring accounts.
var errCategoryNotFound = errors.New("httpapi: category not found")

// categoryInputBody is the lenient decode target for CategoryInput
// ({name}), used by both CreateCategory and UpdateCategory.
type categoryInputBody struct {
	Name string `json:"name"`
}

// ListCategories implements GET /api/categories. Returns the summary shape
// ({id, name}), the same list/get-vs-create/update asymmetry accounts has.
func (s *ResourceServer) ListCategories(w http.ResponseWriter, r *http.Request) {
	var rows []dbgen.Category
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		rows, err = q.ListCategories(r.Context(), pgUserID(userID))
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logResourceError(r, "list categories", err)
		resp := openapi.ListCategories500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch categories")}
		_ = resp.VisitListCategoriesResponse(w)
		return
	}

	// Nil slice from zero rows would serialize as JSON null; the contract
	// (and the frontend's .map) requires [].
	summaries := make([]openapi.CategorySummary, 0, len(rows))
	for _, row := range rows {
		summaries = append(summaries, toCategorySummary(row))
	}
	resp := openapi.ListCategories200JSONResponse{Data: summaries}
	_ = resp.VisitListCategoriesResponse(w)
}

// GetCategory implements GET /api/categories/{id}. Returns the summary
// shape; 404 for a missing or foreign id.
func (s *ResourceServer) GetCategory(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var category dbgen.Category
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.GetCategory(r.Context(), dbgen.GetCategoryParams{ID: id, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return errCategoryNotFound
		}
		if err != nil {
			return err
		}
		category = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.GetCategory200JSONResponse{Data: toCategorySummary(category)}
		_ = resp.VisitGetCategoryResponse(w)
	case errors.Is(err, errCategoryNotFound):
		resp := openapi.GetCategory404JSONResponse{CategoryNotFoundErrorJSONResponse: categoryNotFoundError()}
		_ = resp.VisitGetCategoryResponse(w)
	default:
		logResourceError(r, "fetch category", err)
		resp := openapi.GetCategory500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to fetch category")}
		_ = resp.VisitGetCategoryResponse(w)
	}
}

// CreateCategory implements POST /api/categories. Returns the full row
// (plaidId always null, since legacy never sets it on create); 409 on a
// duplicate (user_id, name).
func (s *ResourceServer) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var body categoryInputBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.CreateCategory400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitCreateCategoryResponse(w)
		return
	}

	if fields := validateCategoryName(body.Name); len(fields) > 0 {
		resp := openapi.CreateCategory400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitCreateCategoryResponse(w)
		return
	}

	// cuid2-shaped id, matching legacy categories.ts's createId(); UUIDv7
	// convergence for finance tables is deferred by post-migration
	// improvement 0002.
	categoryID := idgen.New()

	var category dbgen.Category
	txErr, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.CreateCategory(r.Context(), dbgen.CreateCategoryParams{
			ID:     categoryID,
			Name:   body.Name,
			UserID: pgUserID(userID),
			// PlaidID stays NULL: legacy categories.ts never sets it on
			// create.
		})
		if err != nil {
			return err
		}
		category = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case txErr == nil:
		resp := openapi.CreateCategory200JSONResponse{Data: toCategory(category)}
		_ = resp.VisitCreateCategoryResponse(w)
	case isUniqueViolation(txErr):
		resp := openapi.CreateCategory409JSONResponse{DuplicateCategoryNameErrorJSONResponse: duplicateCategoryNameError(txErr)}
		_ = resp.VisitCreateCategoryResponse(w)
	default:
		logResourceError(r, "create category", txErr)
		resp := openapi.CreateCategory500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to create category")}
		_ = resp.VisitCreateCategoryResponse(w)
	}
}

// UpdateCategory implements PATCH /api/categories/{id}.
//
// Duplicate names return 409, NOT the 500 the legacy handler produces. This
// is the one deliberate behavior divergence in this ticket: legacy
// categories.ts wraps its UPDATE in a bare catch with no 23505 branch, so
// renaming a category onto an existing name surfaces as a server error
// (fixtures line 321). spec.md line 104 requires that mismatch to be decided
// explicitly in the contract rather than inherited accidentally, and the
// contract decided 409 — updateCategory declares it, with the same
// DuplicateCategoryNameError shape as create. Post-migration improvement
// 0005 tracks the history.
//
// Otherwise identical to UpdateAccount: the single UPDATE statement is both
// the ownership check and the race-free update, so a foreign or missing row
// yields pgx.ErrNoRows -> 404 with no existence pre-check.
func (s *ResourceServer) UpdateCategory(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var body categoryInputBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.UpdateCategory400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitUpdateCategoryResponse(w)
		return
	}

	if fields := validateCategoryName(body.Name); len(fields) > 0 {
		resp := openapi.UpdateCategory400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitUpdateCategoryResponse(w)
		return
	}

	var category dbgen.Category
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.UpdateCategoryName(r.Context(), dbgen.UpdateCategoryNameParams{
			Name:   body.Name,
			ID:     id,
			UserID: pgUserID(userID),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errCategoryNotFound
		}
		if err != nil {
			return err
		}
		category = row
		return nil
	})
	if !ok {
		return
	}

	switch {
	case err == nil:
		resp := openapi.UpdateCategory200JSONResponse{Data: toCategory(category)}
		_ = resp.VisitUpdateCategoryResponse(w)
	case errors.Is(err, errCategoryNotFound):
		resp := openapi.UpdateCategory404JSONResponse{CategoryNotFoundErrorJSONResponse: categoryNotFoundError()}
		_ = resp.VisitUpdateCategoryResponse(w)
	case isUniqueViolation(err):
		resp := openapi.UpdateCategory409JSONResponse{DuplicateCategoryNameErrorJSONResponse: duplicateCategoryNameError(err)}
		_ = resp.VisitUpdateCategoryResponse(w)
	default:
		logResourceError(r, "update category", err)
		resp := openapi.UpdateCategory500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to update category")}
		_ = resp.VisitUpdateCategoryResponse(w)
	}
}

// DeleteCategory implements DELETE /api/categories/{id}. 404 for a missing
// or foreign id. Deleting a category does NOT delete the transactions that
// reference it: transactions.category_id is ON DELETE SET NULL (migration
// 00002), so those transactions survive with a null category. This is the
// opposite of accounts, where deletion cascades the transactions away.
func (s *ResourceServer) DeleteCategory(w http.ResponseWriter, r *http.Request, id openapi.ResourceId) {
	var deletedID string
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		row, err := q.DeleteCategory(r.Context(), dbgen.DeleteCategoryParams{ID: id, UserID: pgUserID(userID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return errCategoryNotFound
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
		resp := openapi.DeleteCategory200JSONResponse{Data: openapi.DeletedResource{Id: deletedID}}
		_ = resp.VisitDeleteCategoryResponse(w)
	case errors.Is(err, errCategoryNotFound):
		resp := openapi.DeleteCategory404JSONResponse{CategoryNotFoundErrorJSONResponse: categoryNotFoundError()}
		_ = resp.VisitDeleteCategoryResponse(w)
	default:
		logResourceError(r, "delete category", err)
		resp := openapi.DeleteCategory500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to delete category")}
		_ = resp.VisitDeleteCategoryResponse(w)
	}
}

// BulkDeleteCategories implements POST /api/categories/bulk-delete. Missing
// or unowned ids are silently ignored by the query; this operation never
// 404s. Referencing transactions have their category_id set to null, same
// as the single delete.
func (s *ResourceServer) BulkDeleteCategories(w http.ResponseWriter, r *http.Request) {
	var body bulkDeleteBody
	if !decodeResourceBody(w, r, &body) {
		resp := openapi.BulkDeleteCategories400JSONResponse{ValidationErrorJSONResponse: validationError()}
		_ = resp.VisitBulkDeleteCategoriesResponse(w)
		return
	}

	if fields := validateBulkDeleteIds(body.Ids); len(fields) > 0 {
		resp := openapi.BulkDeleteCategories400JSONResponse{ValidationErrorJSONResponse: validationError(fields...)}
		_ = resp.VisitBulkDeleteCategoriesResponse(w)
		return
	}

	var deletedIDs []string
	err, ok := s.withUser(w, r, func(userID uuid.UUID, q *dbgen.Queries) error {
		var err error
		deletedIDs, err = q.BulkDeleteCategories(r.Context(), dbgen.BulkDeleteCategoriesParams{
			Ids:    body.Ids,
			UserID: pgUserID(userID),
		})
		return err
	})
	if !ok {
		return
	}
	if err != nil {
		logResourceError(r, "delete categories", err)
		resp := openapi.BulkDeleteCategories500JSONResponse{DatabaseErrorJSONResponse: dbError("DatabaseError - Failed to delete categories")}
		_ = resp.VisitBulkDeleteCategoriesResponse(w)
		return
	}

	deleted := make([]openapi.DeletedResource, 0, len(deletedIDs))
	for _, id := range deletedIDs {
		deleted = append(deleted, openapi.DeletedResource{Id: id})
	}
	resp := openapi.BulkDeleteCategories200JSONResponse{Data: deleted}
	_ = resp.VisitBulkDeleteCategoriesResponse(w)
}

// --- validation -----------------------------------------------------------

// validateCategoryName enforces the contract's CategoryInput.name:
// minLength 1, counted in runes rather than bytes. No trimming — trimming
// would change stored values relative to legacy.
func validateCategoryName(name string) []apiFieldError {
	return validateResourceName(name)
}

// --- projections ----------------------------------------------------------

// toCategorySummary projects a row into the contract's CategorySummary
// shape ({id, name}) used by list/get.
func toCategorySummary(c dbgen.Category) openapi.CategorySummary {
	return openapi.CategorySummary{Id: c.ID, Name: c.Name}
}

// toCategory projects a row into the contract's full Category shape used by
// create/update. plaidId is always serialized, never omitted (nil renders
// as JSON null, satisfying the schema's required plaidId).
func toCategory(c dbgen.Category) openapi.Category {
	var plaidID *string
	if c.PlaidID.Valid {
		plaidID = &c.PlaidID.String
	}
	return openapi.Category{
		Id:      c.ID,
		Name:    c.Name,
		PlaidId: plaidID,
		UserId:  uuid.UUID(c.UserID.Bytes).String(),
	}
}

// --- errors ---------------------------------------------------------------

// categoryNotFoundError mirrors the CategoryNotFoundError response example.
func categoryNotFoundError() openapi.CategoryNotFoundErrorJSONResponse {
	return openapi.CategoryNotFoundErrorJSONResponse{
		Error: openapi.ApiError{Code: "CATEGORY_NOT_FOUND", Message: "Category not found."},
	}
}

// duplicateCategoryNameError mirrors the DuplicateCategoryNameError
// response example, carrying the violated constraint under
// details.constraint (the contract's ApiError is additionalProperties:
// false, so unlike the legacy flat shape the constraint goes in details).
func duplicateCategoryNameError(err error) openapi.DuplicateCategoryNameErrorJSONResponse {
	details := map[string]any{"constraint": constraintName(err)}
	return openapi.DuplicateCategoryNameErrorJSONResponse{
		Error: openapi.ApiError{
			Code:    "DUPLICATE_CATEGORY_NAME",
			Message: "You already have a category with this name.",
			Details: &details,
		},
	}
}
