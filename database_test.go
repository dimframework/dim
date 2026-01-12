package dim

import (
	"testing"
)

func TestFormatConnString(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		database string
		username string
		password string
		sslmode  string
		expected string
	}{
		{
			name:     "basic connection with disable ssl",
			host:     "localhost",
			port:     5432,
			database: "testdb",
			username: "user",
			password: "pass",
			sslmode:  "disable",
			expected: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
		{
			name:     "remote host with require ssl",
			host:     "db.example.com",
			port:     5432,
			database: "production",
			username: "dbuser",
			password: "dbpass",
			sslmode:  "require",
			expected: "postgres://dbuser:dbpass@db.example.com:5432/production?sslmode=require",
		},
		{
			name:     "custom port with verify-full ssl",
			host:     "localhost",
			port:     5433,
			database: "testdb",
			username: "user",
			password: "pass",
			sslmode:  "verify-full",
			expected: "postgres://user:pass@localhost:5433/testdb?sslmode=verify-full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatConnectionString(tt.host, tt.port, tt.database, tt.username, tt.password, tt.sslmode)
			if result != tt.expected {
				t.Errorf("formatConnectionString() = %s, want %s", result, tt.expected)
			}
		})
	}
}
