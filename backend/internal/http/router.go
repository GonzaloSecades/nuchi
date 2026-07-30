package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires the API routes owned by this service. authServer is
// optional (nil skips mounting the /api/auth/* routes) so callers that only
// need health/scaffold behavior — e.g. tests unrelated to auth — are not
// forced to construct a database pool. resources is likewise optional (nil
// skips mounting the owned-resource routes) so existing tests that build a
// router without a pool keep working unchanged.
//
// /api/health and /api/auth/* stay public (the latter is how a client gets
// a token in the first place). Every owned-resource route (accounts,
// categories, transactions, summary — mounted by #44-#48) belongs inside
// the RequireAuth group below, so the Bearer-JWT check happens once at the
// router level instead of being repeated per handler the way the legacy
// Hono routes each ran their own clerkMiddleware() + inline check.
func NewRouter(authServer *AuthServer, resources *ResourceServer) http.Handler {
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

			if resources != nil {
				r.Route("/api/accounts", func(r chi.Router) {
					r.Get("/", resources.ListAccounts)
					r.Post("/", resources.CreateAccount)
					r.Post("/bulk-delete", resources.BulkDeleteAccounts)
					r.Get("/{id}", withResourceID(resources.GetAccount))
					r.Patch("/{id}", withResourceID(resources.UpdateAccount))
					r.Delete("/{id}", withResourceID(resources.DeleteAccount))
				})

				r.Route("/api/categories", func(r chi.Router) {
					r.Get("/", resources.ListCategories)
					r.Post("/", resources.CreateCategory)
					r.Post("/bulk-delete", resources.BulkDeleteCategories)
					r.Get("/{id}", withResourceID(resources.GetCategory))
					r.Patch("/{id}", withResourceID(resources.UpdateCategory))
					r.Delete("/{id}", withResourceID(resources.DeleteCategory))
				})

				r.Route("/api/transactions", func(r chi.Router) {
					r.Get("/", withListTransactionsParams(resources.ListTransactions))
					r.Post("/", resources.CreateTransaction)
					// Static paths are registered before /{id} for readability;
					// chi's trie prefers a static segment over a parameter
					// regardless of declaration order, and these are POST-only
					// while /{id} has no POST, so there is no ambiguity either
					// way.
					r.Post("/bulk-create", resources.BulkCreateTransactions)
					r.Post("/bulk-delete", resources.BulkDeleteTransactions)
					r.Get("/{id}", withResourceID(resources.GetTransaction))
					r.Patch("/{id}", withResourceID(resources.UpdateTransaction))
					r.Delete("/{id}", withResourceID(resources.DeleteTransaction))
				})
			}
		})
	}

	return router
}
