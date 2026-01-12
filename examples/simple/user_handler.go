package main

import (
	"encoding/json"
	"net/http"

	"github.com/nuradiyana/dim"
)

type UserHandler struct {
	userStore *dim.PostgresUserStore
}

func NewUserHandler(userStore *dim.PostgresUserStore) *UserHandler {
	return &UserHandler{
		userStore: userStore,
	}
}

// UpdateProfileHandler handles PATCH /api/profile
// Allows partial updates - only updates fields that are sent
func (h *UserHandler) UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user, ok := dim.GetUser(r)
	if !ok {
		dim.JsonError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Parse request body
	var req dim.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dim.JsonError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// Validate fields
	v := dim.NewValidator()
	v.OptionalEmail("email", req.Email)
	v.OptionalMinLength("name", req.Name, 3)
	v.OptionalMaxLength("name", req.Name, 100)
	v.OptionalMinLength("password", req.Password, 8)

	if !v.IsValid() {
		dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.ErrorMap())
		return
	}

	// Perform partial update
	if err := h.userStore.UpdatePartial(r.Context(), user.ID, &req); err != nil {
		dim.JsonError(w, http.StatusInternalServerError, "Update failed", nil)
		return
	}

	// Fetch and return updated user
	updatedUser, err := h.userStore.FindByID(r.Context(), user.ID)
	if err != nil {
		dim.JsonError(w, http.StatusInternalServerError, "Failed to fetch user", nil)
		return
	}

	dim.Json(w, http.StatusOK, updatedUser)
}
