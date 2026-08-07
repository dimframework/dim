package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler menulis 200 beserta penanda agar route yang terpanggil bisa dibedakan.
func okHandler(marker string) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(marker))
	}
}

func doRequest(t *testing.T, router *Router, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

// TestTreeParamRouteDoesNotServeDeeperPaths memastikan route berparameter tidak
// melayani path yang lebih dalam dari pola yang didaftarkan (issue #15).
func TestTreeParamRouteDoesNotServeDeeperPaths(t *testing.T) {
	router := NewRouter()
	router.Get("/m/{slug}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("home:" + Of(w, r).Param("slug")))
	})
	router.Build()

	tests := []struct {
		path     string
		wantCode int
		wantBody string
	}{
		{"/m/abc", http.StatusOK, "home:abc"},
		{"/m/abc/x", http.StatusNotFound, ""},
		{"/m/abc/x/y/z", http.StatusNotFound, ""},
		{"/m/abc/", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != tt.wantCode {
			t.Errorf("%s: status = %d, want %d", tt.path, w.Code, tt.wantCode)
		}
		if tt.wantBody != "" && w.Body.String() != tt.wantBody {
			t.Errorf("%s: body = %q, want %q", tt.path, w.Body.String(), tt.wantBody)
		}
	}
}

// TestTreeStaticSegmentAfterParamDoesNotServeDeeperPaths memastikan segmen statis
// di belakang parameter juga menolak sisa path (issue #15).
func TestTreeStaticSegmentAfterParamDoesNotServeDeeperPaths(t *testing.T) {
	router := NewRouter()
	router.Get("/campaign/{slug}/donate", okHandler("donate"))
	router.Build()

	tests := []struct {
		path     string
		wantCode int
	}{
		{"/campaign/x/donate", http.StatusOK},
		{"/campaign/x/donate/lagi", http.StatusNotFound},
		{"/campaign/x/donate/", http.StatusNotFound},
		{"/campaign/x", http.StatusNotFound},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != tt.wantCode {
			t.Errorf("%s: status = %d, want %d", tt.path, w.Code, tt.wantCode)
		}
	}
}

// TestTreeStaticRouteIsNotAffected memastikan route tanpa parameter (peta statis)
// tetap berperilaku sama.
func TestTreeStaticRouteIsNotAffected(t *testing.T) {
	router := NewRouter()
	router.Get("/halo", okHandler("halo"))
	router.Build()

	if w := doRequest(t, router, http.MethodGet, "/halo"); w.Code != http.StatusOK {
		t.Errorf("/halo: status = %d, want 200", w.Code)
	}
	if w := doRequest(t, router, http.MethodGet, "/halo/x"); w.Code != http.StatusNotFound {
		t.Errorf("/halo/x: status = %d, want 404", w.Code)
	}
}

// TestTreeCatchAllStillCapturesRemainder memastikan perbaikan tidak mematikan
// route catch-all.
func TestTreeCatchAllStillCapturesRemainder(t *testing.T) {
	router := NewRouter()
	router.Get("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(GetParam(r, "path")))
	})
	router.Build()

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/files/doc.pdf", "doc.pdf"},
		{"/files/a/b/c.txt", "a/b/c.txt"},
		{"/files/", ""},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tt.path, w.Code)
			continue
		}
		if w.Body.String() != tt.wantBody {
			t.Errorf("%s: param = %q, want %q", tt.path, w.Body.String(), tt.wantBody)
		}
	}
}

// TestTreeMethodNotAllowedOnParamRoute memastikan 405 tetap dikembalikan saat
// path cocok tetapi metodenya tidak terdaftar.
func TestTreeMethodNotAllowedOnParamRoute(t *testing.T) {
	router := NewRouter()
	router.Get("/users/{id}", okHandler("get"))
	router.Delete("/users/{id}", okHandler("delete"))
	router.Build()

	w := doRequest(t, router, http.MethodPost, "/users/7")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /users/7: status = %d, want 405", w.Code)
	}
	if got := w.Header().Get("Allow"); got != "DELETE, GET" {
		t.Errorf("Allow header = %q, want %q", got, "DELETE, GET")
	}
}

// TestTreeDeeperPathIsNotMethodNotAllowed memastikan path yang lebih dalam
// menghasilkan 404, bukan 405 dari endpoint milik node induk.
func TestTreeDeeperPathIsNotMethodNotAllowed(t *testing.T) {
	router := NewRouter()
	router.Get("/users/{id}", okHandler("get"))
	router.Build()

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := doRequest(t, router, method, "/users/7/profile")
		if w.Code != http.StatusNotFound {
			t.Errorf("%s /users/7/profile: status = %d, want 404", method, w.Code)
		}
	}
}

// TestTreeMethodNotAllowedDoesNotBlockSiblingBranch memastikan cabang yang
// hanya cocok sebagian (405) tidak menghentikan penelusuran cabang lain yang
// benar-benar punya handler untuk metode tersebut.
func TestTreeMethodNotAllowedDoesNotBlockSiblingBranch(t *testing.T) {
	router := NewRouter()
	router.Post("/user/new/{id}", okHandler("post-new"))
	router.Get("/user/{name}/{id}", okHandler("get-name"))
	router.Build()

	w := doRequest(t, router, http.MethodGet, "/user/new/5")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /user/new/5: status = %d, want 200", w.Code)
	}
	if w.Body.String() != "get-name" {
		t.Errorf("GET /user/new/5: body = %q, want %q", w.Body.String(), "get-name")
	}

	// Metode yang tidak terdaftar di cabang mana pun tetap 405.
	w = doRequest(t, router, http.MethodDelete, "/user/new/5")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /user/new/5: status = %d, want 405", w.Code)
	}
}

// TestTreeAllowHeaderListsMethodsFromAllBranches memastikan header Allow memuat
// gabungan metode dari seluruh cabang yang path-nya cocok, bukan hanya cabang
// pertama (RFC 7231 §7.4.1).
func TestTreeAllowHeaderListsMethodsFromAllBranches(t *testing.T) {
	router := NewRouter()
	router.Post("/user/new/{id}", okHandler("post-new"))
	router.Get("/user/{name}/{id}", okHandler("get-name"))
	router.Build()

	w := doRequest(t, router, http.MethodDelete, "/user/new/5")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /user/new/5: status = %d, want 405", w.Code)
	}
	if got := w.Header().Get("Allow"); got != "GET, POST" {
		t.Errorf("Allow header = %q, want %q", got, "GET, POST")
	}
}

// TestTreeTriesEveryParamChild memastikan seluruh anak parameter pada satu level
// ikut dicoba. Nama parameter yang berbeda membuat node terpisah, dan sebelumnya
// hanya node pertama yang pernah dijangkau sehingga route lain tak terjangkau.
func TestTreeTriesEveryParamChild(t *testing.T) {
	router := NewRouter()
	router.Get("/a/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("id=" + GetParam(r, "id")))
	})
	router.Get("/a/{slug}/edit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slug=" + GetParam(r, "slug") + "|id=" + GetParam(r, "id")))
	})
	router.Build()

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/a/5", "id=5"},
		{"/a/5/edit", "slug=5|id="},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tt.path, w.Code)
			continue
		}
		if w.Body.String() != tt.wantBody {
			t.Errorf("%s: body = %q, want %q", tt.path, w.Body.String(), tt.wantBody)
		}
	}
}

// TestTreeBacktracksToParamBranch memastikan cabang statis yang gagal tetap
// mundur ke cabang parameter.
func TestTreeBacktracksToParamBranch(t *testing.T) {
	router := NewRouter()
	router.Get("/user/new/{id}", okHandler("static-branch"))
	router.Get("/user/{name}/edit", okHandler("param-branch"))
	router.Build()

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/user/new/9", "static-branch"},
		{"/user/bob/edit", "param-branch"},
		{"/user/new/edit", "static-branch"},
	}

	for _, tt := range tests {
		w := doRequest(t, router, http.MethodGet, tt.path)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tt.path, w.Code)
			continue
		}
		if w.Body.String() != tt.wantBody {
			t.Errorf("%s: body = %q, want %q", tt.path, w.Body.String(), tt.wantBody)
		}
	}

	if w := doRequest(t, router, http.MethodGet, "/user/new/edit/extra"); w.Code != http.StatusNotFound {
		t.Errorf("/user/new/edit/extra: status = %d, want 404", w.Code)
	}
}

// TestTreeParamDoesNotLeakOnFailedMatch memastikan parameter dari cabang yang
// gagal (termasuk cabang yang berakhir 405) tidak ikut terbawa ke handler yang
// akhirnya cocok.
func TestTreeParamDoesNotLeakOnFailedMatch(t *testing.T) {
	router := NewRouter()
	router.Post("/a/{x}/b/{y}", okHandler("post"))
	router.Get("/a/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(GetParam(r, "rest") + "|x=" + GetParam(r, "x") + "|y=" + GetParam(r, "y")))
	})
	router.Build()

	w := doRequest(t, router, http.MethodGet, "/a/p/b/q")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /a/p/b/q: status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "p/b/q|x=|y=" {
		t.Errorf("GET /a/p/b/q: params = %q, want %q", got, "p/b/q|x=|y=")
	}
}
