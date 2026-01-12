package main

import (
	"net/http"

	"github.com/nuradiyana/dim"
)

type CommonHandler struct{}

func NewCommonHandler() *CommonHandler {
	return &CommonHandler{}
}

// HealthHandler returns the health status of the server
func (h *CommonHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	dim.Json(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// ProfileHandler returns the current user's profile
func (h *CommonHandler) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := dim.GetUser(r)
	if !ok {
		dim.JsonError(w, http.StatusUnauthorized, "User not found in context", nil)
		return
	}

	dim.Json(w, http.StatusOK, user)
}

// NotFoundHandler handles 404 errors
func (h *CommonHandler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	dim.JsonError(w, http.StatusNotFound, "Endpoint not found", nil)
}
