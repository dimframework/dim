package dim

import (
	"fmt"
	"strings"
)

// RouteListCommand menampilkan semua route yang terdaftar beserta handler dan middleware.
type RouteListCommand struct{}

func (c *RouteListCommand) Name() string {
	return "route:list"
}

func (c *RouteListCommand) Description() string {
	return "Display all registered routes"
}

func (c *RouteListCommand) Execute(ctx *CommandContext) error {
	if ctx.Router == nil {
		return fmt.Errorf("router is required")
	}

	routes := ctx.Router.GetRoutes()

	if len(routes) == 0 {
		fmt.Println("No routes registered")
		return nil
	}

	// Count stripped handlers for warning
	strippedCount := 0
	for _, route := range routes {
		if route.Handler == "<stripped>" || strings.Contains(route.Handler, "<stripped>") {
			strippedCount++
		}
	}

	fmt.Printf("Registered Routes (%d total):\n\n", len(routes))

	for _, route := range routes {
		// Format: METHOD  PATH  -> Handler  [Middleware1, Middleware2]
		middlewareStr := ""
		if len(route.Middlewares) > 0 {
			middlewareStr = fmt.Sprintf(" [%s]", strings.Join(route.Middlewares, ", "))
		}

		fmt.Printf("%-7s %-35s -> %-45s%s\n",
			route.Method,
			route.Path,
			route.Handler,
			middlewareStr,
		)
	}

	// Display warning if binary is stripped
	if strippedCount > 0 {
		fmt.Println()
		fmt.Printf("⚠ Warning: %d route(s) show <stripped> handlers.\n", strippedCount)
		fmt.Println("This happens when the binary is compiled with -ldflags=\"-s -w\"")
		fmt.Println("To see handler names, compile without stripping debug symbols.")
	}

	return nil
}
