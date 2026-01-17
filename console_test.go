package dim

import (
	"flag"
	"fmt"
	"strings"
	"testing"
)

// MockCommand untuk testing
type MockCommand struct {
	name        string
	description string
	executed    bool
	executeErr  error
	args        []string
}

func (m *MockCommand) Name() string        { return m.name }
func (m *MockCommand) Description() string { return m.description }
func (m *MockCommand) Execute(ctx *CommandContext) error {
	m.executed = true
	m.args = ctx.Args
	return m.executeErr
}

// MockFlaggedCommand untuk testing commands dengan flags
type MockFlaggedCommand struct {
	MockCommand
	flagValue string
}

func (m *MockFlaggedCommand) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&m.flagValue, "flag", "", "Test flag")
}

// TestNewConsole tests Console creation
func TestNewConsole(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	if console == nil {
		t.Fatal("NewConsole returned nil")
	}

	if console.commands == nil {
		t.Error("Console commands map is nil")
	}

	if len(console.commands) != 0 {
		t.Errorf("Expected empty commands map, got %d commands", len(console.commands))
	}
}

// TestConsoleRegister tests command registration
func TestConsoleRegister(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	cmd := &MockCommand{
		name:        "test",
		description: "Test command",
	}

	err := console.Register(cmd)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if len(console.commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(console.commands))
	}

	if _, exists := console.commands["test"]; !exists {
		t.Error("Command 'test' not found in registry")
	}
}

// TestConsoleRegisterDuplicate tests duplicate command registration
func TestConsoleRegisterDuplicate(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	cmd1 := &MockCommand{name: "test", description: "First"}
	cmd2 := &MockCommand{name: "test", description: "Second"}

	err := console.Register(cmd1)
	if err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	err = console.Register(cmd2)
	if err == nil {
		t.Error("Expected error when registering duplicate command, got nil")
	}

	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("Expected 'already registered' error, got: %v", err)
	}
}

// TestConsoleRegisterBuiltInCommands tests built-in command registration
func TestConsoleRegisterBuiltInCommands(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	console.RegisterBuiltInCommands()

	expectedCommands := []string{
		"serve",
		"migrate",
		"migrate:rollback",
		"migrate:list",
		"route:list",
		"help",
		"make:migration",
	}

	for _, cmdName := range expectedCommands {
		if _, exists := console.commands[cmdName]; !exists {
			t.Errorf("Built-in command '%s' not registered", cmdName)
		}
	}

	if len(console.commands) != len(expectedCommands) {
		t.Errorf("Expected %d built-in commands, got %d", len(expectedCommands), len(console.commands))
	}
}

// TestConsoleRunSimpleCommand tests running a simple command
func TestConsoleRunSimpleCommand(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	cmd := &MockCommand{
		name:        "test",
		description: "Test command",
	}

	console.Register(cmd)

	err := console.Run([]string{"test"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !cmd.executed {
		t.Error("Command was not executed")
	}
}

// TestConsoleRunWithArgs tests running command with arguments
func TestConsoleRunWithArgs(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	cmd := &MockCommand{
		name:        "test",
		description: "Test command",
	}

	console.Register(cmd)

	err := console.Run([]string{"test", "arg1", "arg2"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !cmd.executed {
		t.Error("Command was not executed")
	}

	expectedArgs := []string{"arg1", "arg2"}
	if len(cmd.args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.args))
	}

	for i, arg := range expectedArgs {
		if cmd.args[i] != arg {
			t.Errorf("Arg %d: expected %s, got %s", i, arg, cmd.args[i])
		}
	}
}

// TestConsoleRunFlaggedCommand tests running command with flags
func TestConsoleRunFlaggedCommand(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	cmd := &MockFlaggedCommand{
		MockCommand: MockCommand{
			name:        "test",
			description: "Test flagged command",
		},
	}

	console.Register(cmd)

	err := console.Run([]string{"test", "-flag", "value"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !cmd.executed {
		t.Error("Command was not executed")
	}

	if cmd.flagValue != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", cmd.flagValue)
	}
}

// TestConsoleRunFlaggedCommandWithRemainingArgs tests flags with remaining args
func TestConsoleRunFlaggedCommandWithRemainingArgs(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	cmd := &MockFlaggedCommand{
		MockCommand: MockCommand{
			name:        "test",
			description: "Test flagged command",
		},
	}

	console.Register(cmd)

	err := console.Run([]string{"test", "-flag", "value", "arg1", "arg2"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if cmd.flagValue != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", cmd.flagValue)
	}

	expectedArgs := []string{"arg1", "arg2"}
	if len(cmd.args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.args))
	}
}

// TestConsoleRunUnknownCommand tests error handling for unknown command
func TestConsoleRunUnknownCommand(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	err := console.Run([]string{"unknown"})
	if err == nil {
		t.Error("Expected error for unknown command, got nil")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", err)
	}
}

// TestConsoleRunCommandError tests error propagation from command
func TestConsoleRunCommandError(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	expectedErr := fmt.Errorf("command execution error")
	cmd := &MockCommand{
		name:        "test",
		description: "Test command",
		executeErr:  expectedErr,
	}

	console.Register(cmd)

	err := console.Run([]string{"test"})
	if err == nil {
		t.Error("Expected error from command execution, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

// TestConsoleRunDefaultCommand tests default command (serve)
func TestConsoleRunDefaultCommand(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	serveCmd := &MockCommand{
		name:        "serve",
		description: "Serve command",
	}

	console.Register(serveCmd)

	// Run with empty args should default to "serve"
	err := console.Run([]string{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !serveCmd.executed {
		t.Error("Default command (serve) was not executed")
	}
}

// TestConsoleRunWithDependencies tests command execution with dependencies
func TestConsoleRunWithDependencies(t *testing.T) {
	// Create mock dependencies
	router := NewRouter()
	config := &Config{
		Server: ServerConfig{
			Port: "8080",
		},
	}

	console := NewConsole(nil, router, config)

	cmd := &MockCommand{
		name:        "test",
		description: "Test command",
	}

	console.Register(cmd)

	err := console.Run([]string{"test"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !cmd.executed {
		t.Error("Command was not executed")
	}
}

// ============================================================================
// Tests for Help Flag Helper
// ============================================================================

func TestHasHelpFlag_WithDashH(t *testing.T) {
	args := []string{"-h"}
	if !hasHelpFlag(args) {
		t.Error("Expected hasHelpFlag to return true for -h")
	}
}

func TestHasHelpFlag_WithDashHelp(t *testing.T) {
	args := []string{"-help"}
	if !hasHelpFlag(args) {
		t.Error("Expected hasHelpFlag to return true for -help")
	}
}

func TestHasHelpFlag_WithOtherArgs(t *testing.T) {
	args := []string{"arg1", "arg2"}
	if hasHelpFlag(args) {
		t.Error("Expected hasHelpFlag to return false for non-help args")
	}
}

func TestHasHelpFlag_WithMixedArgs(t *testing.T) {
	args := []string{"arg1", "-h", "arg2"}
	if !hasHelpFlag(args) {
		t.Error("Expected hasHelpFlag to return true when -h is present among other args")
	}
}

func TestHasHelpFlag_EmptyArgs(t *testing.T) {
	args := []string{}
	if hasHelpFlag(args) {
		t.Error("Expected hasHelpFlag to return false for empty args")
	}
}

// ============================================================================
// Tests for IO Injection
// ============================================================================

func TestConsole_SetOutput(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	// Create custom writers
	var outBuf, errBuf strings.Builder

	// Set custom output
	console.SetOutput(&outBuf, &errBuf)

	if console.out != &outBuf {
		t.Error("Expected out to be set to custom writer")
	}

	if console.err != &errBuf {
		t.Error("Expected err to be set to custom writer")
	}
}

func TestConsole_SetOutput_NilHandling(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	originalOut := console.out
	originalErr := console.err

	// Set nil writers should not change
	console.SetOutput(nil, nil)

	if console.out != originalOut {
		t.Error("Expected out to remain unchanged when nil")
	}

	if console.err != originalErr {
		t.Error("Expected err to remain unchanged when nil")
	}
}
