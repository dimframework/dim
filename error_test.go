package dim

import (
	"errors"
	"testing"
)

func TestNewAppError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		statusCode int
		wantErr    string
	}{
		{
			name:       "create basic error",
			message:    "test error",
			statusCode: 400,
			wantErr:    "test error",
		},
		{
			name:       "create internal server error",
			message:    "something went wrong",
			statusCode: 500,
			wantErr:    "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAppError(tt.message, tt.statusCode)
			if err.Error() != tt.wantErr {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.wantErr)
			}
			if err.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", err.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestWithFieldError(t *testing.T) {
	err := NewAppError("validation failed", 400)
	result := err.WithFieldError("email", "invalid email format")

	if result.Errors["email"] != "invalid email format" {
		t.Errorf("field error not set correctly: got %v", result.Errors["email"])
	}

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 field error, got %d", len(result.Errors))
	}
}

func TestWithFieldErrors(t *testing.T) {
	err := NewAppError("validation failed", 400)
	fieldErrors := map[string]string{
		"email":    "invalid email format",
		"password": "password too weak",
	}
	result := err.WithFieldErrors(fieldErrors)

	if len(result.Errors) != 2 {
		t.Errorf("expected 2 field errors, got %d", len(result.Errors))
	}

	if result.Errors["email"] != "invalid email format" {
		t.Errorf("email error mismatch")
	}
	if result.Errors["password"] != "password too weak" {
		t.Errorf("password error mismatch")
	}
}

func TestErrorWithFieldErrors(t *testing.T) {
	err := NewAppError("validation failed", 400).
		WithFieldError("email", "invalid").
		WithFieldError("name", "required")

	errStr := err.Error()

	if len(err.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(err.Errors))
	}

	if !containsString(errStr, "validation failed") {
		t.Errorf("error message not in Error(): %s", errStr)
	}
}

func TestIsAppError(t *testing.T) {
	appErr := NewAppError("test", 400)
	genericErr := errors.New("generic error")

	if !IsAppError(appErr) {
		t.Errorf("IsAppError failed for AppError")
	}

	if IsAppError(genericErr) {
		t.Errorf("IsAppError should return false for non-AppError")
	}
}

func TestAsAppError(t *testing.T) {
	appErr := NewAppError("test", 400)
	genericErr := errors.New("generic error")

	if converted, ok := AsAppError(appErr); !ok || converted.Message != "test" {
		t.Errorf("AsAppError failed for AppError")
	}

	if _, ok := AsAppError(genericErr); ok {
		t.Errorf("AsAppError should return false for non-AppError")
	}
}

func TestCommonErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		wantStatus int
	}{
		{"BadRequest", ErrBadRequest, 400},
		{"Validation", ErrValidation, 400},
		{"Unauthorized", ErrUnauthorized, 401},
		{"Forbidden", ErrForbidden, 403},
		{"NotFound", ErrNotFound, 404},
		{"Conflict", ErrConflict, 409},
		{"InternalServerError", ErrInternalServerError, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.StatusCode != tt.wantStatus {
				t.Errorf("status code = %d, want %d", tt.err.StatusCode, tt.wantStatus)
			}
		})
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
