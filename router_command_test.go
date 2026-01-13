package dim

import (
	"net/http"
	"testing"
)

func TestRouteListCommand_Name(t *testing.T) {
	cmd := &RouteListCommand{}
	if cmd.Name() != "route:list" {
		t.Errorf("Expected name 'route:list', got '%s'", cmd.Name())
	}
}

func TestRouteListCommand_Description(t *testing.T) {
	cmd := &RouteListCommand{}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestRouteListCommand_Execute_NoRouter(t *testing.T) {
	cmd := &RouteListCommand{}
	ctx := &CommandContext{
		Router: nil,
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when router is nil")
	}

	if err.Error() != "router is required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestRouteListCommand_Execute_EmptyRouter(t *testing.T) {
	cmd := &RouteListCommand{}
	router := NewRouter()

	ctx := &CommandContext{
		Router: router,
	}

	// Should not error with empty router
	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error with empty router: %v", err)
	}
}

func TestRouteListCommand_Execute_WithRoutes(t *testing.T) {
	cmd := &RouteListCommand{}
	router := NewRouter()

	// Register some test routes
	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {})
	router.Post("/users", func(w http.ResponseWriter, r *http.Request) {})
	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {})

	ctx := &CommandContext{
		Router: router,
	}

	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify routes are tracked
	routes := router.GetRoutes()
	if len(routes) != 3 {
		t.Errorf("Expected 3 routes to be tracked, got %d", len(routes))
	}
}

func TestRouteListCommand_Execute_RoutesWithMiddleware(t *testing.T) {
	cmd := &RouteListCommand{}
	router := NewRouter()

	// Mock middleware
	mockMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
		}
	}

	// Register route with middleware
	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {}, mockMiddleware)

	ctx := &CommandContext{
		Router: router,
	}

	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify middleware is tracked
	routes := router.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	if len(routes[0].Middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(routes[0].Middlewares))
	}
}
