package dim

import "net/http"

// HandlerFunc is the standard HTTP handler function signature
type HandlerFunc func(http.ResponseWriter, *http.Request)

// MiddlewareFunc is a function that wraps a handler with middleware
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Chain membungkus handler dengan multiple middleware functions secara beruntun.
// Middleware diterapkan dalam urutan maju (pertama di list dijalankan pertama).
// Contoh: Chain(handler, m1, m2, m3) menghasilkan execution order: m1 -> m2 -> m3 -> handler.
// Gunakan untuk menerapkan middleware chain ke single handler.
//
// Parameters:
//   - handler: HandlerFunc yang akan dibungkus dengan middleware
//   - middleware: variadic list dari MiddlewareFunc yang diterapkan berurutan
//
// Returns:
//   - HandlerFunc: handler baru dengan middleware chain diterapkan
//
// Example:
//
//	finalHandler := Chain(myHandler, LoggerMiddleware, AuthMiddleware, RecoveryMiddleware)
func Chain(handler HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	// Apply middleware in reverse order so the first one in the slice is the outermost
	// This way m1, m2, m3 results in m1(m2(m3(handler)))
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// ChainMiddleware membuat MiddlewareFunc dari multiple middleware tanpa final handler.
// Berguna untuk membuat reusable middleware chain yang bisa diterapkan ke multiple routes.
// Return value adalah MiddlewareFunc yang bisa digunakan di route registration.
//
// Parameters:
//   - middleware: variadic list dari MiddlewareFunc yang akan di-chain
//
// Returns:
//   - MiddlewareFunc: middleware function yang combine semua middleware
//
// Example:
//
//	authChain := ChainMiddleware(AuthMiddleware, LoggerMiddleware)
//	router.Get("/users", getUsersHandler, authChain)
func ChainMiddleware(middleware ...MiddlewareFunc) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return Chain(next, middleware...)
	}
}

// Compose membuat middleware baru dari multiple middleware functions.
// Similar dengan Chain tetapi return MiddlewareFunc dan dapat digunakan sebagai single middleware.
// Berguna untuk membuat composite middleware yang dapat di-reuse di berbagai places.
//
// Parameters:
//   - middleware: variadic list dari MiddlewareFunc yang akan di-compose
//
// Returns:
//   - MiddlewareFunc: composed middleware function
//
// Example:
//
//	authAndLog := Compose(AuthMiddleware, LoggerMiddleware)
//	router.Get("/protected", handler, authAndLog)
func Compose(middleware ...MiddlewareFunc) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		return next
	}
}

// HandlerToMiddleware mengkonversi http.Handler menjadi MiddlewareFunc.
// Memungkinkan penggunaan standard http.Handler implementations sebagai middleware.
// Berguna untuk integrasi dengan third-party packages yang implement http.Handler.
//
// Parameters:
//   - h: http.Handler yang akan dikonversi menjadi middleware
//
// Returns:
//   - MiddlewareFunc: middleware function yang membungkus http.Handler
//
// Example:
//
//	corsHandler := HandlerToMiddleware(corsStandardHandler)
//	router.Use(corsHandler)
func HandlerToMiddleware(h http.Handler) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
}

// MiddlewareToHandler mengkonversi MiddlewareFunc menjadi http.Handler.
// Berguna untuk menggunakan middleware dengan standard http patterns dan packages.
// Wrapper next parameter memungkinkan middleware untuk call next handler jika diperlukan.
//
// Parameters:
//   - m: MiddlewareFunc yang akan dikonversi
//   - next: http.Handler yang akan dijalankan setelah middleware
//
// Returns:
//   - http.Handler: handler yang mengimplementasikan http.Handler interface
//
// Example:
//
//	handler := MiddlewareToHandler(LoggerMiddleware, http.DefaultServeMux)
//	http.ListenAndServe(":8080", handler)
func MiddlewareToHandler(m MiddlewareFunc, next http.Handler) http.Handler {
	return m(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}).ToHandler()
}

// ToHandler mengkonversi HandlerFunc menjadi http.Handler interface.
// Memungkinkan HandlerFunc digunakan di mana saja yang expect http.Handler.
// Conversion ini implicit di many Go standard library functions.
//
// Returns:
//   - http.Handler: http.Handler yang membungkus HandlerFunc
//
// Example:
//
//	handler := myHandlerFunc.ToHandler()
//	http.Handle("/path", handler)
func (h HandlerFunc) ToHandler() http.Handler {
	return http.HandlerFunc(h)
}

// ServeHTTP mengimplementasikan http.Handler interface untuk HandlerFunc.
// Memungkinkan HandlerFunc digunakan di mana http.Handler diexpect.
// Ini membuatnya transparent untuk menggunakan HandlerFunc dengan standard http patterns.
//
// Parameters:
//   - w: http.ResponseWriter untuk menulis response
//   - r: *http.Request request yang diproses
//
// Example:
//
//	var h HandlerFunc = myHandler
//	h.ServeHTTP(w, r)  // calls h(w, r)
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(w, r)
}
