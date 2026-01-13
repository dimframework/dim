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

func TestRouterMethodCaseInsensitive(t *testing.T) {
	router := NewRouter()
	called := false

	// Register with lowercase "get" - should be normalized to GET
	router.Register("get", "/lower", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}, nil)

	w := httptest.NewRecorder()
	// Request with uppercase "GET" (standard)
	r := httptest.NewRequest("GET", "/lower", nil)
	router.ServeHTTP(w, r)

	if !called {
		t.Errorf("Router did not handle uppercase Request against lowercase Registration")
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	router := NewRouter()

	// Register GET only
	router.Get("/only-get", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	w := httptest.NewRecorder()
	// Request POST
	r := httptest.NewRequest("POST", "/only-get", nil)
	router.ServeHTTP(w, r)

	// Expect 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed, got %d", w.Code)
	}
}

// ============================================================================
// GetRoutes() Tests - CLI Route Introspection
// ============================================================================

func TestRouter_GetRoutes_Empty(t *testing.T) {
	router := NewRouter()

	routes := router.GetRoutes()

	if routes == nil {
		t.Fatal("GetRoutes returned nil")
	}

	if len(routes) != 0 {
		t.Errorf("Expected 0 routes, got %d", len(routes))
	}
}

func TestRouter_GetRoutes_SingleRoute(t *testing.T) {
	router := NewRouter()

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})

	routes := router.GetRoutes()

	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	route := routes[0]
	if route.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", route.Method)
	}

	if route.Path != "/users" {
		t.Errorf("Expected path '/users', got '%s'", route.Path)
	}

	if route.Handler == "" {
		t.Error("Handler name should not be empty")
	}
}

func TestRouter_GetRoutes_MultipleRoutes(t *testing.T) {
	router := NewRouter()

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})
	router.Post("/users", func(w http.ResponseWriter, r *http.Request) {})
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {})

	routes := router.GetRoutes()

	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d", len(routes))
	}

	// Verify first route
	if routes[0].Method != "GET" || routes[0].Path != "/users" {
		t.Errorf("Route 0: got %s %s", routes[0].Method, routes[0].Path)
	}

	// Verify second route
	if routes[1].Method != "POST" || routes[1].Path != "/users" {
		t.Errorf("Route 1: got %s %s", routes[1].Method, routes[1].Path)
	}

	// Verify third route
	if routes[2].Method != "GET" || routes[2].Path != "/users/{id}" {
		t.Errorf("Route 2: got %s %s", routes[2].Method, routes[2].Path)
	}
}

func TestRouter_GetRoutes_WithMiddleware(t *testing.T) {
	router := NewRouter()

	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
		}
	}

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {}, middleware1, middleware2)

	routes := router.GetRoutes()

	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	route := routes[0]
	if len(route.Middlewares) != 2 {
		t.Fatalf("Expected 2 middlewares, got %d", len(route.Middlewares))
	}

	// Middleware names should not be empty
	for i, mw := range route.Middlewares {
		if mw == "" {
			t.Errorf("Middleware %d name should not be empty", i)
		}
	}
}

func TestRouter_GetRoutes_NoMiddleware(t *testing.T) {
	router := NewRouter()

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})

	routes := router.GetRoutes()

	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	route := routes[0]
	if len(route.Middlewares) != 0 {
		t.Errorf("Expected 0 middlewares, got %d", len(route.Middlewares))
	}
}

func TestRouter_GetRoutes_AllHTTPMethods(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request) {}

	router.Get("/test", handler)
	router.Post("/test", handler)
	router.Put("/test", handler)
	router.Delete("/test", handler)
	router.Patch("/test", handler)
	router.Options("/test", handler)
	router.Head("/test", handler)

	routes := router.GetRoutes()

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	if len(routes) != len(expectedMethods) {
		t.Fatalf("Expected %d routes, got %d", len(expectedMethods), len(routes))
	}

	for i, expectedMethod := range expectedMethods {
		if routes[i].Method != expectedMethod {
			t.Errorf("Route %d: expected method '%s', got '%s'", i, expectedMethod, routes[i].Method)
		}
	}
}

func TestRouter_GetRoutes_IsolationCopy(t *testing.T) {
	router := NewRouter()

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})

	// Get routes twice
	routes1 := router.GetRoutes()
	routes2 := router.GetRoutes()

	// Should be separate copies
	if len(routes1) != len(routes2) {
		t.Fatalf("Routes length mismatch: %d vs %d", len(routes1), len(routes2))
	}

	// Modifying routes1 should not affect routes2
	routes1[0].Path = "/modified"

	if routes2[0].Path == "/modified" {
		t.Error("GetRoutes() should return isolated copies, but modification affected second call")
	}
}

func TestRouter_GetRoutes_Caching(t *testing.T) {
	router := NewRouter()

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})

	// Build should cache routes
	router.Build()

	routes := router.GetRoutes()

	if len(routes) != 1 {
		t.Fatalf("Expected 1 route after Build(), got %d", len(routes))
	}

	if routes[0].Method != "GET" || routes[0].Path != "/users" {
		t.Errorf("Route info incorrect after Build()")
	}
}

func TestRouter_GetRoutes_AfterNewRegistration(t *testing.T) {
	router := NewRouter()

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})

	routes := router.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	// Register new route
	router.Post("/users", func(w http.ResponseWriter, r *http.Request) {})

	routes = router.GetRoutes()
	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes after new registration, got %d", len(routes))
	}
}
