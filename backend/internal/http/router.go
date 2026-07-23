package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires the API routes owned by this service. authServer is
// optional (nil skips mounting the /api/auth/* routes) so callers that only
// need health/scaffold behavior — e.g. tests unrelated to auth — are not
// forced to construct a database pool.
//
// /api/health and /api/auth/* stay public (the latter is how a client gets
// a token in the first place). Every owned-resource route (accounts,
// categories, transactions, summary — mounted by #44-#48) belongs inside
// the RequireAuth group below, so the Bearer-JWT check happens once at the
// router level instead of being repeated per handler the way the legacy
// Hono routes each ran their own clerkMiddleware() + inline check.
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

		router.Group(func(r chi.Router) {
			r.Use(RequireAuth(authServer.jwtSecret))
			// Owned-resource routes (accounts, categories, transactions,
			// summary) mount here in #44-#48.
		})
	}

	return router
}
