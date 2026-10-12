package dim

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func allowAll(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func requestFrom(ip string) *http.Request {
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = ip + ":8080"
	return r
}

func serve(h HandlerFunc, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

// Regresi #1316 di Halo: dua RateLimit berambang berbeda di atas satu store berbagi
// kunci "ip:<alamat>", sehingga 100 donasi dari satu kantor mengunci login (ambang 5)
// rekan sekantor. Name memisahkan embernya.
func TestRateLimitMiddlewareNameSeparatesBucketsOnOneStore(t *testing.T) {
	store := NewInMemoryRateLimitStore(time.Hour)
	defer store.Close()

	login := RateLimit(RateLimitConfig{Enabled: true, Name: "login", PerIP: 1, ResetPeriod: time.Hour}, store)(allowAll)
	donate := RateLimit(RateLimitConfig{Enabled: true, Name: "donate", PerIP: 100, ResetPeriod: time.Hour}, store)(allowAll)

	for i := 0; i < 100; i++ {
		if w := serve(donate, requestFrom("10.0.0.1")); w.Code != http.StatusOK {
			t.Fatalf("donasi ke-%d: %d", i+1, w.Code)
		}
	}
	if w := serve(login, requestFrom("10.0.0.1")); w.Code != http.StatusOK {
		t.Errorf("login pertama sesudah 100 donasi dari IP yang sama = %d, harus 200: ember 'login' terisi trafik 'donate'", w.Code)
	}
	if w := serve(login, requestFrom("10.0.0.1")); w.Code != http.StatusTooManyRequests {
		t.Errorf("login kedua = %d, harus 429: ambang 'login' sendiri tetap berlaku", w.Code)
	}
}

// Tanpa Name, perilaku lama dipertahankan: ember dibagi, ambang yang menolak adalah
// milik limiter yang sedang diperiksa.
func TestRateLimitMiddlewareWithoutNameStillSharesTheLegacyBucket(t *testing.T) {
	store := NewInMemoryRateLimitStore(time.Hour)
	defer store.Close()

	login := RateLimit(RateLimitConfig{Enabled: true, PerIP: 1, ResetPeriod: time.Hour}, store)(allowAll)
	donate := RateLimit(RateLimitConfig{Enabled: true, PerIP: 100, ResetPeriod: time.Hour}, store)(allowAll)

	serve(donate, requestFrom("10.0.0.1"))
	if w := serve(login, requestFrom("10.0.0.1")); w.Code != http.StatusTooManyRequests {
		t.Errorf("tanpa Name login = %d, harus 429 — kalau ini berubah, aplikasi yang mengandalkan satu ember global ikut berubah", w.Code)
	}
}

func TestRateLimiterKeysWithAndWithoutName(t *testing.T) {
	plain := NewRateLimiter(RateLimitConfig{PerIP: 1}, nil)
	named := NewRateLimiter(RateLimitConfig{PerIP: 1, Name: "login"}, nil)

	if got := plain.IPKey("1.2.3.4"); got != "ip:1.2.3.4" {
		t.Errorf("IPKey tanpa Name = %q, harus tetap bentuk lama", got)
	}
	if got := plain.UserKey("42"); got != "user:42" {
		t.Errorf("UserKey tanpa Name = %q, harus tetap bentuk lama", got)
	}
	if got := named.IPKey("1.2.3.4"); got != "ip:login:1.2.3.4" {
		t.Errorf("IPKey dengan Name = %q", got)
	}
	if got := named.UserKey("42"); got != "user:login:42" {
		t.Errorf("UserKey dengan Name = %q", got)
	}
}

func TestRateLimitMiddlewareUserBucketFollowsName(t *testing.T) {
	store := NewInMemoryRateLimitStore(time.Hour)
	defer store.Close()
	user := &TokenUser{ID: "7", Email: "a@contoh.test"}

	a := RateLimit(RateLimitConfig{Enabled: true, Name: "a", PerIP: 100, PerUser: 1, ResetPeriod: time.Hour}, store)(allowAll)
	b := RateLimit(RateLimitConfig{Enabled: true, Name: "b", PerIP: 100, PerUser: 1, ResetPeriod: time.Hour}, store)(allowAll)

	serve(a, SetUser(requestFrom("10.0.0.1"), user))
	if w := serve(b, SetUser(requestFrom("10.0.0.1"), user)); w.Code != http.StatusOK {
		t.Errorf("ember user 'b' = %d, harus 200: terisi oleh limiter 'a'", w.Code)
	}
}

func TestRateLimitMiddlewareKeyFuncLimitsPerKeyAndIgnoresIP(t *testing.T) {
	store := NewInMemoryRateLimitStore(time.Hour)
	defer store.Close()

	byAccount := RateLimit(RateLimitConfig{
		Enabled: true, Name: "profile", PerIP: 1, PerKey: 2, ResetPeriod: time.Hour,
		KeyFunc: func(r *http.Request) string { return "account:" + r.Header.Get("X-Account") },
	}, store)(allowAll)

	asAccount := func(id string) *http.Request {
		r := requestFrom("10.0.0.1")
		r.Header.Set("X-Account", id)
		return r
	}

	for i := 0; i < 2; i++ {
		if w := serve(byAccount, asAccount("A")); w.Code != http.StatusOK {
			t.Fatalf("akun A request ke-%d = %d, harus 200 (PerIP=1 harus diabaikan saat KeyFunc diisi)", i+1, w.Code)
		}
	}
	if w := serve(byAccount, asAccount("A")); w.Code != http.StatusTooManyRequests {
		t.Errorf("akun A request ke-3 = %d, harus 429 (PerKey=2)", w.Code)
	}
	if w := serve(byAccount, asAccount("B")); w.Code != http.StatusOK {
		t.Errorf("akun B dari IP yang sama = %d, harus 200: ember per kunci, bukan per IP", w.Code)
	}
}

func TestRateLimitMiddlewareKeyFuncEmptyKeyFailsClosed(t *testing.T) {
	store := NewInMemoryRateLimitStore(time.Hour)
	defer store.Close()

	reached := false
	h := RateLimit(RateLimitConfig{
		Enabled: true, PerKey: 10, ResetPeriod: time.Hour,
		KeyFunc: func(*http.Request) string { return "" },
	}, store)(func(w http.ResponseWriter, r *http.Request) { reached = true })

	w := serve(h, requestFrom("10.0.0.1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("kunci kosong = %d, harus 500: limiter dipasang sebelum penyedia kuncinya", w.Code)
	}
	if reached {
		t.Error("handler tetap dipanggil padahal kuncinya kosong — limiter jadi tidak berefek tanpa suara")
	}
}

type retryAfterStore struct {
	RateLimitStore
	retryAfter time.Duration
}

func (s retryAfterStore) AllowWithRetryAfter(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	allowed, err := s.RateLimitStore.Allow(ctx, key, limit, window)
	if err != nil || allowed {
		return allowed, 0, err
	}
	return false, s.retryAfter, nil
}

func TestRateLimitMiddlewareRetryAfterComesFromTheStoreWhenItKnows(t *testing.T) {
	store := retryAfterStore{RateLimitStore: NewInMemoryRateLimitStore(time.Hour), retryAfter: 6500 * time.Millisecond}
	defer store.Close()

	h := RateLimit(RateLimitConfig{Enabled: true, PerIP: 1, ResetPeriod: 15 * time.Minute}, store)(allowAll)
	serve(h, requestFrom("10.0.0.1"))
	w := serve(h, requestFrom("10.0.0.1"))

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("code = %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "7" {
		t.Errorf("Retry-After = %q, harus \"7\" (6,5 detik dibulatkan ke atas), bukan 900 dari ResetPeriod", got)
	}
}

func TestRateLimitMiddlewareRetryAfterFallsBackToResetPeriod(t *testing.T) {
	h := RateLimit(RateLimitConfig{Enabled: true, PerIP: 1, ResetPeriod: 15 * time.Minute})(allowAll)
	serve(h, requestFrom("10.0.0.1"))
	w := serve(h, requestFrom("10.0.0.1"))

	if got := w.Header().Get("Retry-After"); got != "900" {
		t.Errorf("Retry-After = %q, harus \"900\": store in-memory tidak tahu sisa waktunya", got)
	}
}

func TestRetryAfterSecondsNeverAdvertisesZero(t *testing.T) {
	cases := map[time.Duration]int{0: 1, 10 * time.Millisecond: 1, time.Second: 1, 1001 * time.Millisecond: 2, 90 * time.Second: 90}
	for d, want := range cases {
		if got := retryAfterSeconds(d); got != want {
			t.Errorf("retryAfterSeconds(%v) = %d, harus %d", d, got, want)
		}
	}
}

func TestRateLimitMiddlewareLogsEveryRejectionWithNameScopeAndPath(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(previous)

	h := RateLimit(RateLimitConfig{Enabled: true, Name: "login", PerIP: 1, ResetPeriod: time.Hour})(allowAll)
	r := requestFrom("10.0.0.1")
	r.URL.Path = "/api/v1/login"
	serve(h, r)
	if buf.Len() != 0 {
		t.Fatalf("request yang diizinkan tidak boleh dicatat: %s", buf.String())
	}

	serve(h, r)
	line := buf.String()
	for _, want := range []string{"rate limit terlampaui", "name=login", "scope=ip", "path=/api/v1/login", "retry_after_seconds=3600"} {
		if !strings.Contains(line, want) {
			t.Errorf("log penolakan tidak memuat %q:\n%s", want, line)
		}
	}
}
