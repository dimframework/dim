package dim

import (
	"fmt"
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

// ============================================================================
// Scale benchmarks: 100 and 500 routes
// ============================================================================

// resources defines API resource names used to generate large route sets.
// Each resource produces 5 routes (CRUD + nested sub-resource).
var resources = []string{
	"users", "posts", "comments", "tags", "categories",
	"products", "orders", "invoices", "payments", "addresses",
	"companies", "employees", "departments", "projects", "tasks",
	"tickets", "reports", "notifications", "messages", "attachments",
	"roles", "permissions", "sessions", "tokens", "logs",
	"events", "webhooks", "integrations", "configs", "secrets",
	"metrics", "alerts", "dashboards", "widgets", "charts",
	"exports", "imports", "jobs", "queues", "workers",
}

// generateRoutes returns n API route patterns spread across resource groups.
// Each resource yields 5 routes: list, create, get, update, delete.
// At n=100 → 20 resources × 5 routes; at n=500 → 40 resources × (5 routes × 2.5 versions).
func generateRoutes(n int) []struct{ method, pattern, reqPath string } {
	routes := make([]struct{ method, pattern, reqPath string }, 0, n)

	versions := []string{"v1", "v2", "v3"}
	methods := []struct{ method, suffix, reqSuffix string }{
		{"GET", "", ""},
		{"POST", "", ""},
		{"GET", "/{id}", "/42"},
		{"PUT", "/{id}", "/42"},
		{"DELETE", "/{id}", "/42"},
	}

	for _, ver := range versions {
		for _, res := range resources {
			base := fmt.Sprintf("/api/%s/%s", ver, res)
			reqBase := fmt.Sprintf("/api/%s/%s", ver, res)
			for _, m := range methods {
				routes = append(routes, struct{ method, pattern, reqPath string }{
					method:  m.method,
					pattern: base + m.suffix,
					reqPath: reqBase + m.reqSuffix,
				})
				if len(routes) >= n {
					return routes
				}
			}
		}
	}
	return routes
}

func setupServeMuxN(n int) (*http.ServeMux, []struct{ method, pattern, reqPath string }) {
	routes := generateRoutes(n)
	mux := http.NewServeMux()
	for _, r := range routes {
		mux.HandleFunc(r.method+" "+r.pattern, benchHandler)
	}
	return mux, routes
}

func setupRouterN(n int) (*Router, []struct{ method, pattern, reqPath string }) {
	routes := generateRoutes(n)
	r := NewRouter()
	for _, rt := range routes {
		r.Register(rt.method, rt.pattern, benchHandler, nil)
	}
	r.Build()
	return r, routes
}

// --- 100 routes ---

func BenchmarkBefore_100Routes_StaticRoute(b *testing.B) {
	mux, routes := setupServeMuxN(100)
	// Pick a static route near the end of the list (worst case for linear scan)
	req := httptest.NewRequest(routes[90].method, routes[90].reqPath, nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_100Routes_StaticRoute(b *testing.B) {
	r, routes := setupRouterN(100)
	req := httptest.NewRequest(routes[90].method, routes[90].reqPath, nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_100Routes_ParamRoute(b *testing.B) {
	mux, routes := setupServeMuxN(100)
	// Pick a dynamic route (has {id}) near the end
	var req *http.Request
	for i := len(routes) - 1; i >= 0; i-- {
		if routes[i].method == "GET" && routes[i].reqPath[len(routes[i].reqPath)-1] != 's' {
			req = httptest.NewRequest(routes[i].method, routes[i].reqPath, nil)
			break
		}
	}
	if req == nil {
		req = httptest.NewRequest("GET", routes[99].reqPath, nil)
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_100Routes_ParamRoute(b *testing.B) {
	r, routes := setupRouterN(100)
	var req *http.Request
	for i := len(routes) - 1; i >= 0; i-- {
		if routes[i].method == "GET" && routes[i].reqPath[len(routes[i].reqPath)-1] != 's' {
			req = httptest.NewRequest(routes[i].method, routes[i].reqPath, nil)
			break
		}
	}
	if req == nil {
		req = httptest.NewRequest("GET", routes[99].reqPath, nil)
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_100Routes_Mixed(b *testing.B) {
	mux, routes := setupServeMuxN(100)
	// Sample: first, middle, and last routes
	reqs := []*http.Request{
		httptest.NewRequest(routes[0].method, routes[0].reqPath, nil),
		httptest.NewRequest(routes[24].method, routes[24].reqPath, nil),
		httptest.NewRequest(routes[49].method, routes[49].reqPath, nil),
		httptest.NewRequest(routes[74].method, routes[74].reqPath, nil),
		httptest.NewRequest(routes[99].method, routes[99].reqPath, nil),
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, reqs[i%len(reqs)])
	}
}

func BenchmarkAfter_100Routes_Mixed(b *testing.B) {
	r, routes := setupRouterN(100)
	reqs := []*http.Request{
		httptest.NewRequest(routes[0].method, routes[0].reqPath, nil),
		httptest.NewRequest(routes[24].method, routes[24].reqPath, nil),
		httptest.NewRequest(routes[49].method, routes[49].reqPath, nil),
		httptest.NewRequest(routes[74].method, routes[74].reqPath, nil),
		httptest.NewRequest(routes[99].method, routes[99].reqPath, nil),
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, reqs[i%len(reqs)])
	}
}

// --- 500 routes ---

func BenchmarkBefore_500Routes_StaticRoute(b *testing.B) {
	mux, routes := setupServeMuxN(500)
	req := httptest.NewRequest(routes[490].method, routes[490].reqPath, nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_500Routes_StaticRoute(b *testing.B) {
	r, routes := setupRouterN(500)
	req := httptest.NewRequest(routes[490].method, routes[490].reqPath, nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_500Routes_ParamRoute(b *testing.B) {
	mux, routes := setupServeMuxN(500)
	var req *http.Request
	for i := len(routes) - 1; i >= 0; i-- {
		if routes[i].method == "GET" && routes[i].reqPath[len(routes[i].reqPath)-1] != 's' {
			req = httptest.NewRequest(routes[i].method, routes[i].reqPath, nil)
			break
		}
	}
	if req == nil {
		req = httptest.NewRequest("GET", routes[499].reqPath, nil)
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkAfter_500Routes_ParamRoute(b *testing.B) {
	r, routes := setupRouterN(500)
	var req *http.Request
	for i := len(routes) - 1; i >= 0; i-- {
		if routes[i].method == "GET" && routes[i].reqPath[len(routes[i].reqPath)-1] != 's' {
			req = httptest.NewRequest(routes[i].method, routes[i].reqPath, nil)
			break
		}
	}
	if req == nil {
		req = httptest.NewRequest("GET", routes[499].reqPath, nil)
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkBefore_500Routes_Mixed(b *testing.B) {
	mux, routes := setupServeMuxN(500)
	reqs := []*http.Request{
		httptest.NewRequest(routes[0].method, routes[0].reqPath, nil),
		httptest.NewRequest(routes[124].method, routes[124].reqPath, nil),
		httptest.NewRequest(routes[249].method, routes[249].reqPath, nil),
		httptest.NewRequest(routes[374].method, routes[374].reqPath, nil),
		httptest.NewRequest(routes[499].method, routes[499].reqPath, nil),
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, reqs[i%len(reqs)])
	}
}

func BenchmarkAfter_500Routes_Mixed(b *testing.B) {
	r, routes := setupRouterN(500)
	reqs := []*http.Request{
		httptest.NewRequest(routes[0].method, routes[0].reqPath, nil),
		httptest.NewRequest(routes[124].method, routes[124].reqPath, nil),
		httptest.NewRequest(routes[249].method, routes[249].reqPath, nil),
		httptest.NewRequest(routes[374].method, routes[374].reqPath, nil),
		httptest.NewRequest(routes[499].method, routes[499].reqPath, nil),
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, reqs[i%len(reqs)])
	}
}
