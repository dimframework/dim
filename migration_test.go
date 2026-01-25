package dim

import (
	"testing"
)

func TestFrameworkMigrationsToggle(t *testing.T) {
	// Simpan state awal agar tidak mengganggu test lain
	originalState := includeFrameworkMigrations
	defer func() {
		includeFrameworkMigrations = originalState
	}()

	t.Run("DefaultEnabled", func(t *testing.T) {
		// Reset ke true (default)
		includeFrameworkMigrations = true
		migrations := GetFrameworkMigrations()
		if len(migrations) == 0 {
			t.Error("Expected framework migrations to be present by default")
		}

		// Verifikasi keberadaan migrasi spesifik (misal: users)
		foundUsers := false
		for _, m := range migrations {
			if m.Name == "create_users_table" {
				foundUsers = true
				break
			}
		}
		if !foundUsers {
			t.Error("Expected create_users_table migration to be present")
		}
	})

	t.Run("CanDisable", func(t *testing.T) {
		// Disable migrations
		DisableFrameworkMigrations()

		migrations := GetFrameworkMigrations()
		if len(migrations) != 0 {
			t.Errorf("Expected 0 migrations after disable, got %d", len(migrations))
		}
	})
}
