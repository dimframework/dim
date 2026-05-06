package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Benchmark Setup
// ============================================================================

// benchHandler is a minimal no-op handler used across all benchmarks.
var benchHandler = func(w http.ResponseWriter, r *http.Request) {}

// benchRoutes defines route patterns and a sample request path for each.
// Covers static, single-param, multi-param, and deeply-nested cases.
var benchRoutes = []struct {
	method  string
	pattern string
	reqPath string
}{
	{"GET", "/", "/"},
	{"GET", "/health", "/health"},
	{"GET", "/api/v1/status", "/api/v1/status"},
	{"POST", "/api/v1/users", "/api/v1/users"},
	{"GET", "/api/v1/users/{id}", "/api/v1/users/42"},
	{"PUT", "/api/v1/users/{id}", "/api/v1/users/42"},
	{"DELETE", "/api/v1/users/{id}", "/api/v1/users/42"},
	{"GET", "/api/v1/users/{id}/posts", "/api/v1/users/42/posts"},
	{"POST", "/api/v1/users/{id}/posts", "/api/v1/users/42/posts"},
	{"GET", "/api/v1/users/{id}/posts/{postID}", "/api/v1/users/42/posts/99"},
	{"GET", "/api/v1/users/{id}/posts/{postID}/comments", "/api/v1/users/42/posts/99/comments"},
	{"GET", "/files/{path...}", "/files/static/assets/logo.png"},
}

// ============================================================================
// Before: http.ServeMux (previous implementation baseline)
// ============================================================================

func setupServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	for _, r := range benchRoutes {
		mux.HandleFunc(r.method+" "+r.pattern, benchHandler)
	}
	return mux
}

func BenchmarkBefore_StaticRoute(b *testing.B) {
	mux := setupServeMux()
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_ParamRoute(b *testing.B) {
	mux := setupServeMux()
	req := httptest.NewRequest("GET", "/api/v1/users/42", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_NestedParamRoute(b *testing.B) {
	mux := setupServeMux()
	req := httptest.NewRequest("GET", "/api/v1/users/42/posts/99", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_CatchAllRoute(b *testing.B) {
	mux := setupServeMux()
	req := httptest.NewRequest("GET", "/files/static/assets/logo.png", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_Mixed12Routes(b *testing.B) {
	mux := setupServeMux()
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/health", nil),
		httptest.NewRequest("GET", "/api/v1/users/1", nil),
		httptest.NewRequest("GET", "/api/v1/users/1/posts/2", nil),
		httptest.NewRequest("GET", "/files/a/b/c.png", nil),
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, reqs[i%len(reqs)])
	}
}

// ============================================================================
// After: dim Router with hybrid static map + radix tree
// ============================================================================

func setupRouter() *Router {
	r := NewRouter()
	for _, rt := range benchRoutes {
		r.Register(rt.method, rt.pattern, benchHandler, nil)
	}
	r.Build()
	return r
}

func BenchmarkAfter_StaticRoute(b *testing.B) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_ParamRoute(b *testing.B) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/api/v1/users/42", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_NestedParamRoute(b *testing.B) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/api/v1/users/42/posts/99", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_CatchAllRoute(b *testing.B) {
	r := setupRouter()
	req := httptest.NewRequest("GET", "/files/static/assets/logo.png", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_Mixed12Routes(b *testing.B) {
	r := setupRouter()
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/health", nil),
		httptest.NewRequest("GET", "/api/v1/users/1", nil),
		httptest.NewRequest("GET", "/api/v1/users/1/posts/2", nil),
		httptest.NewRequest("GET", "/files/a/b/c.png", nil),
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, reqs[i%len(reqs)])
	}
}
