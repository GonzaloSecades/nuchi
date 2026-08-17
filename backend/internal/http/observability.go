package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterOptions supplies platform concerns without coupling handlers to an
// exporter. Nil fields are replaced with quiet local/test defaults.
type RouterOptions struct {
	Logger    *slog.Logger
	Recorder  telemetry.Recorder
	Readiness *Readiness
}

func observeRequests(logger *slog.Logger, recorder telemetry.Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			requestID := middleware.GetReqID(r.Context())
			w.Header().Set("X-Request-ID", requestID)
			statusWriter := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(statusWriter, r)
			status := statusWriter.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := chi.RouteContext(r.Context()).RoutePattern()
			operation := operationID(r.Method, route)
			duration := time.Since(started)
			class := strconv.Itoa(status/100) + "xx"

			logger.LogAttrs(context.WithoutCancel(r.Context()), slog.LevelInfo, "http request completed",
				slog.String("request_id", requestID),
				slog.String("operation_id", operation),
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", status),
				slog.Int64("duration_ms", duration.Milliseconds()),
			)
			recorder.RecordRequest(context.WithoutCancel(r.Context()), telemetry.RequestResult{
				Operation: operation, Method: r.Method, Route: route, StatusClass: class, Duration: duration,
			})
		})
	}
}

func operationID(method, route string) string {
	if operation, ok := operationIDs[method+" "+route]; ok {
		return operation
	}
	return "unmatched"
}

var operationIDs = map[string]string{
	"GET /api/health":                       "getHealth",
	"GET /api/ready":                        "getReadiness",
	"POST /api/auth/register":               "registerUser",
	"POST /api/auth/login":                  "loginUser",
	"POST /api/auth/refresh":                "refreshSession",
	"POST /api/auth/logout":                 "logoutUser",
	"POST /api/auth/verify-email":           "verifyEmail",
	"POST /api/auth/password-reset/request": "requestPasswordReset",
	"POST /api/auth/password-reset/confirm": "confirmPasswordReset",
	"GET /api/accounts/":                    "listAccounts",
	"POST /api/accounts/":                   "createAccount",
	"POST /api/accounts/bulk-delete":        "bulkDeleteAccounts",
	"GET /api/accounts/{id}":                "getAccount",
	"PATCH /api/accounts/{id}":              "updateAccount",
	"DELETE /api/accounts/{id}":             "deleteAccount",
	"GET /api/categories/":                  "listCategories",
	"POST /api/categories/":                 "createCategory",
	"POST /api/categories/bulk-delete":      "bulkDeleteCategories",
	"GET /api/categories/{id}":              "getCategory",
	"PATCH /api/categories/{id}":            "updateCategory",
	"DELETE /api/categories/{id}":           "deleteCategory",
	"GET /api/transactions/":                "listTransactions",
	"POST /api/transactions/":               "createTransaction",
	"POST /api/transactions/bulk-create":    "bulkCreateTransactions",
	"POST /api/transactions/bulk-delete":    "bulkDeleteTransactions",
	"GET /api/transactions/{id}":            "getTransaction",
	"PATCH /api/transactions/{id}":          "updateTransaction",
	"DELETE /api/transactions/{id}":         "deleteTransaction",
	"GET /api/summary":                      "getSummary",
}
