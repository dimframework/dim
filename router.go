package dim

import (
	"net/http"
	"strings"
	"sync"
)

// Router adalah router HTTP utama yang dibangun di atas stdlib http.ServeMux dengan dukungan middleware yang ditingkatkan.
type Router struct {
	mux           *http.ServeMux
	middleware    []MiddlewareFunc
	cachedHandler http.Handler
	initialized   bool
	lock          sync.RWMutex
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
