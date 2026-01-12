package dim

import (
	"log/slog"
	"path"
)

// RouterGroup merepresentasikan sekelompok route dengan prefix yang sama.
type RouterGroup struct {
	router     *Router
	prefix     string
	middleware []MiddlewareFunc
}

// combineMiddleware menggabungkan middleware grup dengan middleware spesifik route secara aman.
// Urutan: middleware grup terlebih dahulu, kemudian middleware route (sehingga grup berada di posisi terluar dengan Rantai tetap).
func (rg *RouterGroup) combineMiddleware(middleware ...MiddlewareFunc) []MiddlewareFunc {
	combined := make([]MiddlewareFunc, 0, len(rg.middleware)+len(middleware))
	combined = append(combined, rg.middleware...)
	combined = append(combined, middleware...)
	return combined
}

// Use menambahkan middleware ke dalam group yang sudah ada.
// Middleware akan diterapkan ke semua route yang didaftarkan SETELAH pemanggilan ini.
//
// Parameter:
//   - middleware: daftar variadic dari MiddlewareFunc
//
// Example:
//
//	api := router.Group("/api")
//	api.Use(AuthMiddleware)
//	api.Get("/profile", profileHandler) // AuthMiddleware diterapkan
func (rg *RouterGroup) Use(middleware ...MiddlewareFunc) {
	rg.middleware = append(rg.middleware, middleware...)
}

// calculateFullPath menghitung path lengkap route dengan aman.
func (rg *RouterGroup) calculateFullPath(relativePath string) string {
	fullPath := path.Join(rg.prefix, relativePath)
	if fullPath != "" && fullPath[0] != '/' {
		fullPath = "/" + fullPath
	}
	return fullPath
}

// Get mendaftarkan route GET dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
// Menggunakan path.Join untuk menggabungkan path secara aman.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api")
//	api.Get("/users", getUsersHandler)  // terdaftar sebagai GET /api/users
func (rg *RouterGroup) Get(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Get(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Post mendaftarkan route POST dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api", AuthMiddleware)
//	api.Post("/users", createUserHandler)  // terdaftar sebagai POST /api/users dengan AuthMiddleware
func (rg *RouterGroup) Post(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Post(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Put mendaftarkan route PUT dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api")
//	api.Put("/users/{id}", updateUserHandler)  // terdaftar sebagai PUT /api/users/{id}
func (rg *RouterGroup) Put(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Put(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Delete mendaftarkan route DELETE dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api")
//	api.Delete("/users/{id}", deleteUserHandler)  // terdaftar sebagai DELETE /api/users/{id}
func (rg *RouterGroup) Delete(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Delete(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Patch mendaftarkan route PATCH dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api")
//	api.Patch("/users/{id}", patchUserHandler)  // terdaftar sebagai PATCH /api/users/{id}
func (rg *RouterGroup) Patch(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Patch(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Options mendaftarkan route OPTIONS dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api")
//	api.Options("/users", optionsHandler)  // terdaftar sebagai OPTIONS /api/users
func (rg *RouterGroup) Options(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Options(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Head mendaftarkan route HEAD dalam grup dengan prefix grup dan middleware.
// Prefix grup dan middleware secara otomatis ditambahkan ke route.
//
// Parameter:
//   - path: path URL relatif terhadap prefix grup
//   - handler: HandlerFunc yang akan menangani permintaan
//   - middleware: middleware spesifik route opsional
//
// Contoh:
//
//	api := router.Group("/api")
//	api.Head("/users", headHandler)  // terdaftar sebagai HEAD /api/users
func (rg *RouterGroup) Head(relativePath string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	rg.router.Head(rg.calculateFullPath(relativePath), handler, rg.combineMiddleware(middleware...)...)
}

// Group membuat grup route bersarang dengan prefix dan middleware gabungan.
// Prefix dan middleware dari grup induk digabungkan dengan grup baru.
// Berguna untuk organisasi route hierarkis (contoh: /api/v1/admin).
// Menggunakan path.Join untuk memastikan format path yang benar (menghindari double slash).
//
// Parameter:
//   - prefix: sub-prefix untuk grup bersarang
//   - middleware: middleware level grup opsional
//
// Mengembalikan:
//   - *RouterGroup: instance grup router bersarang
//
// Example:
//
//	api := router.Group("/api")
//	v1 := api.Group("/v1", AuthMiddleware)
//	admin := v1.Group("/admin", AdminAuthMiddleware)
//	admin.Get("/users", listAllUsersHandler)  // terdaftar sebagai GET /api/v1/admin/users dengan middleware gabungan
func (rg *RouterGroup) Group(prefix string, middleware ...MiddlewareFunc) *RouterGroup {
	// Normalize path using path.Join to avoid // and ensure clean paths
	// We join existing prefix with new prefix
	newPrefix := path.Join(rg.prefix, prefix)

	// path.Join removes leading slash if the result is not root, but for routing we usually want absolute paths
	// unless the parent prefix was empty
	if newPrefix != "" && newPrefix[0] != '/' {
		newPrefix = "/" + newPrefix
	}

	// Logs warning if user provided prefix didn't look like a path segment,
	// but we've fixed it automatically above.
	if prefix != "" && prefix[0] != '/' {
		slog.Debug("normalizing router group prefix",
			"original", prefix,
			"normalized", newPrefix)
	}

	return &RouterGroup{
		router:     rg.router,
		prefix:     newPrefix,
		middleware: rg.combineMiddleware(middleware...),
	}
}
