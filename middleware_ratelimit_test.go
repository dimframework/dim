package dim

import (
	"context"
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

	user := &TokenUser{ID: "1", Email: "test@example.com"}

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

	limiter := NewRateLimiter(config, nil)
	ctx := context.Background()

	// First two should succeed
	allowed, err := limiter.CheckIPLimit(ctx, "127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !allowed {
		t.Errorf("first IP limit check should succeed")
	}

	allowed, err = limiter.CheckIPLimit(ctx, "127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !allowed {
		t.Errorf("second IP limit check should succeed")
	}

	// Third should fail
	allowed, err = limiter.CheckIPLimit(ctx, "127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if allowed {
		t.Errorf("third IP limit check should fail")
	}
}

func TestRateLimiterReset(t *testing.T) {
	config := RateLimitConfig{
		PerIP:       1,
		PerUser:     100,
		ResetPeriod: 100 * time.Millisecond,
	}

	limiter := NewRateLimiter(config, nil)
	ctx := context.Background()

	// Use up the limit
	limiter.CheckIPLimit(ctx, "127.0.0.1")

	// Should fail (Goreus Cache will auto-expire after ResetPeriod)
	allowed, _ := limiter.CheckIPLimit(ctx, "127.0.0.1")
	if allowed {
		t.Errorf("should fail after limit reached")
	}

	// Wait for TTL expiration + small buffer
	time.Sleep(150 * time.Millisecond)

	// Cache entry should be expired, counter resets to 0
	// Next call should start counting from 1 again
	allowed, _ = limiter.CheckIPLimit(ctx, "127.0.0.1")
	if !allowed {
		t.Errorf("should succeed after TTL expiration")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xRealIP    string
		expectedIP string
	}{
		{
			name:       "uses RemoteAddr, ignores X-Forwarded-For",
			xff:        "1.2.3.4",
			remoteAddr: "10.0.0.1:8080",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "uses RemoteAddr, ignores X-Real-IP",
			xRealIP:    "1.2.3.4",
			remoteAddr: "10.0.0.2:8080",
			expectedIP: "10.0.0.2",
		},
		{
			name:       "uses RemoteAddr directly",
			remoteAddr: "192.168.1.3:8080",
			expectedIP: "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
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

func TestGetClientIPWithTrustedProxies(t *testing.T) {
	tests := []struct {
		name              string
		xff               string
		xffLines          []string // baris X-Forwarded-For terpisah (Header.Add berulang)
		remoteAddr        string
		trustedProxyCount int
		expectedIP        string
	}{
		{
			name:              "zero proxies falls back to RemoteAddr",
			xff:               "1.2.3.4",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 0,
			expectedIP:        "10.0.0.1",
		},
		{
			name:              "negative proxies falls back to RemoteAddr",
			xff:               "1.2.3.4",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: -1,
			expectedIP:        "10.0.0.1",
		},
		{
			name:              "one trusted proxy, returns second from right",
			xff:               "spoofed, 5.6.7.8",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "5.6.7.8",
		},
		{
			name:              "one trusted proxy, single entry in XFF",
			xff:               "5.6.7.8",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "5.6.7.8",
		},
		{
			name:              "two trusted proxies, returns third from right",
			xff:               "spoofed, 5.6.7.8, 10.0.0.2",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 2,
			expectedIP:        "5.6.7.8",
		},
		{
			// Rantai proxy lebih pendek daripada konfigurasi: entri paling kiri
			// dikendalikan klien, jadi harus jatuh ke RemoteAddr, bukan di-clamp.
			name:              "trustedProxyCount exceeds XFF entries, falls back to RemoteAddr",
			xff:               "5.6.7.8",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 5,
			expectedIP:        "10.0.0.1",
		},
		{
			name:              "no XFF header, falls back to RemoteAddr",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "10.0.0.1",
		},
		{
			name:              "invalid IP in XFF, falls back to RemoteAddr",
			xff:               "not-an-ip",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "10.0.0.1",
		},
		{
			name:              "spoofed XFF with one trusted proxy returns actual client IP (rightmost - 1)",
			xff:               "1.2.3.4, 5.6.7.8, 9.10.11.12",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "9.10.11.12",
		},
		{
			name:              "IPv6 address in XFF",
			xff:               "::1",
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "::1",
		},
		{
			// Klien mengirim baris X-Forwarded-For sendiri; proxy tepercaya
			// menambahkan baris terpisah berisi IP klien yang sebenarnya.
			// Seluruh baris harus digabungkan sebelum dibaca dari kanan.
			name:              "spoofed XFF on a separate header line is not trusted",
			xffLines:          []string{"6.6.6.6", "203.0.113.9"},
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 1,
			expectedIP:        "203.0.113.9",
		},
		{
			name:              "multiple header lines counted as one chain",
			xffLines:          []string{"6.6.6.6, 198.51.100.7", "203.0.113.9"},
			remoteAddr:        "10.0.0.1:8080",
			trustedProxyCount: 2,
			expectedIP:        "198.51.100.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			for _, line := range tt.xffLines {
				r.Header.Add("X-Forwarded-For", line)
			}
			if tt.remoteAddr != "" {
				r.RemoteAddr = tt.remoteAddr
			}

			ip := GetClientIPWithTrustedProxies(r, tt.trustedProxyCount)
			if ip != tt.expectedIP {
				t.Errorf("GetClientIPWithTrustedProxies(%d) = %s, want %s", tt.trustedProxyCount, ip, tt.expectedIP)
			}
		})
	}
}

func TestRateLimitBypassPrevention(t *testing.T) {
	// Memastikan rate limit tidak dapat dilewati dengan memanipulasi X-Forwarded-For
	// ketika middleware ClientIP tidak dipasang (default aman: pakai RemoteAddr).
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

	// Kirim 3 request dari RemoteAddr yang sama, dengan X-Forwarded-For berbeda-beda.
	// Rate limit harus dihitung dari RemoteAddr, bukan dari X-Forwarded-For.
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "127.0.0.1:8080"
		r.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('1'+i)))
		wrappedHandler(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("request %d should succeed", i+1)
		}
	}

	// Request ketiga dari RemoteAddr yang sama harus kena rate limit,
	// meskipun X-Forwarded-For berbeda.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:8080"
	r.Header.Set("X-Forwarded-For", "10.0.0.99")
	wrappedHandler(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("request should be rate limited regardless of X-Forwarded-For spoofing, got %d", w.Code)
	}
}
