package dim

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func discardLogger() *Logger {
	return NewLoggerWithWriter(io.Discard, slog.LevelError)
}

// Regresi issue #16: membungkus ResponseWriter tanpa Unwrap dan tanpa
// meneruskan antarmuka opsional membuat SSE, streaming, upgrade WebSocket, dan
// jalur cepat sendfile lumpuh untuk SETIAP handler di balik LoggerMiddleware.
func TestLoggerMiddlewarePreservesOptionalInterfaces(t *testing.T) {
	type caps struct {
		flusher    bool
		hijacker   bool
		readerFrom bool
		rcFlushErr string
	}

	probe := func(w http.ResponseWriter) caps {
		_, f := w.(http.Flusher)
		_, h := w.(http.Hijacker)
		_, rf := w.(io.ReaderFrom)
		got := caps{flusher: f, hijacker: h, readerFrom: rf, rcFlushErr: "<nil>"}
		if err := http.NewResponseController(w).Flush(); err != nil {
			got.rcFlushErr = err.Error()
		}
		return got
	}

	var direct, wrapped caps

	r := NewRouter()
	r.Get("/direct", func(w http.ResponseWriter, _ *http.Request) { direct = probe(w) })
	r.Group("/logged", LoggerMiddleware(discardLogger())).
		Get("/wrapped", func(w http.ResponseWriter, _ *http.Request) { wrapped = probe(w) })
	r.Build()

	srv := httptest.NewServer(r)
	defer srv.Close()

	for _, path := range []string{"/direct", "/logged/wrapped"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
	}

	if direct != wrapped {
		t.Errorf("antarmuka hilang saat dibungkus:\n  tanpa  logger: %+v\n  dengan logger: %+v", direct, wrapped)
	}
	if !wrapped.flusher || !wrapped.hijacker || !wrapped.readerFrom {
		t.Errorf("writer terbungkus kehilangan antarmuka: %+v", wrapped)
	}
	if wrapped.rcFlushErr != "<nil>" {
		t.Errorf("ResponseController.Flush gagal di writer terbungkus: %s", wrapped.rcFlushErr)
	}
}

// Pembungkus tidak boleh MENAMBAH antarmuka yang tidak dimiliki writer aslinya.
// Menyediakan Hijack tanpa syarat akan membuat w.(http.Hijacker) berhasil di
// HTTP/2, mengubah kegagalan bersih menjadi error saat pemanggilan.
func TestWrapResponseWriterDoesNotFakeInterfaces(t *testing.T) {
	// httptest.ResponseRecorder punya Flush, tetapi bukan Hijacker maupun ReaderFrom.
	rec := httptest.NewRecorder()
	wrapped, _ := wrapResponseWriter(rec)

	if _, ok := wrapped.(http.Flusher); !ok {
		t.Error("Flusher seharusnya dipertahankan — ResponseRecorder memilikinya")
	}
	if _, ok := wrapped.(http.Hijacker); ok {
		t.Error("Hijacker dipalsukan — ResponseRecorder tidak memilikinya")
	}
	if _, ok := wrapped.(io.ReaderFrom); ok {
		t.Error("ReaderFrom dipalsukan — ResponseRecorder tidak memilikinya")
	}

	// Writer paling polos: tidak boleh mendapat antarmuka apa pun.
	bare, _ := wrapResponseWriter(bareWriter{httptest.NewRecorder()})
	if _, ok := bare.(http.Flusher); ok {
		t.Error("Flusher dipalsukan pada writer tanpa kemampuan apa pun")
	}
	if _, ok := bare.(http.Hijacker); ok {
		t.Error("Hijacker dipalsukan pada writer tanpa kemampuan apa pun")
	}
}

// bareWriter menyembunyikan seluruh antarmuka opsional milik writer di dalamnya.
type bareWriter struct{ rec *httptest.ResponseRecorder }

func (b bareWriter) Header() http.Header         { return b.rec.Header() }
func (b bareWriter) Write(p []byte) (int, error) { return b.rec.Write(p) }
func (b bareWriter) WriteHeader(code int)        { b.rec.WriteHeader(code) }

// Status tetap tertangkap benar setelah pembungkusan.
func TestLoggerMiddlewareCapturesStatus(t *testing.T) {
	for _, want := range []int{http.StatusOK, http.StatusAccepted, http.StatusNotFound} {
		_, rw := wrapResponseWriter(httptest.NewRecorder())
		var w http.ResponseWriter = rw
		if want != http.StatusOK {
			w.WriteHeader(want)
		}
		fmt.Fprint(w, "body")

		if rw.statusCode != want {
			t.Errorf("statusCode = %d, want %d", rw.statusCode, want)
		}
	}
}

// SSE end-to-end: klien harus menerima potongan sebelum handler selesai.
// Tanpa Flush yang berfungsi, respons tertahan di buffer sampai handler kembali.
func TestLoggerMiddlewareAllowsStreaming(t *testing.T) {
	release := make(chan struct{})

	r := NewRouter()
	r.Use(LoggerMiddleware(discardLogger()))
	r.Get("/sse", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: pertama\n\n")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush: %v", err)
		}
		<-release // handler sengaja ditahan setelah flush
		fmt.Fprint(w, "data: kedua\n\n")
	})
	r.Build()

	srv := httptest.NewServer(r)
	defer srv.Close()
	defer close(release)

	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Potongan pertama harus tiba meski handler masih berjalan.
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := resp.Body.Read(buf)
		done <- string(buf[:n])
	}()

	select {
	case got := <-done:
		if !strings.Contains(got, "pertama") {
			t.Errorf("potongan pertama = %q, tidak memuat \"pertama\"", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tidak ada data sebelum handler selesai — streaming tertahan di buffer")
	}
}

// Hijack harus berfungsi di balik middleware — inilah yang dipakai gorilla/websocket
// dan coder/websocket untuk mengambil alih koneksi.
func TestLoggerMiddlewareAllowsHijack(t *testing.T) {
	r := NewRouter()
	r.Use(LoggerMiddleware(discardLogger()))
	r.Get("/ws", func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("writer bukan http.Hijacker — upgrade WebSocket akan ditolak pustaka")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Errorf("Hijack: %v", err)
			return
		}
		defer conn.Close()
		buf.WriteString("HTTP/1.1 101 Switching Protocols\r\n\r\nterambil")
		buf.Flush()
	})
	r.Build()

	srv := httptest.NewServer(r)
	defer srv.Close()

	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	fmt.Fprint(conn, "GET /ws HTTP/1.1\r\nHost: x\r\n\r\n")
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	got, err := io.ReadAll(bufio.NewReader(conn))
	if err != nil {
		t.Fatalf("baca: %v", err)
	}
	if !strings.Contains(string(got), "terambil") {
		t.Errorf("respons = %q, tidak memuat penanda hijack", got)
	}
}
