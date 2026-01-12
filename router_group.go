package dim

import "log/slog"

// RouterGroup represents a group of routes with a common prefix
type RouterGroup struct {
	router     *Router
	prefix     string
	middleware []MiddlewareFunc
}

// combineMiddleware safely combines group middleware with route-specific middleware
// Order: group middleware first, then route middleware (so group is outermost with fixed Chain)
func (rg *RouterGroup) combineMiddleware(middleware ...MiddlewareFunc) []MiddlewareFunc {
	combined := make([]MiddlewareFunc, 0, len(rg.middleware)+len(middleware))
	combined = append(combined, rg.middleware...)
	combined = append(combined, middleware...)
	return combined
}

// Get mendaftarkan route GET dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
// Contoh: group prefix="/api", route "/users" menjadi "/api/users".
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api")
//	api.Get("/users", getUsersHandler)  // registered as GET /api/users
func (rg *RouterGroup) Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Get(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Post mendaftarkan route POST dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api", AuthMiddleware)
//	api.Post("/users", createUserHandler)  // registered as POST /api/users with AuthMiddleware
func (rg *RouterGroup) Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Post(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Put mendaftarkan route PUT dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api")
//	api.Put("/users/{id}", updateUserHandler)  // registered as PUT /api/users/{id}
func (rg *RouterGroup) Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Put(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Delete mendaftarkan route DELETE dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api")
//	api.Delete("/users/{id}", deleteUserHandler)  // registered as DELETE /api/users/{id}
func (rg *RouterGroup) Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Delete(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Patch mendaftarkan route PATCH dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api")
//	api.Patch("/users/{id}", patchUserHandler)  // registered as PATCH /api/users/{id}
func (rg *RouterGroup) Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Patch(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Options mendaftarkan route OPTIONS dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api")
//	api.Options("/users", optionsHandler)  // registered as OPTIONS /api/users
func (rg *RouterGroup) Options(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Options(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Head mendaftarkan route HEAD dalam group dengan group prefix dan middleware.
// Group prefix dan middleware otomatis di-prepend ke route.
//
// Parameters:
//   - path: URL path relative to group prefix
//   - handler: HandlerFunc yang akan menangani request
//   - middleware: optional route-specific middleware
//
// Example:
//
//	api := router.Group("/api")
//	api.Head("/users", headHandler)  // registered as HEAD /api/users
func (rg *RouterGroup) Head(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Head(rg.prefix+path, handler, rg.combineMiddleware(middleware...)...)
}

// Group membuat nested route group dengan combined prefix dan middleware.
// Prefix dan middleware dari parent group di-combine dengan new group.
// Berguna untuk hierarchical route organization (contoh: /api/v1/admin).
//
// Parameters:
//   - prefix: sub-prefix untuk nested group
//   - middleware: optional group-level middleware
//
// Returns:
//   - *RouterGroup: nested router group instance
//
// Example:
//
//	api := router.Group("/api")
//	v1 := api.Group("/v1", AuthMiddleware)
//	admin := v1.Group("/admin", AdminAuthMiddleware)
//	admin.Get("/users", listAllUsersHandler)  // registered as GET /api/v1/admin/users with combined middleware
func (rg *RouterGroup) Group(prefix string, middleware ...MiddlewareFunc) *RouterGroup {
	// Validate and normalize prefix
	if prefix != "" && prefix[0] != '/' {
		slog.Warn("router group prefix should start with /",
			"prefix", prefix,
			"note", "prefix will be used as-is, consider using /"+prefix)
	}

	return &RouterGroup{
		router:     rg.router,
		prefix:     rg.prefix + prefix,
		middleware: rg.combineMiddleware(middleware...),
	}
}
