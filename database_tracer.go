package dim

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// QueryHook is a function that is called after a query execution.
type QueryHook func(ctx context.Context, query string, args []interface{}, duration time.Duration, err error)

// Observable defines specific interface for adding observability hooks
type Observable interface {
	AddHook(hook QueryHook)
}

// hookManager handles thread-safe hook management
type hookManager struct {
	hooks []QueryHook
	mu    sync.RWMutex
}

func (hm *hookManager) Add(hook QueryHook) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.hooks = append(hm.hooks, hook)
}

func (hm *hookManager) Execute(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	// Skip if no hooks
	if len(hm.hooks) == 0 {
		return
	}
	for _, hook := range hm.hooks {
		hook(ctx, query, args, duration, err)
	}
}

// dbTracer implements pgx.QueryTracer to hook into low-level query execution
type dbTracer struct {
	hm *hookManager
}

type traceContextKey string

const (
	traceStartKey traceContextKey = "traceStart"
	traceQueryKey traceContextKey = "traceQuery"
	traceArgsKey  traceContextKey = "traceArgs"
)

var sensitiveKeywords = []string{
	"password",
	"email",
	"token",
	"secret",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
}

// sanitizeArgs masks arguments if the query contains sensitive keywords.
func sanitizeArgs(query string, args []interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}

	queryLower := strings.ToLower(query)
	isSensitive := false
	for _, kw := range sensitiveKeywords {
		if strings.Contains(queryLower, kw) {
			isSensitive = true
			break
		}
	}

	if !isSensitive {
		return args
	}

	// Create a new slice to avoid modifying the original args
	masked := make([]interface{}, len(args))
	for i := range args {
		masked[i] = "*****"
	}
	return masked
}

func (dt *dbTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = context.WithValue(ctx, traceStartKey, time.Now())
	ctx = context.WithValue(ctx, traceQueryKey, data.SQL)
	// IMPORTANT: args are available here
	ctx = context.WithValue(ctx, traceArgsKey, data.Args)
	return ctx
}

func (dt *dbTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	start, ok := ctx.Value(traceStartKey).(time.Time)
	if !ok {
		return
	}
	query, _ := ctx.Value(traceQueryKey).(string)
	// data.Args from TraceQueryStart is []any (interface{})
	args, _ := ctx.Value(traceArgsKey).([]interface{})

	// Sanitize args before passing to hooks
	safeArgs := sanitizeArgs(query, args)

	duration := time.Since(start)
	dt.hm.Execute(ctx, query, safeArgs, duration, data.Err)
}
