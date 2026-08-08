package dim

import (
	"net/http"
	"testing"
	"testing/fstest"
)

func newRedirectRouter(t *testing.T) *Router {
	t.Helper()
	router := NewRouter()
	router.RedirectTrailingSlash(true)
	router.Get("/halo", okHandler("halo"))
	router.Get("/m/{slug}", okHandler("m"))
	router.Get("/p/{slug}/", okHandler("p"))
	router.Get("/f/{path...}", okHandler("f"))
	router.Post("/kirim", okHandler("kirim"))
	router.Build()
	return router
}

// TestRedirectTrailingSlashDisabledByDefault memastikan opsi ini opt-in dan
// tidak mengubah perilaku router yang sudah ada.
func TestRedirectTrailingSlashDisabledByDefault(t *testing.T) {
	router := NewRouter()
	router.Get("/halo", okHandler("halo"))
	router.Get("/m/{slug}", okHandler("m"))
	router.Build()

	for _, path := range []string{"/halo/", "/m/abc/"} {
		w := doRequest(t, router, http.MethodGet, path)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, w.Code)
		}
	}
}

// TestRedirectTrailingSlashStripsSlash menguji arah "slash berlebih dibuang",
// baik untuk route peta statis maupun route pohon radix.
func TestRedirectTrailingSlashStripsSlash(t *testing.T) {
	router := newRedirectRouter(t)

	tests := []struct {
		path     string
		wantLoc  string
		wantCode int
	}{
		{"/halo/", "/halo", http.StatusMovedPermanently},
		{"/m/abc/", "/m/abc", http.StatusMovedPermanently},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != tt.wantCode {
			t.Errorf("%s: status = %d, want %d", tt.path, w.Code, tt.wantCode)
		}
		if got := w.Header().Get("Location"); got != tt.wantLoc {
			t.Errorf("%s: Location = %q, want %q", tt.path, got, tt.wantLoc)
		}
	}
}

// TestRedirectTrailingSlashAddsSlash menguji arah sebaliknya — pola memang
// didaftarkan dengan slash di akhir.
func TestRedirectTrailingSlashAddsSlash(t *testing.T) {
	router := newRedirectRouter(t)

	tests := []struct{ path, wantLoc string }{
		{"/p/abc", "/p/abc/"},
		{"/f", "/f/"},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != http.StatusMovedPermanently {
			t.Errorf("%s: status = %d, want 301", tt.path, w.Code)
		}
		if got := w.Header().Get("Location"); got != tt.wantLoc {
			t.Errorf("%s: Location = %q, want %q", tt.path, got, tt.wantLoc)
		}
	}
}

// TestRedirectTrailingSlashPreservesQuery memastikan query string ikut terbawa,
// karena kehilangan query saat redirect memutus paginasi dan filter.
func TestRedirectTrailingSlashPreservesQuery(t *testing.T) {
	router := newRedirectRouter(t)

	w := doRequest(t, router, http.MethodGet, "/m/abc/?page=2&sort=-created_at")
	if got, want := w.Header().Get("Location"), "/m/abc?page=2&sort=-created_at"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

// TestRedirectTrailingSlashUses308ForNonIdempotentMethods memastikan metode
// selain GET/HEAD memakai 308 agar klien mempertahankan metode dan body.
// 301 membuat sebagian klien mengubah POST menjadi GET.
func TestRedirectTrailingSlashUses308ForNonIdempotentMethods(t *testing.T) {
	router := newRedirectRouter(t)

	w := doRequest(t, router, http.MethodPost, "/kirim/")
	if w.Code != http.StatusPermanentRedirect {
		t.Errorf("POST /kirim/: status = %d, want 308", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/kirim" {
		t.Errorf("POST /kirim/: Location = %q, want %q", got, "/kirim")
	}
}

// TestRedirectTrailingSlashOnlyWhenCounterpartHandles memastikan redirect tidak
// diarahkan ke path yang tetap tidak melayani metode tersebut.
func TestRedirectTrailingSlashOnlyWhenCounterpartHandles(t *testing.T) {
	router := newRedirectRouter(t)

	tests := []struct {
		method, path string
	}{
		{http.MethodPost, "/halo/"},     // /halo hanya GET
		{http.MethodGet, "/tidakada/"},  // tidak terdaftar sama sekali
		{http.MethodGet, "/m/abc/x/y/"}, // kedalaman path tetap tidak cocok
	}

	for _, tt := range tests {
		w := doRequest(t, router, tt.method, tt.path)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want 404", tt.method, tt.path, w.Code)
		}
	}
}

// TestRedirectTrailingSlashChecksTreeWhenStaticMapMisses memastikan padanan yang
// kebetulan ada di peta statis dengan metode lain tidak menghentikan pemeriksaan
// ke pohon radix.
func TestRedirectTrailingSlashChecksTreeWhenStaticMapMisses(t *testing.T) {
	router := NewRouter()
	router.RedirectTrailingSlash(true)
	router.Post("/users/", okHandler("post-static"))
	router.Get("/{resource}/", okHandler("get-tree"))
	router.Build()

	w := doRequest(t, router, http.MethodGet, "/users")
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("GET /users: status = %d, want 301", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/users/" {
		t.Errorf("Location = %q, want %q", got, "/users/")
	}
}

// TestRedirectTrailingSlashYieldsTo405 memastikan path yang memang terdaftar —
// hanya dengan metode lain — dijawab 405, bukan diredirect. Path itu bukan salah
// ketik yang perlu dikanonikalisasi.
func TestRedirectTrailingSlashYieldsTo405(t *testing.T) {
	router := NewRouter()
	router.RedirectTrailingSlash(true)
	router.Post("/users", okHandler("post"))
	router.Get("/users/", okHandler("get-slash"))
	router.Build()

	w := doRequest(t, router, http.MethodGet, "/users")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /users: status = %d, want 405", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("GET /users: Location = %q, want empty", loc)
	}
}

// TestRedirectTrailingSlashSafeUnderConcurrentToggle menjaga agar flag tetap
// dibaca lewat operasi atomik. Dijalankan dengan -race.
func TestRedirectTrailingSlashSafeUnderConcurrentToggle(t *testing.T) {
	router := newRedirectRouter(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			router.RedirectTrailingSlash(i%2 == 0)
		}
	}()
	for i := 0; i < 200; i++ {
		doRequest(t, router, http.MethodGet, "/halo/")
	}
	<-done
}

// TestRedirectTrailingSlashLeavesRootAlone memastikan "/" tidak pernah
// diredirect ke path kosong.
func TestRedirectTrailingSlashLeavesRootAlone(t *testing.T) {
	router := newRedirectRouter(t)

	w := doRequest(t, router, http.MethodGet, "/")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /: status = %d, want 404", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("GET /: Location = %q, want empty", loc)
	}
}

// TestRedirectTrailingSlashRunsBeforeSPAFallback memastikan redirect diperiksa
// sebelum fallback mux. Wildcard GET /{path...} milik SPA menangkap semua path
// yang tersisa, sehingga urutan ini menentukan.
func TestRedirectTrailingSlashRunsBeforeSPAFallback(t *testing.T) {
	router := NewRouter()
	router.RedirectTrailingSlash(true)
	router.Get("/api/user/{id}", okHandler("user"))
	router.SPA(fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html>SPA</html>")}}, "index.html")
	router.Build()

	w := doRequest(t, router, http.MethodGet, "/api/user/7/")
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("GET /api/user/7/: status = %d, want 301", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/api/user/7" {
		t.Errorf("Location = %q, want %q", got, "/api/user/7")
	}

	// Path yang tidak punya padanan tetap jatuh ke SPA.
	w = doRequest(t, router, http.MethodGet, "/dashboard/")
	if w.Code != http.StatusOK || w.Body.String() != "<html>SPA</html>" {
		t.Errorf("GET /dashboard/: status = %d body = %q, want 200 SPA index", w.Code, w.Body.String())
	}
}
