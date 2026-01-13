package dim

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// StartServer starts the HTTP server with graceful shutdown support.
// It listens on the specified port and serves requests using the provided handler.
// When a SIGINT or SIGTERM signal is received (or context cancelled), it will attempt to shut down
// the server gracefully.
//
// Parameters:
//   - ctx: context to control the server (e.g., from main)
//   - config: ServerConfig containing port and timeouts.
//   - handler: http.Handler to serve (usually the Router).
//
// Returns:
//   - error: error if server fails to start or shutdown error.
//
// Example:
//
//	ctx := context.Background()
//	config := dim.ServerConfig{Port: "8080"}
//	router := dim.NewRouter()
//	// ... register routes ...
//	if err := dim.StartServer(ctx, config, router); err != nil {
//	    log.Fatal(err)
//	}
func StartServer(ctx context.Context, config ServerConfig, handler http.Handler) error {
	addr := config.Port
	// Automatic port formatting if needed
	if addr == "" {
		addr = ":8080" // Default port
	} else if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	// Safety: Apply default timeouts if not set to prevent Slowloris attacks
	// Default: 10s for Read/Write, 2m for Idle, 10s for Shutdown
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 10 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 10 * time.Second
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	// Use net.Listen explicitly to confirm port binding before logging
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind port %s: %w", addr, err)
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking wait: either for a server error, a shutdown signal, or context cancellation
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		signal.Stop(shutdown)
		slog.Info("shutdown signal received", "signal", sig.String())

	case <-ctx.Done():
		slog.Info("context cancelled, shutting down server")
	}

	// Shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		// Just log the error, don't force Close() defensively unless critical
		// srv.Shutdown already closes the listener on context expiry/error
		return fmt.Errorf("shutdown error: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}
