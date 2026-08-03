package dim

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// responseWriter membungkus http.ResponseWriter untuk menangkap status code.
//
// Membungkus ResponseWriter menyembunyikan antarmuka opsional yang dimiliki
// writer aslinya — http.Flusher, http.Hijacker, io.ReaderFrom. Embedding tidak
// menolong: *responseWriter bukan http.Flusher sekalipun writer di dalamnya
// begitu. Akibatnya SSE, streaming, upgrade WebSocket, dan jalur cepat sendfile
// untuk file statis semuanya lumpuh.
//
// Karena itu pembungkus ini TIDAK dipakai langsung. Gunakan wrapResponseWriter,
// yang memilih varian pembungkus sesuai kemampuan writer aslinya.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
	hijacked   bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write captures writes
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Unwrap memberi http.ResponseController akses ke ResponseWriter asli, sehingga
// Flush, Hijack, SetReadDeadline, dan SetWriteDeadline tetap terjangkau handler
// meski response writer-nya terbungkus. Wajib ada sejak Go 1.20.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// markWritten menandai header sudah terkirim tanpa meneruskan WriteHeader.
// Flush dan ReadFrom sama-sama menyebabkan net/http mengirim header 200 secara
// implisit, jadi status yang tercatat harus ikut menyesuaikan.
func (rw *responseWriter) markWritten() {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
}

// --- Mixin untuk antarmuka opsional ---
//
// Mixin menyimpan *responseWriter sebagai field bernama, bukan embedded, agar
// tidak bentrok dengan *responseWriter yang di-embed varian pembungkus.

type flushMixin struct{ rw *responseWriter }

func (m flushMixin) Flush() {
	m.rw.markWritten()
	m.rw.ResponseWriter.(http.Flusher).Flush()
}

type hijackMixin struct{ rw *responseWriter }

func (m hijackMixin) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, buf, err := m.rw.ResponseWriter.(http.Hijacker).Hijack()
	if err == nil {
		m.rw.hijacked = true
	}
	return conn, buf, err
}

type readFromMixin struct{ rw *responseWriter }

func (m readFromMixin) ReadFrom(src io.Reader) (int64, error) {
	m.rw.markWritten()
	return m.rw.ResponseWriter.(io.ReaderFrom).ReadFrom(src)
}

// --- Varian pembungkus ---
//
// Satu varian per kombinasi antarmuka yang dimiliki writer asli. Pendekatan ini
// dipilih supaya pembungkus tidak pernah berbohong: menyediakan Hijack tanpa
// syarat akan membuat w.(http.Hijacker) berhasil di HTTP/2 yang tidak
// mendukungnya, sehingga kegagalan bersih berubah menjadi error saat pemanggilan.

type rwF struct {
	*responseWriter
	flushMixin
}

type rwH struct {
	*responseWriter
	hijackMixin
}

type rwR struct {
	*responseWriter
	readFromMixin
}

type rwFH struct {
	*responseWriter
	flushMixin
	hijackMixin
}

type rwFR struct {
	*responseWriter
	flushMixin
	readFromMixin
}

type rwHR struct {
	*responseWriter
	hijackMixin
	readFromMixin
}

type rwFHR struct {
	*responseWriter
	flushMixin
	hijackMixin
	readFromMixin
}

// wrapResponseWriter membungkus w untuk menangkap status code sambil
// mempertahankan antarmuka opsional yang dimilikinya.
//
// Mengembalikan writer yang harus diteruskan ke handler, beserta *responseWriter
// untuk membaca status yang tertangkap setelah handler selesai.
//
// http.Pusher sengaja tidak dipertahankan: HTTP/2 server push sudah tidak
// didukung peramban arus utama. Handler yang membutuhkannya tetap dapat
// menjangkaunya lewat Unwrap.
func wrapResponseWriter(w http.ResponseWriter) (http.ResponseWriter, *responseWriter) {
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	_, canFlush := w.(http.Flusher)
	_, canHijack := w.(http.Hijacker)
	_, canReadFrom := w.(io.ReaderFrom)

	switch {
	case canFlush && canHijack && canReadFrom:
		return rwFHR{rw, flushMixin{rw}, hijackMixin{rw}, readFromMixin{rw}}, rw
	case canFlush && canHijack:
		return rwFH{rw, flushMixin{rw}, hijackMixin{rw}}, rw
	case canFlush && canReadFrom:
		return rwFR{rw, flushMixin{rw}, readFromMixin{rw}}, rw
	case canHijack && canReadFrom:
		return rwHR{rw, hijackMixin{rw}, readFromMixin{rw}}, rw
	case canFlush:
		return rwF{rw, flushMixin{rw}}, rw
	case canHijack:
		return rwH{rw, hijackMixin{rw}}, rw
	case canReadFrom:
		return rwR{rw, readFromMixin{rw}}, rw
	default:
		return rw, rw
	}
}
