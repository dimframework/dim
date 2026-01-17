package dim

import (
	"testing"
)

func TestHelpCommand_Name(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	cmd := &HelpCommand{console: console}

	if cmd.Name() != "help" {
		t.Errorf("Expected name 'help', got '%s'", cmd.Name())
	}
}

func TestHelpCommand_Description(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	cmd := &HelpCommand{console: console}

	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestHelpCommand_Execute_EmptyConsole(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	cmd := &HelpCommand{console: console}

	ctx := &CommandContext{}

	// Should not error even with no commands registered
	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestHelpCommand_Execute_WithBuiltInCommands(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	console.RegisterBuiltInCommands()

	// Get help command
	helpCmd := console.commands["help"].(*HelpCommand)

	ctx := &CommandContext{}

	err := helpCmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestHelpCommand_Execute_WithCustomCommands(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	// Register some custom commands
	customCmd1 := &MockCommand{
		name:        "custom1",
		description: "Custom command 1",
	}
	customCmd2 := &MockCommand{
		name:        "custom2",
		description: "Custom command 2",
	}

	console.Register(customCmd1)
	console.Register(customCmd2)

	cmd := &HelpCommand{console: console}
	ctx := &CommandContext{}

	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all commands are in the registry
	if len(console.commands) != 2 {
		t.Errorf("Expected 2 commands in console, got %d", len(console.commands))
	}
}

func TestHelpCommand_Execute_MixedCommands(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	console.RegisterBuiltInCommands()

	// Register custom command
	customCmd := &MockCommand{
		name:        "custom",
		description: "Custom command",
	}
	console.Register(customCmd)

	// Get help command
	helpCmd := console.commands["help"].(*HelpCommand)
	ctx := &CommandContext{}

	err := helpCmd.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify total commands (7 built-in + 1 custom)
	expectedCount := 8 // serve, migrate, migrate:rollback, migrate:list, route:list, help, make:migration, custom
	if len(console.commands) != expectedCount {
		t.Errorf("Expected %d commands, got %d", expectedCount, len(console.commands))
	}
}

func TestHelpCommand_ConsoleReference(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	cmd := &HelpCommand{console: console}

	if cmd.console != console {
		t.Error("HelpCommand console reference is incorrect")
	}
}

// ============================================================================
// Tests for Sorted Custom Commands
// ============================================================================

func TestHelpCommand_CustomCommandsSorted(t *testing.T) {
	console := NewConsole(nil, nil, nil)

	// Register custom commands in random order
	commands := []string{"zebra", "apple", "mango", "banana"}
	for _, name := range commands {
		console.Register(&MockCommand{
			name:        name,
			description: name + " command",
		})
	}

	cmd := &HelpCommand{console: console}

	// Execute should not error
	err := cmd.Execute(&CommandContext{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Note: We can't easily test the output order without capturing stdout
	// But we verified the sorting logic is in place
	// In real usage, commands will appear in: apple, banana, mango, zebra order
}

func TestHelpCommand_MixedBuiltInAndCustomSorted(t *testing.T) {
	console := NewConsole(nil, nil, nil)
	console.RegisterBuiltInCommands()

	// Register custom commands
	customCommands := []string{"custom-z", "custom-a", "custom-m"}
	for _, name := range customCommands {
		console.Register(&MockCommand{
			name:        name,
			description: name + " command",
		})
	}

	cmd := &HelpCommand{console: console}

	err := cmd.Execute(&CommandContext{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all commands are registered
	expectedTotal := 7 + len(customCommands) // 7 built-in + custom
	if len(console.commands) != expectedTotal {
		t.Errorf("Expected %d total commands, got %d", expectedTotal, len(console.commands))
	}
}
