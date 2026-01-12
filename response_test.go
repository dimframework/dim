package dim

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJson(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"id":   1,
		"name": "John",
	}

	Json(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header not set correctly")
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["id"] != float64(1) || result["name"] != "John" {
		t.Errorf("response data mismatch")
	}
}

func TestJsonArray(t *testing.T) {
	w := httptest.NewRecorder()
	data := []map[string]interface{}{
		{"id": 1, "name": "John"},
		{"id": 2, "name": "Jane"},
	}

	Json(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var result []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result) != 2 {
		t.Errorf("array length mismatch")
	}
}

func TestJsonPagination(t *testing.T) {
	w := httptest.NewRecorder()
	data := []map[string]interface{}{
		{"id": 1, "name": "John"},
	}
	meta := PaginationMeta{
		Page:       1,
		PerPage:    10,
		Total:      100,
		TotalPages: 10,
	}

	JsonPagination(w, http.StatusOK, data, meta)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var result PaginationResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Meta.Page != 1 || result.Meta.Total != 100 {
		t.Errorf("pagination meta mismatch")
	}
}

func TestJsonError(t *testing.T) {
	w := httptest.NewRecorder()
	errors := map[string]string{
		"email":    "invalid email",
		"password": "too weak",
	}

	JsonError(w, http.StatusBadRequest, "Validation failed", errors)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var result ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Message != "Validation failed" {
		t.Errorf("error message mismatch")
	}

	if len(result.Errors) != 2 {
		t.Errorf("expected 2 field errors, got %d", len(result.Errors))
	}
}

func TestJsonErrorNoFields(t *testing.T) {
	w := httptest.NewRecorder()

	JsonError(w, http.StatusUnauthorized, "Unauthorized", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code mismatch")
	}

	var result ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Message != "Unauthorized" {
		t.Errorf("error message mismatch")
	}

	if len(result.Errors) > 0 {
		t.Errorf("errors should be empty")
	}
}

func TestJsonAppError(t *testing.T) {
	w := httptest.NewRecorder()
	appErr := NewAppError("validation failed", 400).
		WithFieldError("email", "invalid format")

	JsonAppError(w, appErr)

	if w.Code != 400 {
		t.Errorf("status code = %d, want 400", w.Code)
	}

	var result ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Message != "validation failed" {
		t.Errorf("message mismatch")
	}

	if result.Errors["email"] != "invalid format" {
		t.Errorf("field error mismatch")
	}
}

func TestSetHeader(t *testing.T) {
	w := httptest.NewRecorder()
	SetHeader(w, "X-Custom", "value")

	if w.Header().Get("X-Custom") != "value" {
		t.Errorf("header not set correctly")
	}
}

func TestSetHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	headers := map[string]string{
		"X-Custom-1": "value1",
		"X-Custom-2": "value2",
	}
	SetHeaders(w, headers)

	if w.Header().Get("X-Custom-1") != "value1" || w.Header().Get("X-Custom-2") != "value2" {
		t.Errorf("headers not set correctly")
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{"id": 1}

	Created(w, data)

	if w.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{"id": 1}

	OK(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestShorthandErrors(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(http.ResponseWriter, string, map[string]string) error
		status  int
		message string
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest, "bad request"},
		{"Conflict", Conflict, http.StatusConflict, "conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.fn(w, tt.message, nil)

			if w.Code != tt.status {
				t.Errorf("status code = %d, want %d", w.Code, tt.status)
			}
		})
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Unauthorized(w, "Invalid credentials")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code mismatch")
	}
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Forbidden(w, "Access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("status code mismatch")
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "Resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status code mismatch")
	}
}

func TestInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalServerError(w, "Something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code mismatch")
	}
}
