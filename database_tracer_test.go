package dim

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestHookManager memverifikasi logika internal hook handler.
func TestHookManager(t *testing.T) {
	t.Run("Execute calls all hooks", func(t *testing.T) {
		hm := &hookManager{}

		var callCount int
		mockHook := func(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) {
			callCount++
		}

		hm.Add(mockHook)
		hm.Add(mockHook)

		hm.Execute(context.Background(), "SELECT 1", nil, time.Second, nil)

		if callCount != 2 {
			t.Errorf("Expected 2 hook calls, got %d", callCount)
		}
	})

	t.Run("Thread safety", func(t *testing.T) {
		hm := &hookManager{}
		var wg sync.WaitGroup
		hookCount := 100

		// Concurrent Adds
		for i := 0; i < hookCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hm.Add(func(ctx context.Context, q string, a []interface{}, d time.Duration, e error) {})
			}()
		}

		wg.Wait()
		if len(hm.hooks) != hookCount {
			t.Errorf("Hook count mismatch after concurrent adds. Expected %d, got %d", hookCount, len(hm.hooks))
		}
	})
}

// TestTracerContextLogic mensimulasikan flow tracer pgx tanpa koneksi DB riil.
// Kita mengetes apakah TraceQueryStart menyimpan data ke context dan TraceQueryEnd mengambilnya kembali.
func TestTracerContextLogic(t *testing.T) {
	hm := &hookManager{}
	tracer := &dbTracer{hm: hm}

	// Capture hook output
	var capturedQuery string
	var capturedArgs []interface{}

	hm.Add(func(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) {
		capturedQuery = query
		capturedArgs = args
	})

	// 1. Simulate Start
	ctx := context.Background()
	sql := "SELECT * FROM users WHERE id = $1"
	args := []interface{}{123}

	dataStart := pgx.TraceQueryStartData{
		SQL:  sql,
		Args: args,
	}
	ctx = tracer.TraceQueryStart(ctx, nil, dataStart)

	// Verify context has values (implementation detail check)
	if ctx.Value(traceStartKey) == nil {
		t.Error("traceStartKey should be present in context")
	}
	if ctx.Value(traceQueryKey) != sql {
		t.Errorf("traceQueryKey mismatch. Expected %s, got %v", sql, ctx.Value(traceQueryKey))
	}

	// 2. Simulate End logic
	// Note: We normally don't call TraceQueryEnd directly differently than pgx,
	// but we want to ensure it extracts from context and calls Execute
	dataEnd := pgx.TraceQueryEndData{
		Err: errors.New("test error"),
	}

	// tracer.TraceQueryEnd(ctx, nil, dataEnd)
	// Is hard to test because TraceQueryEnd returns void and calls internal hookManager.
	// But since we added a hook to hm above, it should capture the values.

	tracer.TraceQueryEnd(ctx, nil, dataEnd)

	if capturedQuery != sql {
		t.Errorf("capturedQuery mismatch. Expected %s, got %s", sql, capturedQuery)
	}
	// Manual slice comparison since we removed testify
	if len(capturedArgs) != len(args) {
		t.Errorf("capturedArgs length mismatch. Expected %d, got %d", len(args), len(capturedArgs))
	}
	if len(capturedArgs) > 0 && capturedArgs[0] != args[0] {
		t.Errorf("capturedArgs[0] mismatch. Expected %v, got %v", args[0], capturedArgs[0])
	}
}

func TestSanitizeArgs(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		args     []interface{}
		expected []interface{}
		masked   bool
	}{
		{
			name:     "Normal query",
			query:    "SELECT * FROM users WHERE id = $1",
			args:     []interface{}{1},
			expected: []interface{}{1},
			masked:   false,
		},
		{
			name:     "Sensitive query (password)",
			query:    "INSERT INTO users (email, password) VALUES ($1, $2)",
			args:     []interface{}{"test@example.com", "secret123"},
			expected: []interface{}{"*****", "*****"},
			masked:   true,
		},
		{
			name:     "Sensitive query (email)",
			query:    "SELECT * FROM users WHERE email = $1",
			args:     []interface{}{"user@example.com"},
			expected: []interface{}{"*****"},
			masked:   true,
		},
		{
			name:     "Case insensitive check",
			query:    "UPDATE tokens SET TOKEN = $1",
			args:     []interface{}{"xyz-token"},
			expected: []interface{}{"*****"},
			masked:   true,
		},
		{
			name:     "Empty args",
			query:    "SELECT * FROM users",
			args:     []interface{}{},
			expected: []interface{}{},
			masked:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeArgs(tt.query, tt.args)

			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch. Expected %d, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Arg mismatch at index %d. Expected %v, got %v", i, tt.expected[i], result[i])
				}
			}
		})
	}
}
