package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// patternOf serves method+path through router and returns the pattern the
// handler saw, or "<not called>" when no handler ran.
func patternOf(t *testing.T, router *Router, method, path string) string {
	t.Helper()
	got := "<not called>"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	if v := w.Header().Get("X-Pattern"); w.Header().Get("X-Called") == "1" {
		got = v
	}
	return got
}

// recordPattern is a handler that reports GetRoutePattern through headers.
func recordPattern(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Called", "1")
	w.Header().Set("X-Pattern", GetRoutePattern(r))
}

func TestGetRoutePattern_ParamRoute(t *testing.T) {
	router := NewRouter()
	router.Get("/users/{id}", recordPattern)
	router.Get("/users/{id}/posts/{postID}", recordPattern)

	if got := patternOf(t, router, "GET", "/users/42"); got != "/users/{id}" {
		t.Errorf("pattern = %q, want %q", got, "/users/{id}")
	}
	if got := patternOf(t, router, "GET", "/users/42/posts/7"); got != "/users/{id}/posts/{postID}" {
		t.Errorf("pattern = %q, want %q", got, "/users/{id}/posts/{postID}")
	}
}

// Static routes live in the static map, not the radix tree, and carry no
// params — the shape most easily left returning "".
func TestGetRoutePattern_StaticRoute(t *testing.T) {
	router := NewRouter()
	router.Get("/health", recordPattern)
	router.Post("/api/v1/users", recordPattern)
	router.Get("/", recordPattern)

	cases := []struct{ method, path, want string }{
		{"GET", "/health", "/health"},
		{"POST", "/api/v1/users", "/api/v1/users"},
		{"GET", "/", "/"},
	}
	for _, tc := range cases {
		if got := patternOf(t, router, tc.method, tc.path); got != tc.want {
			t.Errorf("%s %s: pattern = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestGetRoutePattern_CatchAll(t *testing.T) {
	router := NewRouter()
	router.Get("/files/{path...}", recordPattern)

	for _, path := range []string{"/files/a", "/files/a/b/c.png", "/files/"} {
		if got := patternOf(t, router, "GET", path); got != "/files/{path...}" {
			t.Errorf("%s: pattern = %q, want %q", path, got, "/files/{path...}")
		}
	}
}

func TestGetRoutePattern_NestedGroups(t *testing.T) {
	router := NewRouter()
	api := router.Group("/api")
	tenant := api.Group("/v1/{tenant}")
	tenant.Get("/users/{id}", recordPattern)
	tenant.Get("/settings", recordPattern)
	api.Get("/status", recordPattern)

	cases := []struct{ path, want string }{
		{"/api/v1/acme/users/42", "/api/v1/{tenant}/users/{id}"},
		{"/api/v1/acme/settings", "/api/v1/{tenant}/settings"},
		{"/api/status", "/api/status"},
	}
	for _, tc := range cases {
		if got := patternOf(t, router, "GET", tc.path); got != tc.want {
			t.Errorf("%s: pattern = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// insert consumes the pattern as it descends and splits existing nodes when a
// later route shares only part of a prefix; neither may leak into the pattern.
func TestGetRoutePattern_SurvivesNodeSplits(t *testing.T) {
	router := NewRouter()
	router.Get("/users/{id}", recordPattern)
	router.Get("/u/{name}", recordPattern)                // splits "/users/" into "/u" + "sers/"
	router.Get("/campaign/{slug}", recordPattern)         // shares "/" only
	router.Get("/campaign/{id}/donations", recordPattern) // sibling param with another name

	cases := []struct{ path, want string }{
		{"/users/42", "/users/{id}"},
		{"/u/budi", "/u/{name}"},
		{"/campaign/banjir", "/campaign/{slug}"},
		{"/campaign/9/donations", "/campaign/{id}/donations"},
	}
	for _, tc := range cases {
		if got := patternOf(t, router, "GET", tc.path); got != tc.want {
			t.Errorf("%s: pattern = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// Metrics and access logs are written by global middleware after next returns,
// outside the router's dispatch — the pattern must be visible there too.
func TestGetRoutePattern_GlobalMiddlewareAfterNext(t *testing.T) {
	router := NewRouter()
	var seen string
	router.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
			seen = GetRoutePattern(r)
		}
	})
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {})
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {})

	cases := []struct{ path, want string }{
		{"/users/42", "/users/{id}"},
		{"/health", "/health"},
		{"/nope", ""},
	}
	for _, tc := range cases {
		seen = "<unset>"
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", tc.path, nil))
		if seen != tc.want {
			t.Errorf("%s: middleware saw pattern %q, want %q", tc.path, seen, tc.want)
		}
	}
}

func TestGetRoutePattern_RouteMiddleware(t *testing.T) {
	router := NewRouter()
	var seen string
	mw := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			seen = GetRoutePattern(r)
			next(w, r)
		}
	}
	router.Get("/reset/{token}", func(w http.ResponseWriter, r *http.Request) {}, mw)

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/reset/s3cr3t", nil))
	if seen != "/reset/{token}" {
		t.Errorf("route middleware saw pattern %q, want %q", seen, "/reset/{token}")
	}
}

func TestGetRoutePattern_EmptyWhenNothingMatched(t *testing.T) {
	router := NewRouter()
	router.RedirectTrailingSlash(true)
	var seen string
	router.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
			seen = GetRoutePattern(r)
		}
	})
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {})
	router.Post("/login", func(w http.ResponseWriter, r *http.Request) {})

	cases := []struct {
		name, method, path string
		code               int
	}{
		{"404", "GET", "/missing", http.StatusNotFound},
		{"405 static", "GET", "/login", http.StatusMethodNotAllowed},
		{"405 tree", "DELETE", "/users/1", http.StatusMethodNotAllowed},
		{"trailing slash redirect", "GET", "/users/1/", http.StatusMovedPermanently},
	}
	for _, tc := range cases {
		seen = "<unset>"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.code {
			t.Fatalf("%s: status = %d, want %d", tc.name, w.Code, tc.code)
		}
		if seen != "" {
			t.Errorf("%s: pattern = %q, want empty", tc.name, seen)
		}
	}
}

func TestGetRoutePattern_EmptyForStaticAndSPA(t *testing.T) {
	router := NewRouter()
	fsys := fstest.MapFS{
		"app.js":     {Data: []byte("x")},
		"index.html": {Data: []byte("<html></html>")},
	}
	var seen string
	router.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
			seen = GetRoutePattern(r)
		}
	})
	router.Static("/assets/", fsys)
	router.SPA(fsys, "index.html")

	for _, path := range []string{"/assets/app.js", "/dashboard"} {
		seen = "<unset>"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", path, w.Code)
		}
		if seen != "" {
			t.Errorf("%s: pattern = %q, want empty", path, seen)
		}
	}
}

func TestGetRoutePattern_RequestOutsideRouter(t *testing.T) {
	r := httptest.NewRequest("GET", "/users/42", nil)
	if got := GetRoutePattern(r); got != "" {
		t.Errorf("pattern = %q, want empty", got)
	}
}

// A dim Router mounted inside another: each router's middleware sees its own
// pattern, and params captured by the outer router stay readable by handlers
// of an inner route that captured none — as they did before routeMatch.
func TestGetRoutePattern_NestedRouters(t *testing.T) {
	inner := NewRouter()
	inner.Get("/t/{tenant}/users/{id}", recordPattern)
	inner.Get("/t/{tenant}/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Called", "1")
		w.Header().Set("X-Pattern", GetRoutePattern(r))
		w.Header().Set("X-Tenant", GetParam(r, "tenant"))
	})

	outer := NewRouter()
	var outerSeen string
	outer.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
			outerSeen = GetRoutePattern(r)
		}
	})
	outer.Get("/t/{tenant}/{rest...}", inner.ServeHTTP)

	if got := patternOf(t, outer, "GET", "/t/acme/users/1"); got != "/t/{tenant}/users/{id}" {
		t.Errorf("inner handler pattern = %q, want %q", got, "/t/{tenant}/users/{id}")
	}
	if outerSeen != "/t/{tenant}/{rest...}" {
		t.Errorf("outer middleware pattern = %q, want %q", outerSeen, "/t/{tenant}/{rest...}")
	}

	w := httptest.NewRecorder()
	outer.ServeHTTP(w, httptest.NewRequest("GET", "/t/acme/health", nil))
	if got := w.Header().Get("X-Tenant"); got != "acme" {
		t.Errorf("inner handler tenant = %q, want %q", got, "acme")
	}
}

func TestCtx_RoutePattern(t *testing.T) {
	router := NewRouter()
	var got string
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		got = Of(w, r).RoutePattern()
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/users/42", nil))
	if got != "/users/{id}" {
		t.Errorf("RoutePattern = %q, want %q", got, "/users/{id}")
	}
}
