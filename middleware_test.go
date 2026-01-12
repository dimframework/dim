package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain(t *testing.T) {
	var order []string

	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1_before")
			next(w, r)
			order = append(order, "m1_after")
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2_before")
			next(w, r)
			order = append(order, "m2_after")
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}

	chained := Chain(handler, middleware1, middleware2)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	chained(w, r)

	// Expected order: m1_before, m2_before, handler, m2_after, m1_after
	expected := []string{"m1_before", "m2_before", "handler", "m2_after", "m1_after"}
	if !equalSlice(order, expected) {
		t.Errorf("middleware execution order = %v, want %v", order, expected)
	}
}

func TestChainNoMiddleware(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
	}

	chained := Chain(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	chained(w, r)

	if !called {
		t.Errorf("handler not called")
	}
}

func TestChainMultipleMiddleware(t *testing.T) {
	var order []string

	m1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "1")
			next(w, r)
		}
	}

	m2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "2")
			next(w, r)
		}
	}

	m3 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "3")
			next(w, r)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "h")
	}

	chained := Chain(handler, m1, m2, m3)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	chained(w, r)

	// m1, m2, m3, handler
	expected := []string{"1", "2", "3", "h"}
	if !equalSlice(order, expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
}

func TestCompose(t *testing.T) {
	var order []string

	m1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1")
			next(w, r)
		}
	}

	m2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2")
			next(w, r)
		}
	}

	composed := Compose(m1, m2)
	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "h")
	}

	chained := composed(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	chained(w, r)

	expected := []string{"m1", "m2", "h"}
	if !equalSlice(order, expected) {
		t.Errorf("order = %v, want %v", order, expected)
	}
}

func TestHandlerFuncToHandler(t *testing.T) {
	called := false
	h := HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := h.ToHandler()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if !called {
		t.Errorf("handler not called")
	}
}

func TestHandlerFuncServeHTTP(t *testing.T) {
	called := false
	h := HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(w, r)

	if !called {
		t.Errorf("handler not called")
	}
}

func TestMiddlewareModifyRequest(t *testing.T) {
	middleware := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Add a header
			r.Header.Set("X-Modified", "true")
			next(w, r)
		}
	}

	var receivedHeader string
	handler := func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Modified")
	}

	chained := Chain(handler, middleware)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	chained(w, r)

	if receivedHeader != "true" {
		t.Errorf("middleware modification not received: got %s", receivedHeader)
	}
}

func TestMiddlewareModifyResponse(t *testing.T) {
	middleware := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Modified", "true")
			next(w, r)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	chained := Chain(handler, middleware)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	chained(w, r)

	if w.Header().Get("X-Modified") != "true" {
		t.Errorf("response header not set")
	}
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
