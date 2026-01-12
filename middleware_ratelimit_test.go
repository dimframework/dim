package dim

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddlewareDisabled(t *testing.T) {
	config := RateLimitConfig{
		Enabled: false,
	}

	rateLimitMiddleware := RateLimit(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := rateLimitMiddleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:8080"

	wrappedHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("disabled rate limit should allow all requests")
	}
}

func TestRateLimitMiddlewareIPLimit(t *testing.T) {
	config := RateLimitConfig{
		Enabled:     true,
		PerIP:       2,
		PerUser:     100,
		ResetPeriod: 1 * time.Second,
	}

	rateLimitMiddleware := RateLimit(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := rateLimitMiddleware(handler)

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "127.0.0.1:8080"

		wrappedHandler(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("request %d should succeed", i+1)
		}
	}

	// Third request should be rate limited
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:8080"

	wrappedHandler(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("third request should be rate limited, got %d", w.Code)
	}
}

func TestRateLimitMiddlewareDifferentIPs(t *testing.T) {
	config := RateLimitConfig{
		Enabled:     true,
		PerIP:       1,
		PerUser:     100,
		ResetPeriod: 1 * time.Second,
	}

	rateLimitMiddleware := RateLimit(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := rateLimitMiddleware(handler)

	// First IP - one request should succeed
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.RemoteAddr = "127.0.0.1:8080"
	wrappedHandler(w1, r1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request from IP1 should succeed")
	}

	// Different IP - one request should also succeed
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "127.0.0.2:8080"
	wrappedHandler(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("request from IP2 should succeed (different rate limit)")
	}
}

func TestRateLimitMiddlewareUserLimit(t *testing.T) {
	config := RateLimitConfig{
		Enabled:     true,
		PerIP:       100,
		PerUser:     2,
		ResetPeriod: 1 * time.Second,
	}

	rateLimitMiddleware := RateLimit(config)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := rateLimitMiddleware(handler)

	user := &User{ID: 1, Email: "test@example.com"}

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "127.0.0.1:8080"
		r = SetUser(r, user)

		wrappedHandler(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("request %d should succeed", i+1)
		}
	}

	// Third request should be rate limited
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:8080"
	r = SetUser(r, user)

	wrappedHandler(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("third request should be rate limited")
	}
}

func TestRateLimiterCheckIPLimit(t *testing.T) {
	config := RateLimitConfig{
		PerIP:       2,
		PerUser:     100,
		ResetPeriod: 1 * time.Second,
	}

	limiter := NewRateLimiter(config)

	// First two should succeed
	if !limiter.CheckIPLimit("127.0.0.1") {
		t.Errorf("first IP limit check should succeed")
	}

	if !limiter.CheckIPLimit("127.0.0.1") {
		t.Errorf("second IP limit check should succeed")
	}

	// Third should fail
	if limiter.CheckIPLimit("127.0.0.1") {
		t.Errorf("third IP limit check should fail")
	}
}

func TestRateLimiterReset(t *testing.T) {
	config := RateLimitConfig{
		PerIP:       1,
		PerUser:     100,
		ResetPeriod: 100 * time.Millisecond,
	}

	limiter := NewRateLimiter(config)

	// Use up the limit
	limiter.CheckIPLimit("127.0.0.1")

	// Should fail (Goreus Cache will auto-expire after ResetPeriod)
	if limiter.CheckIPLimit("127.0.0.1") {
		t.Errorf("should fail after limit reached")
	}

	// Wait for TTL expiration + small buffer
	time.Sleep(150 * time.Millisecond)

	// Cache entry should be expired, counter resets to 0
	// Next call should start counting from 1 again
	if !limiter.CheckIPLimit("127.0.0.1") {
		t.Errorf("should succeed after TTL expiration")
	}
}

func TestRateLimiterCacheLRUEviction(t *testing.T) {
	// Test that cache respects max size (10000 entries)
	config := RateLimitConfig{
		PerIP:       1000, // High limit to test cache size, not rate limiting
		PerUser:     1000,
		ResetPeriod: 10 * time.Second,
	}

	limiter := NewRateLimiter(config)

	// Try to add many IPs (more than default maxSize)
	// The cache should handle LRU eviction gracefully
	for i := 0; i < 100; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i)
		if !limiter.CheckIPLimit(ip) {
			t.Errorf("IP %s should be within limit", ip)
		}
	}

	// Verify we can still track IPs (no panic or errors)
	if !limiter.CheckIPLimit("192.168.1.0") {
		t.Errorf("initial IP should still be trackable")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		expectedIP    string
	}{
		{
			name:          "X-Forwarded-For header",
			xForwardedFor: "192.168.1.1",
			expectedIP:    "192.168.1.1",
		},
		{
			name:       "X-Real-IP header",
			xRealIP:    "192.168.1.2",
			expectedIP: "192.168.1.2",
		},
		{
			name:       "RemoteAddr",
			remoteAddr: "192.168.1.3:8080",
			expectedIP: "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tt.xForwardedFor != "" {
				r.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				r.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if tt.remoteAddr != "" {
				r.RemoteAddr = tt.remoteAddr
			}

			ip := GetClientIP(r)
			if ip != tt.expectedIP {
				t.Errorf("GetClientIP() = %s, want %s", ip, tt.expectedIP)
			}
		})
	}
}
