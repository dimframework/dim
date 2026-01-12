package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterBasicRoute(t *testing.T) {
	router := NewRouter()
	called := false

	router.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/hello", nil)
	router.ServeHTTP(w, r)

	if !called {
		t.Errorf("handler not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want 200", w.Code)
	}
}

func TestNewRouterPathParameter(t *testing.T) {
	router := NewRouter()

	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(id))
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/123", nil)
	router.ServeHTTP(w, r)

	if w.Body.String() != "123" {
		t.Errorf("param value = %s, want 123", w.Body.String())
	}
}

func TestNewRouterCatchAll(t *testing.T) {
	router := NewRouter()

	router.Get("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := GetParam(r, "path")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(path))
	})

	tests := []struct {
		url      string
		expected string
	}{
		{"/files/doc.pdf", "doc.pdf"},
		{"/files/images/photo.jpg", "images/photo.jpg"},
		{"/files/deep/nested/path.txt", "deep/nested/path.txt"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", tt.url, nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("url %s: status = %d, want 200", tt.url, w.Code)
		}

		if w.Body.String() != tt.expected {
			t.Errorf("url %s: path = %s, want %s", tt.url, w.Body.String(), tt.expected)
		}
	}
}

func TestNewRouterMultipleMethods(t *testing.T) {
	router := NewRouter()
	var methods []string

	handler := func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusOK)
	}

	router.Get("/api", handler)
	router.Post("/api", handler)
	router.Put("/api", handler)
	router.Delete("/api", handler)

	tests := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range tests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api", nil)
		router.ServeHTTP(w, r)
	}

	if len(methods) != 4 {
		t.Errorf("expected 4 handlers called, got %d", len(methods))
	}
}

func TestNewRouterGlobalMiddleware(t *testing.T) {
	router := NewRouter()
	var order []string

	m1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1")
			next(w, r)
		}
	}

	router.Use(m1)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, r)

	if len(order) != 2 || order[0] != "m1" || order[1] != "handler" {
		t.Errorf("middleware order = %v, want [m1 handler]", order)
	}
}

func TestNewRouterRouteMiddleware(t *testing.T) {
	router := NewRouter()
	var middlewareCalled bool

	m := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next(w, r)
		}
	}

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, m)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, r)

	if !middlewareCalled {
		t.Errorf("route middleware not called")
	}
}

func TestNewRouterMiddlewareOrder(t *testing.T) {
	router := NewRouter()
	var execOrder []string

	globalMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			execOrder = append(execOrder, "global-before")
			next(w, r)
			execOrder = append(execOrder, "global-after")
		}
	}

	routeMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			execOrder = append(execOrder, "route-before")
			next(w, r)
			execOrder = append(execOrder, "route-after")
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		execOrder = append(execOrder, "handler")
		w.WriteHeader(http.StatusOK)
	}

	router.Use(globalMiddleware)
	router.Get("/test", handler, routeMiddleware)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, r)

	expected := []string{"global-before", "route-before", "handler", "route-after", "global-after"}

	if len(execOrder) != len(expected) {
		t.Errorf("Execution order length mismatch: got %d, want %d", len(execOrder), len(expected))
	}

	for i := range expected {
		if i >= len(execOrder) {
			break
		}
		if execOrder[i] != expected[i] {
			t.Errorf("Execution order[%d] = %s, want %s", i, execOrder[i], expected[i])
			t.Logf("Full execution order: %v", execOrder)
			t.Logf("Expected order:      %v", expected)
			break
		}
	}
}
