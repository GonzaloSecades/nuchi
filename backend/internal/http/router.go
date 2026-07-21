package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires the API routes owned by this service. authServer is
// optional (nil skips mounting the /api/auth/* routes) so callers that only
// need health/scaffold behavior — e.g. tests unrelated to auth — are not
// forced to construct a database pool. Resource routes (accounts,
// categories, transactions, summary) and auth middleware/RLS binding are
// left to later issues (#43+).
func NewRouter(authServer *AuthServer) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/api/health", healthHandler)

	if authServer != nil {
		router.Post("/api/auth/register", authServer.RegisterUser)
		router.Post("/api/auth/login", authServer.LoginUser)
		router.Post("/api/auth/refresh", authServer.RefreshSession)
		router.Post("/api/auth/logout", authServer.LogoutUser)
		router.Post("/api/auth/verify-email", authServer.VerifyEmail)
		router.Post("/api/auth/password-reset/request", authServer.RequestPasswordReset)
		router.Post("/api/auth/password-reset/confirm", authServer.ConfirmPasswordReset)
	}

	return router
}
