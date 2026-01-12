package main

import (
	"github.com/nuradiyana/dim"
)

// setupHandlers registers all HTTP handlers to the router
func setupHandlers(router *dim.Router, authService *dim.AuthService, userStore *dim.PostgresUserStore, jwtManager *dim.JWTManager) {

	// Common handler
	commonHandler := NewCommonHandler()

	// Health check endpoint
	router.Get("/health", commonHandler.HealthHandler)

	// Auth handler
	authHandler := NewAuthHandler(authService)

	// Public auth routes
	router.Post("/auth/register", authHandler.RegisterHandler)
	router.Post("/auth/login", authHandler.LoginHandler)
	router.Post("/auth/refresh", authHandler.RefreshTokenHandler)

	// User handler
	userHandler := NewUserHandler(userStore)

	// Protected routes
	protected := router.Group("/api", dim.RequireAuth(jwtManager))
	protected.Get("/profile", commonHandler.ProfileHandler)
	protected.Patch("/profile", userHandler.UpdateProfileHandler)
	protected.Post("/logout", authHandler.LogoutHandler)

	// NotFound handler
	router.SetNotFound(commonHandler.NotFoundHandler)
}
