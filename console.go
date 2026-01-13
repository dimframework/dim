package dim

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Command adalah interface yang harus diimplementasikan oleh semua CLI commands.
// Setiap command mendefinisikan nama, deskripsi, dan logika eksekusi.
type Command interface {
	// Name mengembalikan nama command (contoh: "serve", "migrate", "route:list")
	Name() string

	// Description mengembalikan deskripsi singkat command untuk help text
	Description() string

	// Execute menjalankan logika command dengan context yang berisi dependencies
	Execute(ctx *CommandContext) error
}

// FlaggedCommand adalah interface opsional untuk command yang membutuhkan flags/options.
// Command yang mengimplementasikan interface ini dapat mendefinisikan flags mereka sendiri.
type FlaggedCommand interface {
	Command

	// DefineFlags mendefinisikan flags untuk command ini
	DefineFlags(fs *flag.FlagSet)
}

// CommandContext berisi dependencies dan arguments yang dibutuhkan command saat eksekusi.
type CommandContext struct {
	// Args adalah arguments yang tidak termasuk flags (positional arguments)
	Args []string

	// DB adalah database instance
	DB *PostgresDatabase

	// Router adalah router instance
	Router *Router

	// Config adalah application configuration
	Config *Config

	// Out adalah output writer untuk stdout (default: os.Stdout)
	// Digunakan untuk normal output dan testing
	Out io.Writer

	// Err adalah output writer untuk stderr (default: os.Stderr)
	// Digunakan untuk error messages dan warnings
	Err io.Writer
}

// Console adalah registry dan executor untuk CLI commands.
// Console mengelola semua registered commands dan menangani parsing/eksekusi.
type Console struct {
	commands map[string]Command
	db       *PostgresDatabase
	router   *Router
	config   *Config
	out      io.Writer // Output writer (default: os.Stdout)
	err      io.Writer // Error writer (default: os.Stderr)
}

// NewConsole membuat instance Console baru dengan dependencies yang diperlukan.
//
// Parameter:
//   - db: database instance (boleh nil jika tidak diperlukan)
//   - router: router instance (boleh nil jika tidak diperlukan)
//   - config: application config (boleh nil jika tidak diperlukan)
//
// Mengembalikan:
//   - *Console: instance console yang siap digunakan
//
// Contoh:
//
//	console := dim.NewConsole(db, router, config)
//	console.RegisterBuiltInCommands()
//	console.Run(os.Args[1:])
func NewConsole(db *PostgresDatabase, router *Router, config *Config) *Console {
	return &Console{
		commands: make(map[string]Command),
		db:       db,
		router:   router,
		config:   config,
		out:      os.Stdout,
		err:      os.Stderr,
	}
}

// SetOutput sets custom output writers for testing purposes.
// If out or err is nil, it will use os.Stdout or os.Stderr respectively.
func (c *Console) SetOutput(out, err io.Writer) {
	if out != nil {
		c.out = out
	}
	if err != nil {
		c.err = err
	}
}

// Register mendaftarkan custom command ke console.
// Command name harus unik, jika sudah ada akan mengembalikan error.
//
// Parameter:
//   - cmd: command yang akan didaftarkan
//
// Mengembalikan:
//   - error: nil jika sukses, error jika command name sudah terdaftar
//
// Contoh:
//
//	console.Register(&MyCustomCommand{})
func (c *Console) Register(cmd Command) error {
	name := cmd.Name()
	if _, exists := c.commands[name]; exists {
		return fmt.Errorf("command already registered: %s", name)
	}
	c.commands[name] = cmd
	return nil
}

// RegisterBuiltInCommands mendaftarkan semua built-in commands.
// Dipanggil setelah NewConsole() untuk menambahkan commands bawaan framework.
//
// Contoh:
//
//	console := dim.NewConsole(db, router, config)
//	console.RegisterBuiltInCommands()
func (c *Console) RegisterBuiltInCommands() {
	// Register built-in commands
	c.Register(&ServeCommand{})
	c.Register(&MigrateCommand{})
	c.Register(&MigrateRollbackCommand{})
	c.Register(&MigrateListCommand{})
	c.Register(&RouteListCommand{})
	c.Register(&HelpCommand{console: c})
}

// Run menjalankan command berdasarkan arguments yang diberikan.
// Jika args kosong, default ke command "serve".
// Menangani flag parsing untuk FlaggedCommand dan help (-h).
//
// Parameter:
//   - args: command arguments (biasanya os.Args[1:])
//
// Mengembalikan:
//   - error: nil jika sukses, error jika command tidak ditemukan atau eksekusi gagal
//
// Contoh:
//
//	if err := console.Run(os.Args[1:]); err != nil {
//	    log.Fatal(err)
//	}
func (c *Console) Run(args []string) error {
	// Default to serve if no args
	if len(args) == 0 {
		args = []string{"serve"}
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	// Find command
	cmd, exists := c.commands[cmdName]
	if !exists {
		return fmt.Errorf("unknown command: %s\nRun 'help' to see available commands", cmdName)
	}

	// Prepare context
	ctx := &CommandContext{
		Args:   cmdArgs,
		DB:     c.db,
		Router: c.router,
		Config: c.config,
		Out:    c.out,
		Err:    c.err,
	}

	// Check if command implements FlaggedCommand
	if flaggedCmd, ok := cmd.(FlaggedCommand); ok {
		fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)

		// Set output for flag errors
		fs.SetOutput(c.err)

		// Customize usage output
		fs.Usage = func() {
			fmt.Fprintf(c.err, "Usage: %s [options]\n\n", cmdName)
			fmt.Fprintf(c.err, "%s\n\n", cmd.Description())
			fmt.Fprintf(c.err, "Options:\n")
			fs.PrintDefaults()
		}

		// Define flags
		flaggedCmd.DefineFlags(fs)

		// Parse flags
		if err := fs.Parse(cmdArgs); err != nil {
			if err == flag.ErrHelp {
				return nil // Help already printed
			}
			return err
		}

		// Update context with remaining args (non-flag arguments)
		ctx.Args = fs.Args()
	} else {
		// Command doesn't have flags, check for help request
		if hasHelpFlag(cmdArgs) {
			printSimpleHelp(c.err, cmdName, cmd.Description())
			return nil
		}
	}

	return cmd.Execute(ctx)
}

// hasHelpFlag checks if args contain help flag (-h or -help)
func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "-help" {
			return true
		}
	}
	return false
}

// printSimpleHelp prints help message for non-flagged commands
func printSimpleHelp(w io.Writer, cmdName, description string) {
	fmt.Fprintf(w, "Usage: %s\n\n", cmdName)
	fmt.Fprintf(w, "%s\n\n", description)
	fmt.Fprintf(w, "This command does not accept any flags.\n")
	fmt.Fprintf(w, "Run 'help' to see all available commands.\n")
}
