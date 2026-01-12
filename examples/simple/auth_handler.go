package main

import (
	"encoding/json"
	"net/http"

	"github.com/nuradiyana/dim"
)

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenRequest is the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResponse is the response for login/register endpoints
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type AuthHandler struct {
	authService *dim.AuthService
}

func NewAuthHandler(authService *dim.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterHandler handles user registration
func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dim.JsonError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	defer r.Body.Close()

	// Validate input
	validator := dim.NewValidator()
	validator.Required("email", req.Email).
		Email("email", req.Email).
		Required("username", req.Username).
		MinLength("username", req.Username, 3)

	if err := dim.ValidatePasswordStrength(req.Password); err != nil {
		if appErr, ok := err.(*dim.AppError); ok {
			// Extract the validation errors and add them to the main validator
			for k, v := range appErr.Errors {
				validator.AddError(k, v)
			}
		}
	}

	if !validator.IsValid() {
		dim.JsonError(w, http.StatusBadRequest, "Validation failed", validator.ErrorMap())
		return
	}

	// Register user
	user, err := h.authService.Register(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		if appErr, ok := err.(*dim.AppError); ok {
			dim.JsonError(w, appErr.StatusCode, appErr.Message, appErr.Errors)
		} else {
			dim.JsonError(w, http.StatusInternalServerError, "Failed to register user", nil)
		}
		return
	}

	dim.Json(w, http.StatusCreated, user)
}

// LoginHandler handles user login
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dim.JsonError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	defer r.Body.Close()

	// Validate input
	validator := dim.NewValidator()
	validator.Required("email", req.Email).
		Email("email", req.Email).
		Required("password", req.Password)

	if !validator.IsValid() {
		dim.JsonError(w, http.StatusBadRequest, "Validation failed", validator.ErrorMap())
		return
	}

	// Login
	accessToken, refreshToken, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if appErr, ok := err.(*dim.AppError); ok {
			dim.JsonError(w, appErr.StatusCode, appErr.Message, appErr.Errors)
		} else {
			dim.JsonError(w, http.StatusInternalServerError, "Failed to login", nil)
		}
		return
	}

	dim.Json(w, http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	})
}

// RefreshTokenHandler handles token refresh
func (h *AuthHandler) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dim.JsonError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	defer r.Body.Close()

	// Validate input
	validator := dim.NewValidator()
	validator.Required("refresh_token", req.RefreshToken)

	if !validator.IsValid() {
		dim.JsonError(w, http.StatusBadRequest, "Validation failed", validator.ErrorMap())
		return
	}

	// Refresh token
	accessToken, refreshToken, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if appErr, ok := err.(*dim.AppError); ok {
			dim.JsonError(w, appErr.StatusCode, appErr.Message, appErr.Errors)
		} else {
			dim.JsonError(w, http.StatusInternalServerError, "Failed to refresh token", nil)
		}
		return
	}

	dim.Json(w, http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	})
}

// LogoutHandler handles user logout
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dim.JsonError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	defer r.Body.Close()

	// Logout
	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		if appErr, ok := err.(*dim.AppError); ok {
			dim.JsonError(w, appErr.StatusCode, appErr.Message, appErr.Errors)
		} else {
			dim.JsonError(w, http.StatusInternalServerError, "Failed to logout", nil)
		}
		return
	}

	dim.Json(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}
