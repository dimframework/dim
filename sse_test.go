package dim

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ─── Setup / header ──────────────────────────────────────────────────────────

func TestSSE_SetsHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	stream, err := SSE(w, r, WithHeartbeatInterval(0))
	if err != nil {
		t.Fatalf("SSE: %v", err)
	}
	defer stream.Close()

	want := map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"X-Accel-Buffering": "no",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// bareWriter (didefinisikan di responsewriter_test.go) tidak mengimplementasikan
// http.Flusher — SSE harus menolaknya secara eksplisit, bukan panic saat Flush
// pertama kali dipanggil.
func TestSSE_RequiresFlusher(t *testing.T) {
	w := bareWriter{rec: httptest.NewRecorder()}
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := SSE(w, r)
	if !errors.Is(err, ErrStreamingUnsupported) {
		t.Errorf("err = %v, want ErrStreamingUnsupported", err)
	}
}

// ─── Framing ─────────────────────────────────────────────────────────────────

func TestSSE_Send_Framing(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	stream, err := SSE(w, r, WithHeartbeatInterval(0))
	if err != nil {
		t.Fatalf("SSE: %v", err)
	}
	defer stream.Close()

	if err := stream.Send(SSEEvent{ID: "1", Event: "update", Data: "baris1\nbaris2"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := stream.Send(SSEEvent{Data: "tanpa-id"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := "id: 1\nevent: update\ndata: baris1\ndata: baris2\n\ndata: tanpa-id\n\n"
	if got := w.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// ─── Last-Event-ID ───────────────────────────────────────────────────────────

func TestLastEventID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Last-Event-ID", "42")
	if got := LastEventID(r); got != "42" {
		t.Errorf("LastEventID = %q, want %q", got, "42")
	}

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := LastEventID(r2); got != "" {
		t.Errorf("LastEventID missing = %q, want empty", got)
	}
}

// ─── Deteksi klien pergi ─────────────────────────────────────────────────────

func TestSSE_SendAfterClientGone(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	stream, err := SSE(w, r, WithHeartbeatInterval(0))
	if err != nil {
		t.Fatalf("SSE: %v", err)
	}
	defer stream.Close()

	cancel()

	if err := stream.Send(SSEEvent{Data: "x"}); err == nil {
		t.Error("Send setelah klien pergi seharusnya mengembalikan error")
	}

	select {
	case <-stream.Done():
	default:
		t.Error("Done() seharusnya closed setelah request context selesai")
	}
}

// ─── Ctx wiring ──────────────────────────────────────────────────────────────

func TestCtx_SSE(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	c := Of(w, r)

	stream, err := c.SSE(WithHeartbeatInterval(0))
	if err != nil {
		t.Fatalf("Ctx.SSE: %v", err)
	}
	defer stream.Close()
	if err := stream.Send(SSEEvent{Data: "halo"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(w.Body.String(), "data: halo\n\n") {
		t.Errorf("body = %q, want event ditulis", w.Body.String())
	}
}

func TestCtx_LastEventID(t *testing.T) {
	w, r := newCtxRequest(http.MethodGet, "/", "")
	r.Header.Set("Last-Event-ID", "7")
	c := Of(w, r)
	if got := c.LastEventID(); got != "7" {
		t.Errorf("LastEventID = %q, want %q", got, "7")
	}
}

// ─── End-to-end lewat koneksi nyata ──────────────────────────────────────────

// Regresi issue #24 bagian 1: WriteTimeout server adalah tenggat mutlak atas
// seluruh response, bukan per-tulisan — tanpa SetWriteDeadline(zero), event
// kedua ini akan gagal ditulis begitu WriteTimeout lewat, berapa pun event
// yang sudah terkirim sebelumnya.
func TestSSE_DisablesServerWriteTimeout(t *testing.T) {
	release := make(chan struct{})
	sendErr := make(chan error, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		stream, err := SSE(w, req, WithHeartbeatInterval(0))
		if err != nil {
			sendErr <- err
			return
		}
		defer stream.Close()
		if err := stream.Send(SSEEvent{Data: "awal"}); err != nil {
			sendErr <- err
			return
		}
		<-release
		sendErr <- stream.Send(SSEEvent{Data: "akhir"})
	})

	srv := httptest.NewUnstartedServer(handler)
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Lewati WriteTimeout sebelum event kedua ditulis.
	time.Sleep(300 * time.Millisecond)
	close(release)

	select {
	case err := <-sendErr:
		if err != nil {
			t.Fatalf("Send event kedua gagal setelah WriteTimeout lewat: %v — write deadline tidak ternonaktifkan", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler tidak selesai")
	}

	buf := make([]byte, 256)
	var body strings.Builder
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		n, err := resp.Body.Read(buf)
		body.Write(buf[:n])
		if strings.Contains(body.String(), "akhir") {
			break
		}
		if err != nil {
			break
		}
	}

	if !strings.Contains(body.String(), "akhir") {
		t.Errorf("body = %q, tidak memuat event kedua — koneksi terputus di WriteTimeout", body.String())
	}
}

func TestSSE_Heartbeat(t *testing.T) {
	r := NewRouter()
	r.Get("/hb", func(w http.ResponseWriter, req *http.Request) {
		stream, err := SSE(w, req, WithHeartbeatInterval(20*time.Millisecond))
		if err != nil {
			t.Errorf("SSE: %v", err)
			return
		}
		defer stream.Close()
		<-stream.Done()
	})
	r.Build()

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/hb")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	got := make(chan string, 1)
	go func() {
		buf := make([]byte, 128)
		n, _ := resp.Body.Read(buf)
		got <- string(buf[:n])
	}()

	select {
	case chunk := <-got:
		if !strings.Contains(chunk, ": keep-alive") {
			t.Errorf("chunk pertama = %q, want heartbeat comment", chunk)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tidak ada heartbeat diterima")
	}
}
