package dim

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		trustedProxyCount int
		xff               string
		xffLines          []string
		remoteAddr        string
		expectedIP        string
	}{
		{
			name:              "default config ignores proxy headers",
			trustedProxyCount: 0,
			xff:               "1.2.3.4",
			remoteAddr:        "10.0.0.1:8080",
			expectedIP:        "10.0.0.1",
		},
		{
			name:              "one trusted proxy resolves real client",
			trustedProxyCount: 1,
			xff:               "spoofed, 203.0.113.9",
			remoteAddr:        "10.0.0.1:8080",
			expectedIP:        "203.0.113.9",
		},
		{
			name:              "separate header lines are joined before reading",
			trustedProxyCount: 1,
			xffLines:          []string{"6.6.6.6", "203.0.113.9"},
			remoteAddr:        "10.0.0.1:8080",
			expectedIP:        "203.0.113.9",
		},
		{
			name:              "chain shorter than configured falls back to RemoteAddr",
			trustedProxyCount: 3,
			xff:               "6.6.6.6",
			remoteAddr:        "10.0.0.1:8080",
			expectedIP:        "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fromHelper, fromCtx string
			handler := ClientIP(ClientIPConfig{TrustedProxyCount: tt.trustedProxyCount})(
				func(w http.ResponseWriter, r *http.Request) {
					fromHelper = GetClientIP(r)
					fromCtx = Of(w, r).ClientIP()
				},
			)

			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			for _, line := range tt.xffLines {
				r.Header.Add("X-Forwarded-For", line)
			}

			handler(httptest.NewRecorder(), r)

			if fromHelper != tt.expectedIP {
				t.Errorf("GetClientIP = %q, want %q", fromHelper, tt.expectedIP)
			}
			// Ctx.ClientIP() harus konsisten dengan GetClientIP.
			if fromCtx != tt.expectedIP {
				t.Errorf("Ctx.ClientIP = %q, want %q", fromCtx, tt.expectedIP)
			}
		})
	}
}

// Tanpa middleware ClientIP, GetClientIP tetap memakai RemoteAddr.
func TestClientIPMiddlewareNotInstalled(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:8080"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got := GetClientIP(r); got != "10.0.0.1" {
		t.Errorf("GetClientIP tanpa middleware = %q, want %q", got, "10.0.0.1")
	}
}

// RateLimit harus memakai IP hasil resolusi middleware ClientIP, sehingga klien
// di belakang satu proxy tepercaya dibatasi per IP asli — bukan per IP proxy.
func TestRateLimitUsesResolvedClientIP(t *testing.T) {
	config := RateLimitConfig{
		Enabled:     true,
		PerIP:       2,
		PerUser:     100,
		ResetPeriod: 1 * time.Second,
	}

	handler := Chain(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
		ClientIP(ClientIPConfig{TrustedProxyCount: 1}),
		RateLimit(config),
	)

	// Semua request datang lewat proxy yang sama (RemoteAddr identik),
	// tetapi dari dua klien asli yang berbeda di posisi paling kanan XFF.
	send := func(realClient string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "10.0.0.1:8080"
		r.Header.Set("X-Forwarded-For", "spoofed, "+realClient)
		handler(w, r)
		return w.Code
	}

	for i := 0; i < 2; i++ {
		if code := send("203.0.113.9"); code != http.StatusOK {
			t.Fatalf("request %d klien A harus lolos, dapat %d", i+1, code)
		}
	}

	if code := send("203.0.113.9"); code != http.StatusTooManyRequests {
		t.Errorf("klien A harus kena rate limit, dapat %d", code)
	}

	// Klien lain di belakang proxy yang sama tidak boleh ikut terkena.
	if code := send("198.51.100.7"); code != http.StatusOK {
		t.Errorf("klien B harus punya bucket sendiri, dapat %d", code)
	}
}

// Klien tidak boleh bisa memalsukan bucket rate limit dengan memutar-mutar
// X-Forwarded-For, meskipun middleware ClientIP terpasang.
func TestRateLimitNotBypassableBehindTrustedProxy(t *testing.T) {
	config := RateLimitConfig{
		Enabled:     true,
		PerIP:       2,
		PerUser:     100,
		ResetPeriod: 1 * time.Second,
	}

	handler := Chain(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
		ClientIP(ClientIPConfig{TrustedProxyCount: 1}),
		RateLimit(config),
	)

	// Penyerang memutar entri kiri; entri paling kanan tetap ditulis proxy.
	send := func(spoofed string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "10.0.0.1:8080"
		r.Header.Set("X-Forwarded-For", spoofed+", 203.0.113.9")
		handler(w, r)
		return w.Code
	}

	send("1.1.1.1")
	send("2.2.2.2")

	if code := send("3.3.3.3"); code != http.StatusTooManyRequests {
		t.Errorf("rate limit harus tetap berlaku meski X-Forwarded-For diputar, dapat %d", code)
	}
}
