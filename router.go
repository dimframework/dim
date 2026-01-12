package dim

import (
	"net/http"
	"sync"
)

// Router is the main HTTP router built on top of stdlib http.ServeMux with enhanced middleware support
type Router struct {
	mux        *http.ServeMux
	middleware []MiddlewareFunc
	notFound   http.Handler
	lock       sync.RWMutex
}

// NewRouter membuat instance router baru menggunakan stdlib http.ServeMux.
// Router mendukung pattern matching modern Go 1.22+:
//   - Static routes: /users
//   - Path parameters: /users/{id}
//   - Catch-all: /files/{path...}
//   - Method routing: GET /users/{id}
//
// Returns:
//   - *Router: router instance yang siap digunakan
//
// Example:
//
//	router := NewRouter()
//	router.Get("/users/{id}", getUserHandler)
//	http.ListenAndServe(":8080", router)
func NewRouter() *Router {
	return &Router{
		mux: http.NewServeMux(),
		notFound: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			JsonError(w, http.StatusNotFound, "Tidak ditemukan", nil)
		}),
	}
}

// Use menambahkan global middleware yang akan diterapkan ke semua routes.
// Middleware diterapkan dalam urutan yang ditambahkan dan sebelum route-specific middleware.
// Thread-safe: protected dengan mutex untuk concurrent access.
//
// Parameters:
//   - middleware: variadic list dari MiddlewareFunc yang akan ditambahkan
//
// Example:
//
//	router.Use(RecoveryMiddleware, LoggerMiddleware)
func (r *Router) Use(middleware ...MiddlewareFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// Get mendaftarkan route GET dengan optional route-specific middleware.
// Path menggunakan stdlib pattern matching:
//   - Static: /users
//   - Parameter: /users/{id}
//   - Catch-all: /files/{path...}
//
// Parameters:
//   - path: URL path untuk route (contoh: /users, /users/{id}, /files/{path...})
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware yang diterapkan sebelum handler
//
// Example:
//
//	router.Get("/users", getUsersHandler)
//	router.Get("/users/{id}", getUserHandler, AuthMiddleware)
func (r *Router) Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("GET", path, handler, middleware)
}

// Post mendaftarkan route POST dengan optional route-specific middleware.
// Path menggunakan stdlib pattern matching.
//
// Parameters:
//   - path: URL path untuk route
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Post("/users", createUserHandler)
//	router.Post("/upload", uploadFileHandler, AuthMiddleware)
func (r *Router) Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("POST", path, handler, middleware)
}

// Put mendaftarkan route PUT dengan optional route-specific middleware.
//
// Parameters:
//   - path: URL path untuk route
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Put("/users/{id}", updateUserHandler, AuthMiddleware)
func (r *Router) Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("PUT", path, handler, middleware)
}

// Delete mendaftarkan route DELETE dengan optional route-specific middleware.
//
// Parameters:
//   - path: URL path untuk route
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Delete("/users/{id}", deleteUserHandler, AuthMiddleware)
func (r *Router) Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("DELETE", path, handler, middleware)
}

// Patch mendaftarkan route PATCH dengan optional route-specific middleware.
//
// Parameters:
//   - path: URL path untuk route
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Patch("/users/{id}", patchUserHandler, AuthMiddleware)
func (r *Router) Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("PATCH", path, handler, middleware)
}

// Options mendaftarkan route OPTIONS dengan optional route-specific middleware.
//
// Parameters:
//   - path: URL path untuk route
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Options("/users", optionsHandler)
func (r *Router) Options(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("OPTIONS", path, handler, middleware)
}

// Head mendaftarkan route HEAD dengan optional route-specific middleware.
//
// Parameters:
//   - path: URL path untuk route
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Head("/users", headHandler)
func (r *Router) Head(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.Register("HEAD", path, handler, middleware)
}

// Group membuat RouterGroup baru dengan prefix dan middleware.
// RouterGroup memudahkan grouping routes dengan prefix dan middleware yang sama.
//
// Parameters:
//   - prefix: prefix path untuk semua routes dalam group
//   - middleware: optional group-level middleware yang diterapkan ke semua routes
//
// Returns:
//   - *RouterGroup: router group instance untuk menambahkan routes
//
// Example:
//
//	api := router.Group("/api", AuthMiddleware)
//	api.Get("/users", getUsersHandler)  // registered as GET /api/users
func (r *Router) Group(prefix string, middleware ...MiddlewareFunc) *RouterGroup {
	return &RouterGroup{
		router:     r,
		prefix:     prefix,
		middleware: middleware,
	}
}

// SetNotFound sets custom handler untuk 404 Not Found responses.
// Handler ini akan dipanggil ketika tidak ada route yang match dengan request.
// Internally registers a catch-all route with lowest priority.
// Thread-safe: protected dengan mutex untuk concurrent access.
//
// Parameters:
//   - handler: HandlerFunc untuk menangani 404 responses
//
// Example:
//
//	router.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
//	  JsonError(w, 404, "Halaman tidak ditemukan", nil)
//	})
func (r *Router) SetNotFound(handler HandlerFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.notFound = http.HandlerFunc(handler)
}

// Register mendaftarkan route dengan HTTP method, path, handler, dan optional middleware.
// Menggunakan stdlib http.ServeMux untuk pattern matching.
// Thread-safe: protected dengan mutex untuk concurrent route registration.
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, DELETE, dll)
//   - path: URL path dengan stdlib pattern (/users/{id}, /files/{path...})
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	router.Register("GET", "/users/{id}", getUserHandler, []MiddlewareFunc{AuthMiddleware})
func (r *Router) Register(method, path string, handler HandlerFunc, middleware []MiddlewareFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()

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

// ServeHTTP mengimplementasikan http.Handler interface untuk menangani HTTP requests.
// Applies global middleware, then delegates to stdlib http.ServeMux for routing.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - req: *http.Request request yang akan diproses
//
// Example:
//
//	http.ListenAndServe(":8080", router)  // router.ServeHTTP akan dipanggil otomatis
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.lock.RLock()
	globalMiddleware := r.middleware
	notFound := r.notFound
	r.lock.RUnlock()

	// Wrap mux.ServeHTTP as HandlerFunc with custom 404 check
	handler := func(w http.ResponseWriter, req *http.Request) {
		_, pattern := r.mux.Handler(req)
		if pattern == "" {
			if notFound != nil {
				notFound.ServeHTTP(w, req)
			} else {
				http.NotFound(w, req)
			}
			return
		}
		r.mux.ServeHTTP(w, req)
	}

	// Apply global middleware
	if len(globalMiddleware) > 0 {
		finalHandler := Chain(handler, globalMiddleware...)
		finalHandler(w, req)
	} else {
		handler(w, req)
	}
}
