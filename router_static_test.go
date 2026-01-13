package dim

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// TestRouter_Static menguji fungsi Static dengan os.DirFS (lokal) dan fstest.MapFS (memory/embed)
func TestRouter_Static(t *testing.T) {
	// 1. Setup Local FS
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "style.css")
	localContent := []byte("body { color: red; }")
	if err := os.WriteFile(testFile, localContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 2. Setup Mock FS (Simulasi Embed)
	mockFS := fstest.MapFS{
		"script.js": &fstest.MapFile{
			Data: []byte("console.log('hello');"),
		},
	}

	outer := NewRouter()

	// Register Local Dir
	outer.Static("/assets/", os.DirFS(tempDir))

	// Register Mock FS
	outer.Static("/static/", mockFS)

	outer.Build()

	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Local File",
			path:         "/assets/style.css",
			expectedCode: http.StatusOK,
			expectedBody: "body { color: red; }",
		},
		{
			name:         "Mock/Embed File",
			path:         "/static/script.js",
			expectedCode: http.StatusOK,
			expectedBody: "console.log('hello');",
		},
		{
			name:         "Not Found",
			path:         "/assets/missing.css",
			expectedCode: http.StatusNotFound,
			expectedBody: "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			outer.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}
			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestRouter_SPA menguji fungsi SPA dengan fallback ke index.html
func TestRouter_SPA(t *testing.T) {
	// Setup Mock FS
	// dist/
	//  ├── index.html
	//  └── app.js
	mockFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html>SPA Index</html>"),
		},
		"app.js": &fstest.MapFile{
			Data: []byte("console.log('spa');"),
		},
	}

	outer := NewRouter()

	// Register API route (untuk memastikan prioritas)
	outer.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Register SPA
	outer.SPA(mockFS, "index.html")

	outer.Build()

	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Existing Static File",
			path:         "/app.js",
			expectedCode: http.StatusOK,
			expectedBody: "console.log('spa');",
		},
		{
			name:         "API Route",
			path:         "/api/health",
			expectedCode: http.StatusOK,
			expectedBody: "OK",
		},
		{
			name:         "Frontend Route (Fallback)",
			path:         "/dashboard/user/1",
			expectedCode: http.StatusOK,
			expectedBody: "<html>SPA Index</html>",
		},
		{
			name:         "Root Route",
			path:         "/",
			expectedCode: http.StatusOK,
			expectedBody: "<html>SPA Index</html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			outer.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}
			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}
