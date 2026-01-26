package dim

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
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

// ============================================================================
// MakeMigrationCommand Tests
// ============================================================================

func TestMakeMigrationCommand_Name(t *testing.T) {
	cmd := &MakeMigrationCommand{}
	if cmd.Name() != "make:migration" {
		t.Errorf("Expected name 'make:migration', got '%s'", cmd.Name())
	}
}

func TestMakeMigrationCommand_Description(t *testing.T) {
	cmd := &MakeMigrationCommand{}
	desc := cmd.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMakeMigrationCommand_DefineFlags(t *testing.T) {
	cmd := &MakeMigrationCommand{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cmd.DefineFlags(fs)

	// Check flags
	dirFlag := fs.Lookup("dir")
	if dirFlag == nil {
		t.Fatal("dir flag not defined")
	}
	if dirFlag.DefValue != "migrations" {
		t.Errorf("Expected default dir 'migrations', got '%s'", dirFlag.DefValue)
	}

	pkgFlag := fs.Lookup("pkg")
	if pkgFlag == nil {
		t.Fatal("pkg flag not defined")
	}
}

func TestMakeMigrationCommand_Execute_GeneratesValidFile(t *testing.T) {
	// Create temp directory for migrations
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := &MakeMigrationCommand{
		dir: tmpDir,
	}

	ctx := &CommandContext{
		Args: []string{"test_feature"},
	}

	// Execute command
	if err := cmd.Execute(ctx); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify file was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file created, got %d", len(files))
	}

	fileName := files[0].Name()
	if !strings.HasSuffix(fileName, "_test_feature.go") {
		t.Errorf("Unexpected filename: %s", fileName)
	}

	// Verify content
	contentBytes, err := os.ReadFile(filepath.Join(tmpDir, fileName))
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	content := string(contentBytes)

	// Check for critical parts of the template that were fixed
	expectedStrings := []string{
		"package migrations", // Default package name
		"dim.Register(dim.Migration{",
		"func UpTestFeature(db dim.Database) error {",   // Correct interface
		"func DownTestFeature(db dim.Database) error {", // Correct interface
		"err := db.Exec(context.Background(), query)",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(content, expected) {
			t.Errorf("Generated file missing expected string: '%s'", expected)
		}
	}

	// Ensure old pgxpool import is NOT present
	if strings.Contains(content, "github.com/jackc/pgx/v5/pgxpool") {
		t.Error("Generated file contains deprecated pgxpool import")
	}
}
