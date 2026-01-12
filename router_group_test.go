package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterGroupPrefix(t *testing.T) {
	router := NewRouter()

	// Case 1: Normal clean paths
	v1 := router.Group("/v1")
	v1.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	// Case 2: Dirty paths that should be cleaned by path.Join
	// "/api" + "///nested" -> "/api/nested"
	api := router.Group("/api/")
	nested := api.Group("///nested//")
	nested.Get("//resource", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("resource"))
	})

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantBody string
	}{
		{"Clean Path", "/v1/users", 200, "users"},
		{"Dirty Path Auto-clean", "/api/nested/resource", 200, "resource"},
		{"NotFound", "/v1/unknown", 404, "Tidak ditemukan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tt.path, nil)
			router.ServeHTTP(w, r)

			if w.Code != tt.wantCode {
				t.Errorf("url %s: status = %d, want %d", tt.path, w.Code, tt.wantCode)
			}

			if tt.wantBody != "" && w.Code == 200 {
				if w.Body.String() != tt.wantBody { // Note: JsonError response body logic in router might vary, checking plain here
					// For 404 router returns JSON usually, but for 200 we wrote plain text
					// Only check body for success to avoid JSON parsing in this simple test
					t.Errorf("url %s: body = %s, want %s", tt.path, w.Body.String(), tt.wantBody)
				}
			}
		})
	}
}

func TestRouterGroupMiddleware(t *testing.T) {
	router := NewRouter()
	var flow []string

	// Middleware markers
	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			flow = append(flow, "root")
			next(w, r)
		}
	}
	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			flow = append(flow, "group")
			next(w, r)
		}
	}
	mw3 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			flow = append(flow, "route")
			next(w, r)
		}
	}

	// Setup hierarchy
	router.Use(mw1)
	api := router.Group("/api", mw2)
	api.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		flow = append(flow, "handler")
		w.WriteHeader(200)
	}, mw3)

	// Execute
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, r)

	// Validate Order: Root -> Group -> Route -> Handler
	expected := []string{"root", "group", "route", "handler"}

	if len(flow) != len(expected) {
		t.Fatalf("Middleware flow length mismatch. Got %v, want %v", flow, expected)
	}

	for i, v := range expected {
		if flow[i] != v {
			t.Errorf("Middleware flow at index %d mismatch. Got %s, want %s", i, flow[i], v)
		}
	}
}

func TestRouterGroupUse(t *testing.T) {
	router := NewRouter()
	var flow []string

	group := router.Group("/group")

	// Route registered BEFORE Use() - should NOT have the middleware
	group.Get("/early", func(w http.ResponseWriter, r *http.Request) {
		flow = append(flow, "handler-early")
	})

	// Add middleware via Use()
	group.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			flow = append(flow, "late-middleware")
			next(w, r)
		}
	})

	// Route registered AFTER Use() - SHOULD have the middleware
	group.Get("/late", func(w http.ResponseWriter, r *http.Request) {
		flow = append(flow, "handler-late")
	})

	// Test 1: Early route (No middleware)
	flow = nil
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/group/early", nil)
	router.ServeHTTP(w, r)

	if len(flow) != 1 || flow[0] != "handler-early" {
		t.Errorf("Early route unexpected flow: %v", flow)
	}

	// Test 2: Late route (With middleware)
	flow = nil
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/group/late", nil)
	router.ServeHTTP(w, r)

	if len(flow) != 2 || flow[0] != "late-middleware" || flow[1] != "handler-late" {
		t.Errorf("Late route unexpected flow: %v", flow)
	}
}

func TestNestedGroups(t *testing.T) {
	router := NewRouter()

	// /api
	api := router.Group("/api")

	// /api/v1
	v1 := api.Group("/v1")

	// /api/v1/users
	users := v1.Group("/users")

	users.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := GetParam(r, "id")
		w.Write([]byte("user:" + id))
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users/99", nil)
	router.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("Nested group routing failed. Code: %d", w.Code)
	}
	if w.Body.String() != "user:99" {
		t.Errorf("Nested group param failed. Body: %s", w.Body.String())
	}
}
