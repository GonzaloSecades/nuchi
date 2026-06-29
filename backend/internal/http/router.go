package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires the API routes owned by this service. It is intentionally
// small for the scaffold issue; future issues add auth, OpenAPI handlers, and
// database-backed resources.
func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/api/health", healthHandler)

	return router
}
