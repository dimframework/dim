package dim

import (
	"flag"
	"testing"
)

// ============================================================================
// MigrateCommand Tests
// ============================================================================

func TestMigrateCommand_Name(t *testing.T) {
	cmd := &MigrateCommand{}
	if cmd.Name() != "migrate" {
		t.Errorf("Expected name 'migrate', got '%s'", cmd.Name())
	}
}

func TestMigrateCommand_Description(t *testing.T) {
	cmd := &MigrateCommand{}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMigrateCommand_DefineFlags(t *testing.T) {
	cmd := &MigrateCommand{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Test that verbose flag is defined
	vFlag := fs.Lookup("v")
	if vFlag == nil {
		t.Fatal("verbose (-v) flag not defined")
	}
}

func TestMigrateCommand_Execute_NoDatabase(t *testing.T) {
	cmd := &MigrateCommand{}
	ctx := &CommandContext{
		DB: nil,
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when database is nil")
	}

	if err.Error() != "database connection required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMigrateCommand_VerboseFlag(t *testing.T) {
	cmd := &MigrateCommand{}
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Parse with -v flag
	args := []string{"-v"}
	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	if !cmd.verbose {
		t.Error("Expected verbose to be true when -v flag is set")
	}
}

func TestMigrateCommand_VerboseDefault(t *testing.T) {
	cmd := &MigrateCommand{}

	// Default should be false
	if cmd.verbose {
		t.Error("Expected verbose to be false by default")
	}
}

// ============================================================================
// MigrateRollbackCommand Tests
// ============================================================================

func TestMigrateRollbackCommand_Name(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	if cmd.Name() != "migrate:rollback" {
		t.Errorf("Expected name 'migrate:rollback', got '%s'", cmd.Name())
	}
}

func TestMigrateRollbackCommand_Description(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMigrateRollbackCommand_DefineFlags(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Test that step flag is defined
	stepFlag := fs.Lookup("step")
	if stepFlag == nil {
		t.Fatal("step flag not defined")
	}
}

func TestMigrateRollbackCommand_Execute_NoDatabase(t *testing.T) {
	cmd := &MigrateRollbackCommand{
		steps: 1,
	}
	ctx := &CommandContext{
		DB: nil,
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when database is nil")
	}

	if err.Error() != "database connection required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMigrateRollbackCommand_Execute_InvalidSteps(t *testing.T) {
	cmd := &MigrateRollbackCommand{
		steps: 0,
	}
	ctx := &CommandContext{
		DB: &PostgresDatabase{}, // Mock DB
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when steps is 0")
	}

	if err.Error() != "steps must be greater than 0" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMigrateRollbackCommand_Execute_NegativeSteps(t *testing.T) {
	cmd := &MigrateRollbackCommand{
		steps: -1,
	}
	ctx := &CommandContext{
		DB: &PostgresDatabase{}, // Mock DB
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when steps is negative")
	}

	if err.Error() != "steps must be greater than 0" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMigrateRollbackCommand_StepFlag(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	fs := flag.NewFlagSet("migrate:rollback", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Parse with -step flag
	args := []string{"-step", "3"}
	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	if cmd.steps != 3 {
		t.Errorf("Expected steps to be 3, got %d", cmd.steps)
	}
}

func TestMigrateRollbackCommand_StepDefaultValue(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	fs := flag.NewFlagSet("migrate:rollback", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Parse without flag (should use default)
	args := []string{}
	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	if cmd.steps != 1 {
		t.Errorf("Expected default steps to be 1, got %d", cmd.steps)
	}
}

// ============================================================================
// MigrateListCommand Tests
// ============================================================================

func TestMigrateListCommand_Name(t *testing.T) {
	cmd := &MigrateListCommand{}
	if cmd.Name() != "migrate:list" {
		t.Errorf("Expected name 'migrate:list', got '%s'", cmd.Name())
	}
}

func TestMigrateListCommand_Description(t *testing.T) {
	cmd := &MigrateListCommand{}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMigrateListCommand_Execute_NoDatabase(t *testing.T) {
	cmd := &MigrateListCommand{}
	ctx := &CommandContext{
		DB: nil,
	}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error when database is nil")
	}

	if err.Error() != "database connection required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// ============================================================================
// New Tests for Force Flag
// ============================================================================

func TestMigrateRollbackCommand_ForceFlag(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	fs := flag.NewFlagSet("migrate:rollback", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Parse with -force flag
	args := []string{"-force"}
	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	if !cmd.force {
		t.Error("Expected force to be true when -force flag is set")
	}
}

func TestMigrateRollbackCommand_ForceFlagDefault(t *testing.T) {
	cmd := &MigrateRollbackCommand{}

	// Default should be false
	if cmd.force {
		t.Error("Expected force to be false by default")
	}
}

func TestMigrateRollbackCommand_BothFlags(t *testing.T) {
	cmd := &MigrateRollbackCommand{}
	fs := flag.NewFlagSet("migrate:rollback", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Parse with both -step and -force flags
	args := []string{"-step", "3", "-force"}
	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	if cmd.steps != 3 {
		t.Errorf("Expected steps to be 3, got %d", cmd.steps)
	}

	if !cmd.force {
		t.Error("Expected force to be true when -force flag is set")
	}
}
