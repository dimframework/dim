package dim

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrStreamingUnsupported dikembalikan oleh SSE ketika http.ResponseWriter
// yang diberikan tidak mengimplementasikan http.Flusher, sehingga event tidak
// bisa didorong ke klien segera setelah ditulis.
var ErrStreamingUnsupported = errors.New("dim: response writer tidak mendukung http.Flusher, SSE tidak dapat dibuka")

// SSEEvent adalah satu peristiwa Server-Sent Events.
//
// ID kosong berarti event ini tidak diberi baris id: — peramban tidak akan
// memperbarui Last-Event-ID miliknya, jadi reconnect berikutnya tidak akan
// meminta ulang event ini lewat header Last-Event-ID. Event kosong berarti
// event generik, ditangkap oleh EventSource.onmessage alih-alih
// addEventListener(nama, ...). Data yang memuat newline dipecah otomatis
// menjadi beberapa baris data: sesuai spesifikasi SSE.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
}

// SSEConfig mengatur perilaku stream yang dibuka SSE.
type SSEConfig struct {
	// HeartbeatInterval adalah jeda antar baris komentar keep-alive (": ...\n\n")
	// yang dikirim otomatis supaya proxy/load balancer di depan tidak menutup
	// koneksi yang terlihat diam. Default 15 detik. Bernilai <= 0 mematikan
	// heartbeat sama sekali.
	HeartbeatInterval time.Duration
}

// SSEOption mengubah SSEConfig saat membuka stream lewat SSE.
type SSEOption func(*SSEConfig)

// WithHeartbeatInterval mengganti jeda heartbeat bawaan (15 detik).
// Interval <= 0 mematikan heartbeat.
func WithHeartbeatInterval(d time.Duration) SSEOption {
	return func(c *SSEConfig) { c.HeartbeatInterval = d }
}

// LastEventID mengambil header Last-Event-ID dari request.
//
// Header ini dikirim otomatis oleh EventSource peramban saat menyambung ulang
// setelah koneksi putus, berisi ID event terakhir yang berhasil diterima.
// dim tidak menyimpan atau mengirim ulang riwayat event dengan sendirinya —
// hanya handler yang tahu di mana riwayat itu tersimpan dan bagaimana
// mengambilnya kembali, jadi nilainya diserahkan sebagai string biasa untuk
// dipakai handler sebelum event baru dikirim.
func LastEventID(r *http.Request) string {
	return r.Header.Get("Last-Event-ID")
}

// SSEWriter menulis Server-Sent Events ke satu koneksi klien.
// Aman dipanggil dari beberapa goroutine sekaligus.
type SSEWriter struct {
	mu       sync.Mutex
	w        http.ResponseWriter
	flusher  http.Flusher
	ctx      context.Context
	stop     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// SSE membuka event-stream di w dan mengurus seluruh bagian mekanisnya:
//
//   - Header: Content-Type: text/event-stream, Cache-Control: no-cache,
//     Connection: keep-alive, dan X-Accel-Buffering: no (wajib di belakang
//     nginx — tanpanya nginx menahan buffer sampai penuh dan stream terlihat
//     mati).
//   - Write deadline: WriteTimeout server (default 10 detik, atau
//     SERVER_WRITE_TIMEOUT bila disetel) adalah tenggat mutlak atas SELURUH
//     response, bukan per-tulisan. Dibiarkan apa adanya, setiap SSE mati begitu
//     tenggat itu lewat, berapa pun event yang sudah terkirim — dan karena
//     EventSource menyambung ulang sendiri, dari sisi pengguna stream-nya
//     "jalan", hanya saja putus-sambung tiap 10/30 detik selamanya. SSE
//     menonaktifkannya lewat http.ResponseController.SetWriteDeadline(zero
//     time), terjangkau karena wrapResponseWriter mempertahankan Unwrap().
//   - Heartbeat: baris komentar ": ...\n\n" berkala (lihat SSEConfig.HeartbeatInterval)
//     supaya perantara tidak menutup koneksi yang diam. Berjalan di goroutine
//     sendiri, sebab itu handler WAJIB memanggil defer stream.Close() —
//     net/http mulai menutup response begitu handler return, dan tanpa
//     Close() menunggu goroutine heartbeat berhenti lebih dulu, keduanya
//     bisa menulis ke response secara bersamaan.
//   - Deteksi putus: goroutine heartbeat berhenti sendiri saat r.Context()
//     selesai. SSEWriter.Done mengekspos channel yang sama untuk dipakai
//     handler menghentikan goroutine pengirimnya sendiri.
//
// Membaca Last-Event-ID dan mengirim ulang event yang terlewat tetap tugas
// handler — panggil dim.LastEventID(r) sebelum event baru dikirim, sebab
// hanya handler yang tahu bagaimana mengambil ulang riwayat yang hilang.
//
// Mengembalikan ErrStreamingUnsupported bila w bukan http.Flusher.
//
// Example:
//
//	stream, err := dim.SSE(w, r)
//	if err != nil {
//	    dim.InternalServerError(w, "streaming tidak didukung")
//	    return
//	}
//	defer stream.Close()
//
//	for {
//	    select {
//	    case <-stream.Done():
//	        return
//	    case evt := <-events:
//	        if err := stream.Send(evt); err != nil {
//	            return
//	        }
//	    }
//	}
func SSE(w http.ResponseWriter, r *http.Request, opts ...SSEOption) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrStreamingUnsupported
	}

	cfg := SSEConfig{HeartbeatInterval: 15 * time.Second}
	for _, opt := range opts {
		opt(&cfg)
	}

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return nil, fmt.Errorf("dim: gagal menonaktifkan write deadline untuk SSE: %w", err)
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sw := &SSEWriter{w: w, flusher: flusher, ctx: r.Context(), stop: make(chan struct{})}

	if cfg.HeartbeatInterval > 0 {
		sw.wg.Add(1)
		go sw.heartbeat(cfg.HeartbeatInterval)
	}

	return sw, nil
}

// Send menulis satu event ke stream dan langsung flush.
// Mengembalikan error bila klien sudah pergi (request context selesai) atau
// penulisan gagal.
func (sw *SSEWriter) Send(event SSEEvent) error {
	select {
	case <-sw.ctx.Done():
		return sw.ctx.Err()
	default:
	}

	var b strings.Builder
	if event.ID != "" {
		fmt.Fprintf(&b, "id: %s\n", event.ID)
	}
	if event.Event != "" {
		fmt.Fprintf(&b, "event: %s\n", event.Event)
	}
	for _, line := range strings.Split(event.Data, "\n") {
		fmt.Fprintf(&b, "data: %s\n", line)
	}
	b.WriteString("\n")

	sw.mu.Lock()
	defer sw.mu.Unlock()

	if _, err := io.WriteString(sw.w, b.String()); err != nil {
		return err
	}
	sw.flusher.Flush()
	return nil
}

// Done mengembalikan channel yang closed saat klien memutus koneksi
// (r.Context() milik request selesai). Pakai dalam select bersama sumber
// event handler untuk menghentikan goroutine pengirim saat peramban pergi:
//
//	for {
//	    select {
//	    case <-stream.Done():
//	        return
//	    case evt := <-events:
//	        stream.Send(evt)
//	    }
//	}
func (sw *SSEWriter) Done() <-chan struct{} {
	return sw.ctx.Done()
}

// Close menghentikan goroutine heartbeat dan menunggunya keluar sepenuhnya.
//
// WAJIB dipanggil (biasanya lewat defer, segera setelah SSE berhasil) sebelum
// handler return. net/http mulai menutup dan menulis penutup response tepat
// setelah handler kembali — kalau goroutine heartbeat masih berjalan saat itu
// terjadi, keduanya menulis ke response secara bersamaan, dan itu race yang
// nyata (bukan cuma teoretis: race detector menangkapnya di test streaming).
// Aman dipanggil berkali-kali dan aman dipanggil meski heartbeat mati
// (HeartbeatInterval <= 0).
func (sw *SSEWriter) Close() {
	sw.stopOnce.Do(func() { close(sw.stop) })
	sw.wg.Wait()
}

func (sw *SSEWriter) heartbeat(interval time.Duration) {
	defer sw.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-sw.stop:
			return
		case <-sw.ctx.Done():
			return
		case <-ticker.C:
			sw.mu.Lock()
			_, err := io.WriteString(sw.w, ": keep-alive\n\n")
			if err == nil {
				sw.flusher.Flush()
			}
			sw.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}
