package dim

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestGCRADecideBurstThenOnePerInterval(t *testing.T) {
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	window, limit := time.Second, 5
	interval := 200 * time.Millisecond

	var tat time.Time
	for i := 1; i <= limit; i++ {
		d := gcraDecide(tat, now, limit, window)
		if !d.allowed {
			t.Fatalf("request ke-%d dari burst %d ditolak", i, limit)
		}
		tat = d.newTAT
	}
	if tat != now.Add(window) {
		t.Errorf("TAT sesudah burst penuh = %v, harus now+window %v", tat, now.Add(window))
	}

	d := gcraDecide(tat, now, limit, window)
	if d.allowed {
		t.Fatal("request ke-6 sesaat sesudah burst harus ditolak")
	}
	if d.retryAfter != interval {
		t.Errorf("retryAfter = %v, harus satu interval %v", d.retryAfter, interval)
	}

	d = gcraDecide(tat, now.Add(interval), limit, window)
	if !d.allowed {
		t.Error("satu interval kemudian satu request harus diizinkan lagi — pulih per laju, bukan per window")
	}
	d = gcraDecide(tat, now.Add(interval-time.Microsecond), limit, window)
	if d.allowed {
		t.Error("semikrodetik sebelum interval habis masih harus ditolak")
	}
}

func TestGCRADecideIdleKeyGetsAFullBurstAgain(t *testing.T) {
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	staleTAT := now.Add(-time.Hour)

	d := gcraDecide(staleTAT, now, 5, time.Second)
	if !d.allowed || d.newTAT != now.Add(200*time.Millisecond) {
		t.Errorf("TAT lama di masa lalu harus dianggap now: allowed=%v newTAT=%v", d.allowed, d.newTAT)
	}
}

func TestGCRADecideRejectsEverythingWhenLimitIsZero(t *testing.T) {
	d := gcraDecide(time.Time{}, time.Now(), 0, time.Minute)
	if d.allowed || d.retryAfter != time.Minute {
		t.Errorf("limit 0: allowed=%v retryAfter=%v, harus ditolak dengan retryAfter=window seperti fixed window (count <= 0)", d.allowed, d.retryAfter)
	}
}

func newSQLiteGCRAStore(t *testing.T) *GCRARateLimitStore {
	t.Helper()
	db, err := NewSQLiteDatabase(DatabaseConfig{Database: ":memory:"})
	if err != nil {
		t.Fatalf("NewSQLiteDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store := NewGCRARateLimitStore(db)
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	return store
}

func newPostgresGCRAStore(t *testing.T, keys ...string) *GCRARateLimitStore {
	t.Helper()
	db := newTestPostgresDB(t)
	store := NewGCRARateLimitStore(db)
	ctx := context.Background()
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	for _, key := range keys {
		if err := db.Exec(ctx, "DELETE FROM rate_limits WHERE key = $1", key); err != nil {
			t.Fatalf("bersihkan sisa data: %v", err)
		}
	}
	return store
}

func TestGCRARateLimitStore_SQLite(t *testing.T) {
	assertGCRAStoreBehaviour(t, newSQLiteGCRAStore(t))
}

func TestGCRARateLimitStore_Postgres(t *testing.T) {
	assertGCRAStoreBehaviour(t, newPostgresGCRAStore(t, "gcra-burst", "gcra-rejected-does-not-write"))
}

// Satu jalan untuk kedua driver: burst penuh, penolakan dengan retryAfter satu
// interval, pemulihan sesudah satu interval, dan request yang ditolak tidak
// menggeser TAT.
func assertGCRAStoreBehaviour(t *testing.T, store *GCRARateLimitStore) {
	t.Helper()
	ctx := context.Background()
	window, limit := time.Second, 5
	interval := window / time.Duration(limit)

	for i := 1; i <= limit; i++ {
		allowed, retryAfter, err := store.AllowWithRetryAfter(ctx, "gcra-burst", limit, window)
		if err != nil {
			t.Fatalf("Allow %d: %v", i, err)
		}
		if !allowed || retryAfter != 0 {
			t.Fatalf("request ke-%d dari burst: allowed=%v retryAfter=%v", i, allowed, retryAfter)
		}
	}

	allowed, retryAfter, err := store.AllowWithRetryAfter(ctx, "gcra-burst", limit, window)
	if err != nil {
		t.Fatalf("Allow %d: %v", limit+1, err)
	}
	if allowed {
		t.Fatalf("request ke-%d sesaat sesudah burst harus ditolak", limit+1)
	}
	if retryAfter <= 0 || retryAfter > interval {
		t.Errorf("retryAfter = %v, harus di antara 0 dan satu interval %v — bukan window penuh", retryAfter, interval)
	}

	// Request yang ditolak tidak boleh menulis: TAT tetap, dan satu interval kemudian
	// tepat satu request diizinkan lagi meski ditolak berkali-kali selama menunggu.
	tatBefore := readTAT(t, store, "gcra-burst")
	for i := 0; i < 20; i++ {
		if allowed, _, _ := store.AllowWithRetryAfter(ctx, "gcra-burst", limit, window); allowed {
			t.Fatalf("penolakan ke-%d justru diizinkan", i+1)
		}
	}
	if tatAfter := readTAT(t, store, "gcra-burst"); !tatAfter.Equal(tatBefore) {
		t.Errorf("TAT bergeser dari %v ke %v oleh request yang ditolak — retry selama penolakan memperpanjang penolakannya sendiri", tatBefore, tatAfter)
	}

	time.Sleep(interval + 50*time.Millisecond)

	allowed, _, err = store.AllowWithRetryAfter(ctx, "gcra-burst", limit, window)
	if err != nil {
		t.Fatalf("Allow sesudah interval: %v", err)
	}
	if !allowed {
		t.Error("satu interval kemudian harus diizinkan lagi — pulih per laju, bukan menunggu window habis")
	}
	allowed, _, err = store.AllowWithRetryAfter(ctx, "gcra-burst", limit, window)
	if err != nil {
		t.Fatalf("Allow kedua sesudah interval: %v", err)
	}
	if allowed {
		t.Error("hanya SATU request yang pulih per interval")
	}
}

func readTAT(t *testing.T, store *GCRARateLimitStore, key string) time.Time {
	t.Helper()
	var tat time.Time
	q := store.db.Rebind("SELECT expires_at FROM rate_limits WHERE key = $1")
	if err := store.db.QueryRow(context.Background(), q, key).Scan(&tat); err != nil {
		t.Fatalf("baca TAT: %v", err)
	}
	return tat
}

// Compare-and-swap diuji di tempat ia bisa kalah: banyak goroutine, satu kunci,
// kapasitas lebih kecil dari jumlah goroutine. Yang diizinkan harus tepat sebanyak
// kapasitas; lebih berarti dua penulis sama-sama menang atas TAT yang sama.
func TestGCRARateLimitStore_PostgresConcurrentBurstAdmitsExactlyTheLimit(t *testing.T) {
	store := newPostgresGCRAStore(t, "gcra-race")
	ctx := context.Background()
	const limit, goroutines = 10, 40

	var wg sync.WaitGroup
	results := make(chan bool, goroutines)
	errs := make(chan error, goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			allowed, err := store.Allow(ctx, "gcra-race", limit, time.Hour)
			if err != nil {
				errs <- err
				return
			}
			results <- allowed
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Errorf("Allow: %v", err)
	}
	admitted := 0
	for allowed := range results {
		if allowed {
			admitted++
		}
	}
	if admitted != limit {
		t.Errorf("diizinkan %d dari %d goroutine, harus tepat %d", admitted, goroutines, limit)
	}
}

func TestRateLimitMiddlewareWithGCRAStoreAdvertisesTheRealWait(t *testing.T) {
	store := newSQLiteGCRAStore(t)

	h := RateLimit(RateLimitConfig{Enabled: true, Name: "donate", PerIP: 2, ResetPeriod: time.Minute}, store)(allowAll)
	for i := 0; i < 2; i++ {
		if w := serve(h, requestFrom("10.0.0.1")); w.Code != http.StatusOK {
			t.Fatalf("request ke-%d = %d", i+1, w.Code)
		}
	}
	w := serve(h, requestFrom("10.0.0.1"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request ke-3 = %d, harus 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "30" {
		t.Errorf("Retry-After = %q, harus \"30\" (satu interval dari 2 per menit), bukan \"60\"", got)
	}
}

func ExampleNewGCRARateLimitStore() {
	var db Database // koneksi aplikasi
	_ = RateLimit(RateLimitConfig{
		Enabled: true, Name: "donate", PerIP: 100, ResetPeriod: 15 * time.Minute,
	}, NewGCRARateLimitStore(db))
	fmt.Println("100 sekaligus, lalu satu tiap 9 detik")
	// Output: 100 sekaligus, lalu satu tiap 9 detik
}
