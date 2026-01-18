package dim

import (
	"context"
	"io/fs"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/atfromhome/goreus/pkg/cache"
)

// RouteInfo menyimpan informasi metadata tentang route yang terdaftar.
// Digunakan untuk introspeksi route (route:list command).
type RouteInfo struct {
	Method      string   // HTTP method (GET, POST, dll)
	Path        string   // URL path pattern
	Handler     string   // Nama handler function
	Middlewares []string // Daftar nama middleware yang diterapkan
}

// Router adalah router HTTP utama yang dibangun di atas stdlib http.ServeMux dengan dukungan middleware yang ditingkatkan.
type Router struct {
	mux           *http.ServeMux
	middleware    []MiddlewareFunc
	cachedHandler http.Handler
	initialized   bool
	lock          sync.RWMutex
	routes        []RouteInfo                               // Semua route yang terdaftar
	routeCache    *cache.InMemoryCache[string, []RouteInfo] // Cache untuk GetRoutes()
}

// NewRouter membuat instance router baru menggunakan stdlib http.ServeMux.
// Router mendukung pencocokan pola Go 1.22+ modern:
//   - Route statis: /users
//   - Parameter path: /users/{id}
//   - Catch-all: /files/{path...}
//   - Routing metode: GET /users/{id}
//
// Mengembalikan:
//   - *Router: instance router yang siap digunakan
//
// Contoh:
//
//	router := NewRouter()
//	router.Get("/users/{id}", getUserHandler)
//	http.ListenAndServe(":8080", router)
func NewRouter() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

// Use menambahkan middleware global yang akan diterapkan ke semua route.
// Middleware diterapkan dalam urutan penambahan dan sebelum middleware spesifik route.
// Thread-safe: dilindungi dengan mutex untuk akses konkuren.
//
// Parameter:
//   - middleware: daftar variadic dari MiddlewareFunc yang akan ditambahkan
//
// Contoh:
//
//	router.Use(RecoveryMiddleware, LoggerMiddleware)
func (r *Router) Use(middleware ...MiddlewareFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.middleware = append(r.middleware, middleware...)
	// Invalidate cached handler
	r.cachedHandler = nil
	r.initialized = false
}

// Build membuild handler chain secara eksplisit.
// Disarankan dipanggil di main() sebelum http.ListenAndServe untuk performa terbaik (menghindari locking saat request).
// Jika tidak dipanggil, handler akan dibangun secara lazy pada request pertama (dengan sedikit overhead locking).
func (r *Router) Build() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cachedHandler = r.buildHandler()
	r.initialized = true

	// Cache routes saat build
	if r.routeCache == nil {
		r.routeCache = cache.NewInMemoryCache[string, []RouteInfo](10, 5*time.Minute)
	}
	routesCopy := make([]RouteInfo, len(r.routes))
	copy(routesCopy, r.routes)
	r.routeCache.Set(context.Background(), "all_routes", routesCopy)
}

// Get mendaftarkan route GET dengan middleware spesifik route opsional.
// Path menggunakan pencocokan pola stdlib:
//   - Statis: /users
//   - Parameter: /users/{id}
//   - Catch-all: /files/{path...}
//
// Parameter:
//   - path: path URL untuk route (contoh: /users, /users/{id}, /files/{path...})
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional yang diterapkan sebelum handler
//
// Contoh:
//
//	router.Get("/users", getUsersHandler)
//	router.Get("/users/{id}", getUserHandler, AuthMiddleware)
func (r *Router) Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("GET", path, handler, middleware)
}

// Post mendaftarkan route POST dengan middleware spesifik route opsional.
// Path menggunakan pencocokan pola stdlib.
//
// Parameter:
//   - path: path URL untuk route
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Post("/users", createUserHandler)
//	router.Post("/upload", uploadFileHandler, AuthMiddleware)
func (r *Router) Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("POST", path, handler, middleware)
}

// Put mendaftarkan route PUT dengan middleware spesifik route opsional.
//
// Parameter:
//   - path: path URL untuk route
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Put("/users/{id}", updateUserHandler, AuthMiddleware)
func (r *Router) Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("PUT", path, handler, middleware)
}

// Delete mendaftarkan route DELETE dengan middleware spesifik route opsional.
//
// Parameter:
//   - path: path URL untuk route
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Delete("/users/{id}", deleteUserHandler, AuthMiddleware)
func (r *Router) Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("DELETE", path, handler, middleware)
}

// Patch mendaftarkan route PATCH dengan middleware spesifik route opsional.
//
// Parameter:
//   - path: path URL untuk route
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Patch("/users/{id}", patchUserHandler, AuthMiddleware)
func (r *Router) Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("PATCH", path, handler, middleware)
}

// Options mendaftarkan route OPTIONS dengan middleware spesifik route opsional.
//
// Parameter:
//   - path: path URL untuk route
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Options("/users", optionsHandler)
func (r *Router) Options(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("OPTIONS", path, handler, middleware)
}

// Head mendaftarkan route HEAD dengan middleware spesifik route opsional.
//
// Parameter:
//   - path: path URL untuk route
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Head("/users", headHandler)
func (r *Router) Head(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("HEAD", path, handler, middleware)
}

// Static melayani file statis dari sistem file (lokal atau embed).
// Secara otomatis menambahkan header keamanan dasar.
//
// Parameter:
//   - prefix: path URL prefix (contoh: "/assets/")
//   - root: fs.FS interface (gunakan os.DirFS("./public") atau embed.FS)
//   - middleware: middleware tambahan (opsional)
func (r *Router) Static(prefix string, root fs.FS, middleware ...MiddlewareFunc) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Base handler
	fsServer := http.FileServer(http.FS(root))
	handler := http.StripPrefix(prefix, fsServer)

	// Default security & caching logic for static assets
	finalHandler := func(w http.ResponseWriter, req *http.Request) {
		// Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Catatan: Caching strategy untuk static assets biasanya tergantung pada
		// apakah file memiliki hash di namanya. Kita biarkan default browser/server
		// atau bisa diatur via middleware tambahan.

		handler.ServeHTTP(w, req)
	}

	// Wrap with optional middleware
	var h http.Handler = http.HandlerFunc(finalHandler)
	if len(middleware) > 0 {
		h = Chain(finalHandler, middleware...)
	}

	r.mux.Handle("GET "+prefix, h)
}

// SPA (Single Page Application) melayani aplikasi frontend modern dengan fallback ke index.html.
// Secara otomatis menambahkan header keamanan dan mematikan cache untuk file index agar user selalu mendapat versi terbaru.
func (r *Router) SPA(root fs.FS, index string, middleware ...MiddlewareFunc) {
	baseHandler := func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			path = index
		}

		// Coba buka file
		f, err := root.Open(path)

		// Jika file tidak ada atau request adalah direktori, sajikan index.html
		isDir := false
		if err == nil {
			stat, _ := f.Stat()
			isDir = stat.IsDir()
			f.Close()
		}

		if err != nil || isDir {
			// SPA Fallback: Sajikan index.html
			indexContent, errRead := fs.ReadFile(root, index)
			if errRead != nil {
				http.Error(w, "SPA Index Not Found", http.StatusInternalServerError)
				return
			}

			// Security & Anti-Cache Headers untuk index.html
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			w.Write(indexContent)
			return
		}

		// Jika file statis biasa (js, css, png), sajikan normal
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.FileServer(http.FS(root)).ServeHTTP(w, req)
	}

	// Wrap with optional middleware
	var h http.Handler = http.HandlerFunc(baseHandler)
	if len(middleware) > 0 {
		h = Chain(baseHandler, middleware...)
	}

	r.mux.Handle("GET /{path...}", h)
}

// Group membuat RouterGroup baru dengan prefix dan middleware.
// RouterGroup memudahkan pengelompokan route dengan prefix dan middleware yang sama.
//
// Parameter:
//   - prefix: path prefix untuk semua route dalam grup
//   - middleware: middleware level grup opsional yang diterapkan ke semua route
//
// Mengembalikan:
//   - *RouterGroup: instance grup router untuk menambahkan route
//
// Contoh:
//
//	api := router.Group("/api", AuthMiddleware)
//	api.Get("/users", getUsersHandler)  // terdaftar sebagai GET /api/users
func (r *Router) Group(prefix string, middleware ...MiddlewareFunc) *RouterGroup {
	return &RouterGroup{
		router:     r,
		prefix:     prefix,
		middleware: middleware,
	}
}

// Register mendaftarkan route dengan metode HTTP, path, handler, dan middleware opsional.
// Menggunakan stdlib http.ServeMux untuk pencocokan pola.
// Thread-safe: dilindungi dengan mutex untuk pendaftaran route konkuren.
// Secara otomatis mengubah metode menjadi huruf besar untuk kepatuhan.
//
// Parameter:
//   - method: metode HTTP (GET, POST, PUT, DELETE, dll)
//   - path: path URL dengan pola stdlib (/users/{id}, /files/{path...})
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	router.Register("GET", "/users/{id}", getUserHandler, []MiddlewareFunc{AuthMiddleware})
func (r *Router) Register(method, path string, handler HandlerFunc, middleware []MiddlewareFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Ensure method is uppercase
	method = strings.ToUpper(method)

	// Combine method and path for stdlib mux
	pattern := method + " " + path

	// Wrap handler with route-specific middleware
	finalHandler := handler
	if len(middleware) > 0 {
		finalHandler = Chain(handler, middleware...)
	}

	// Register to stdlib mux
	r.mux.HandleFunc(pattern, finalHandler)

	// Track route info untuk CLI introspection
	handlerName := getFunctionName(handler)
	middlewareNames := make([]string, 0, len(middleware))
	for _, mw := range middleware {
		middlewareNames = append(middlewareNames, getFunctionName(mw))
	}

	r.routes = append(r.routes, RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     handlerName,
		Middlewares: middlewareNames,
	})

	// Invalidate cache
	if r.routeCache != nil {
		r.routeCache.Delete(context.Background(), "all_routes")
	}
}

// ServeHTTP mengimplementasikan antarmuka http.Handler untuk menangani permintaan HTTP.
// Menerapkan middleware global dan menggunakan cached handler untuk kinerja yang lebih baik.
//
// Parameter:
//   - w: http.ResponseWriter untuk menulis respons
//   - req: *http.Request permintaan yang akan diproses
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Jalur Cepat (Fast Path) - Tanpa Lock jika sudah initialized (Build dipanggil manual)
	if r.initialized {
		r.cachedHandler.ServeHTTP(w, req)
		return
	}

	// Optimistic read dengan RLock (Lazy load)
	r.lock.RLock()
	handler := r.cachedHandler
	r.lock.RUnlock()

	// Jika handler belum ada (cache miss), bangun ulang dengan Lock
	if handler == nil {
		r.lock.Lock()
		// Double-checked locking untuk memastikan tidak dibangun dua kali
		if r.cachedHandler == nil {
			r.cachedHandler = r.buildHandler()
		}
		handler = r.cachedHandler
		r.lock.Unlock()
	}

	handler.ServeHTTP(w, req)
}

// buildHandler membuat handler chain dengan middleware global.
func (r *Router) buildHandler() http.Handler {
	// Base handler: mux
	// Kita menggunakan stdlib mux sepenuhnya untuk routing, 404, dan 405.
	// Ini membuat implementasi lebih idiomatic dan sesuai dengan standar Go.

	base := HandlerFunc(r.mux.ServeHTTP)

	// Wrap dengan global middleware
	if len(r.middleware) > 0 {
		return Chain(base, r.middleware...)
	}
	return base
}

// GetRoutes mengembalikan semua route yang terdaftar dengan caching.
// Thread-safe dan menggunakan in-memory cache untuk performa optimal.
//
// Mengembalikan:
//   - []RouteInfo: copy dari semua route yang terdaftar
//
// Contoh:
//
//	routes := router.GetRoutes()
//	for _, route := range routes {
//	    fmt.Printf("%s %s -> %s\n", route.Method, route.Path, route.Handler)
//	}
func (r *Router) GetRoutes() []RouteInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()

	// Check cache first
	if r.routeCache != nil {
		if cached, found := r.routeCache.Get(context.Background(), "all_routes"); found {
			// Return a copy to prevent external modification
			cachedCopy := make([]RouteInfo, len(cached))
			copy(cachedCopy, cached)
			return cachedCopy
		}
	}

	// Initialize cache if nil
	if r.routeCache == nil {
		r.lock.RUnlock()
		r.lock.Lock()
		// Double-check after acquiring write lock
		if r.routeCache == nil {
			r.routeCache = cache.NewInMemoryCache[string, []RouteInfo](10, 5*time.Minute)
		}
		r.lock.Unlock()
		r.lock.RLock()
	}

	// Make copy to prevent external modification
	routesCopy := make([]RouteInfo, len(r.routes))
	copy(routesCopy, r.routes)

	// Store in cache
	r.routeCache.Set(context.Background(), "all_routes", routesCopy)

	return routesCopy
}

// getFunctionName mengekstrak nama function dari function pointer menggunakan reflection.
// Digunakan untuk mendapatkan nama handler dan middleware untuk route introspection.
//
// Note: Function names depend on debug symbols. If binary is stripped with -ldflags="-s -w",
// function names may not be available and will show as "<stripped>" or memory addresses.
func getFunctionName(fn interface{}) string {
	if fn == nil {
		return "<nil>"
	}

	// Get function value
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return "<invalid>"
	}

	// Get function pointer
	fnPtr := fnValue.Pointer()
	if fnPtr == 0 {
		return "<nil>"
	}

	// Get function info
	fnInfo := runtime.FuncForPC(fnPtr)
	if fnInfo == nil {
		return "<stripped>"
	}

	// Get full name
	fullName := fnInfo.Name()
	if fullName == "" {
		return "<stripped>"
	}

	// Extract the last part after the last slash (package path)
	// Format: github.com/user/repo/package.FunctionName
	lastSlash := strings.LastIndex(fullName, "/")
	if lastSlash >= 0 {
		fullName = fullName[lastSlash+1:]
	}

	// Clean up -fm suffix (func method)
	fullName = strings.TrimSuffix(fullName, "-fm")

	// Clean up pointer receiver syntax (*Type) -> Type
	fullName = strings.ReplaceAll(fullName, "(*", "")
	fullName = strings.ReplaceAll(fullName, ")", "")

	// Check if this is an anonymous function
	// Pattern: package.Type.MethodName.func1 or package.FunctionName.func1
	if strings.Contains(fullName, ".func") {
		// Extract the parent function/type name before .func
		parts := strings.Split(fullName, ".func")
		if len(parts) > 0 && parts[0] != "" {
			// Get the last meaningful part before .func
			nameParts := strings.Split(parts[0], ".")
			if len(nameParts) > 0 {
				parentName := nameParts[len(nameParts)-1]
				// Clean output: just show the parent function name for middleware factories
				return parentName
			}
		}
		return "<anonymous>"
	}

	return fullName
}
