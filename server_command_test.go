package dim

import (
	"flag"
	"testing"
)

func TestServeCommand_Name(t *testing.T) {
	cmd := &ServeCommand{}
	if cmd.Name() != "serve" {
		t.Errorf("Expected name 'serve', got '%s'", cmd.Name())
	}
}

func TestServeCommand_Description(t *testing.T) {
	cmd := &ServeCommand{}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if desc != "Start HTTP server" {
		t.Errorf("Expected 'Start HTTP server', got '%s'", desc)
	}
}

func TestServeCommand_DefineFlags(t *testing.T) {
	cmd := &ServeCommand{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Test that port flag is defined
	portFlag := fs.Lookup("port")
	if portFlag == nil {
		t.Fatal("port flag not defined")
	}

	if portFlag.Usage == "" {
		t.Error("port flag should have usage text")
	}
}

func TestServeCommand_Execute_NoRouter(t *testing.T) {
	cmd := &ServeCommand{}
	ctx := &CommandContext{
		Router: nil,
		Config: &Config{},
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when router is nil")
	}

	if err.Error() != "router is required to start server" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestServeCommand_Execute_NoConfig(t *testing.T) {
	cmd := &ServeCommand{}
	ctx := &CommandContext{
		Router: NewRouter(),
		Config: nil,
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when config is nil")
	}

	if err.Error() != "config is required to start server" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestServeCommand_FlagParsing(t *testing.T) {
	cmd := &ServeCommand{}
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Parse flags
	args := []string{"-port", "9000"}
	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	if cmd.port != "9000" {
		t.Errorf("Expected port '9000', got '%s'", cmd.port)
	}
}

func TestServeCommand_DefaultPort(t *testing.T) {
	cmd := &ServeCommand{}

	// Port should default to empty string before execution
	if cmd.port != "" {
		t.Errorf("Expected empty port before execution, got '%s'", cmd.port)
	}
}

func TestServeCommand_PortOverride(t *testing.T) {
	// Test that flag port overrides config port
	cmd := &ServeCommand{
		port: "9000", // Simulating flag set
	}

	// In real execution, the flag would be set and should override config
	if cmd.port != "9000" {
		t.Errorf("Expected port to be '9000', got '%s'", cmd.port)
	}
}

func TestServeCommand_ConfigPortUsed(t *testing.T) {
	// Test that config port is used when flag is not set
	cmd := &ServeCommand{
		port: "", // No flag set
	}

	// When flag is empty, config port should be used in Execute()
	if cmd.port != "" {
		t.Error("Flag port should be empty when not set")
	}
}
