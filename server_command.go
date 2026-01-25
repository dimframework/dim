package dim

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
)

// ServeCommand menjalankan HTTP server dengan konfigurasi yang dapat di-override via flags.
type ServeCommand struct {
	port string
}

func (c *ServeCommand) Name() string {
	return "serve"
}

func (c *ServeCommand) Description() string {
	return "Start HTTP server"
}

func (c *ServeCommand) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.port, "port", "", "Server port (default from config)")
}

func (c *ServeCommand) Execute(ctx *CommandContext) error {
	if ctx.Router == nil {
		return fmt.Errorf("router is required to start server")
	}

	if ctx.Config == nil {
		return fmt.Errorf("config is required to start server")
	}

	config := ctx.Config.Server

	// Override config with flags if provided
	if c.port != "" {
		config.Port = c.port
	}

	// Default port if not set
	if config.Port == "" {
		config.Port = "8080"
	}

	slog.Info("starting server", "port", config.Port)
	return StartServer(context.Background(), config, ctx.Router)
}
