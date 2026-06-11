package dim

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newCtxRequest(method, path string, body string) (*httptest.ResponseRecorder, *http.Request) {
	var b *strings.Reader
	if body != "" {
		b = strings.NewReader(body)
	} else {
		b = strings.NewReader("")
	}
	r := httptest.NewRequest(method, path, b)
	w := httptest.NewRecorder()
	return w, r
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func TestOf(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	if c == nil {
		t.Fatal("Of returned nil")
	}
}

// ─── Request helpers ─────────────────────────────────────────────────────────

func TestCtx_Param(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/users/42", "")
	rp := &routeParams{keys: []string{"id"}, vals: []string{"42"}}
	r = r.WithContext(context.WithValue(r.Context(), paramsKey, rp))

	c := Of(w, r)
	if got := c.Param("id"); got != "42" {
		t.Errorf("Param = %q, want %q", got, "42")
	}
}

func TestCtx_Query(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/?page=3", "")
	c := Of(w, r)
	if got := c.Query("page"); got != "3" {
		t.Errorf("Query = %q, want %q", got, "3")
	}
	if got := c.Query("missing"); got != "" {
		t.Errorf("Query missing key = %q, want empty", got)
	}
}

func TestCtx_Queries(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/?page=2&limit=10", "")
	c := Of(w, r)
	got := c.Queries("page", "limit")
	if got["page"] != "2" || got["limit"] != "10" {
		t.Errorf("Queries = %v, want page=2 limit=10", got)
	}
}

func TestCtx_Header(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	r.Header.Set("X-Custom", "hello")
	c := Of(w, r)
	if got := c.Header("X-Custom"); got != "hello" {
		t.Errorf("Header = %q, want %q", got, "hello")
	}
}

func TestCtx_Cookie(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	r.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	c := Of(w, r)
	if got := c.Cookie("session"); got != "abc" {
		t.Errorf("Cookie = %q, want %q", got, "abc")
	}
	if got := c.Cookie("missing"); got != "" {
		t.Errorf("Cookie missing = %q, want empty", got)
	}
}

func TestCtx_AuthToken(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	r.Header.Set("Authorization", "Bearer mytoken")
	c := Of(w, r)
	tok, ok := c.AuthToken()
	if !ok || tok != "mytoken" {
		t.Errorf("AuthToken = (%q, %v), want (mytoken, true)", tok, ok)
	}
}

func TestCtx_AuthToken_Missing(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	_, ok := c.AuthToken()
	if ok {
		t.Error("AuthToken should return false when header absent")
	}
}

func TestCtx_RequestID(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	r = SetRequestID(r, "req-123")
	c := Of(w, r)
	if got := c.RequestID(); got != "req-123" {
		t.Errorf("RequestID = %q, want %q", got, "req-123")
	}
}

func TestCtx_ClientIP(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	r.Header.Set("X-Real-IP", "1.2.3.4")
	c := Of(w, r)
	if got := c.ClientIP(); got != "1.2.3.4" {
		t.Errorf("ClientIP = %q, want %q", got, "1.2.3.4")
	}
}

func TestCtx_User(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	_, ok := Of(w, r).User()
	if ok {
		t.Error("User should be absent on fresh request")
	}
}

func TestCtx_Claims(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	if got := Of(w, r).Claims(); got != nil {
		t.Errorf("Claims on unauthenticated request should be nil, got %v", got)
	}
}

// ─── Bind ────────────────────────────────────────────────────────────────────

func TestCtx_Bind(t *testing.T) {
	body := `{"name":"Alice","age":30}`
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c := Of(w, r)

	var payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := c.Bind(&payload); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if payload.Name != "Alice" || payload.Age != 30 {
		t.Errorf("Bind result = %+v, want {Alice 30}", payload)
	}
}

func TestCtx_Bind_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid}"))
	w := httptest.NewRecorder()
	c := Of(w, r)
	var v map[string]interface{}
	if err := c.Bind(&v); err == nil {
		t.Error("Bind should return error for invalid JSON")
	}
}

// ─── Validate ────────────────────────────────────────────────────────────────

func TestCtx_Validate(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	v := c.Validate()
	if v == nil {
		t.Fatal("Validate returned nil")
	}
	v.Required("name", "")
	if v.IsValid() {
		t.Error("Validator should report invalid when required field is empty")
	}
}

// ─── Response helpers ────────────────────────────────────────────────────────

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return m
}

func TestCtx_JSON(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.JSON(http.StatusAccepted, map[string]string{"status": "ok"})
	if w.Code != http.StatusAccepted {
		t.Errorf("JSON status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestCtx_OK(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.OK(map[string]string{"key": "val"})
	if w.Code != http.StatusOK {
		t.Errorf("OK status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCtx_Created(t *testing.T) {
	w, r := newCtxRequest(http.MethodPost, "/", "")
	c := Of(w, r)
	c.Created(map[string]int{"id": 1})
	if w.Code != http.StatusCreated {
		t.Errorf("Created status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestCtx_NoContent(t *testing.T) {
	w, r := newCtxRequest(http.MethodDelete, "/", "")
	c := Of(w, r)
	c.NoContent()
	if w.Code != http.StatusNoContent {
		t.Errorf("NoContent status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestCtx_BadRequest(t *testing.T) {
	w, r := newCtxRequest(http.MethodPost, "/", "")
	c := Of(w, r)
	c.BadRequest("invalid", FieldErrors{"field": "required"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("BadRequest status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	body := decodeJSON(t, w)
	if body["message"] != "invalid" {
		t.Errorf("BadRequest message = %v", body["message"])
	}
}

func TestCtx_Unauthorized(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.Unauthorized("not auth")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Unauthorized status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCtx_Forbidden(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.Forbidden("forbidden")
	if w.Code != http.StatusForbidden {
		t.Errorf("Forbidden status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCtx_NotFound(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.NotFound("not found")
	if w.Code != http.StatusNotFound {
		t.Errorf("NotFound status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCtx_Conflict(t *testing.T) {
	w, r := newCtxRequest(http.MethodPost, "/", "")
	c := Of(w, r)
	c.Conflict("conflict", FieldErrors{"email": "taken"})
	if w.Code != http.StatusConflict {
		t.Errorf("Conflict status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestCtx_InternalServerError(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.InternalServerError("oops")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ISE status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCtx_TooManyRequests(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)
	c.TooManyRequests(60)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("TooManyRequests status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if w.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want %q", w.Header().Get("Retry-After"), "60")
	}
}

func TestCtx_AppError(t *testing.T) {
	w, r := newCtxRequest(http.MethodPost, "/", "")
	c := Of(w, r)
	appErr := NewAppError("validasi gagal", http.StatusBadRequest).
		WithFieldError("email", "tidak valid")
	c.AppError(appErr)
	if w.Code != http.StatusBadRequest {
		t.Errorf("AppError status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	body := decodeJSON(t, w)
	if body["message"] != "validasi gagal" {
		t.Errorf("AppError message = %v", body["message"])
	}
	errs, _ := body["errors"].(map[string]interface{})
	if errs["email"] != "tidak valid" {
		t.Errorf("AppError field error = %v", errs)
	}
}
