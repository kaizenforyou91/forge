package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseCaptureInitialState(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	if cap.Status() != http.StatusOK {
		t.Fatalf("expected initial status %d, got %d", http.StatusOK, cap.Status())
	}

	if cap.BytesWritten() != 0 {
		t.Fatalf("expected initial bytes 0, got %d", cap.BytesWritten())
	}

	if cap.WroteHeader() {
		t.Fatal("expected initial WroteHeader false")
	}
}

func TestResponseCaptureWriteHeaderCapturesStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	cap.WriteHeader(http.StatusCreated)

	if cap.Status() != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, cap.Status())
	}

	if !cap.WroteHeader() {
		t.Fatal("expected WroteHeader true after WriteHeader")
	}
}

func TestResponseCaptureRepeatedWriteHeaderDoesNotChangeStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	cap.WriteHeader(http.StatusCreated)
	cap.WriteHeader(http.StatusNoContent)

	if cap.Status() != http.StatusCreated {
		t.Fatalf("expected status %d after repeated WriteHeader, got %d", http.StatusCreated, cap.Status())
	}
}

func TestResponseCaptureImplicitStatusOKOnWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	_, err := cap.Write([]byte("ok"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	if cap.Status() != http.StatusOK {
		t.Fatalf("expected implicit status %d, got %d", http.StatusOK, cap.Status())
	}

	if cap.BytesWritten() != len("ok") {
		t.Fatalf("expected bytes %d, got %d", len("ok"), cap.BytesWritten())
	}
}

func TestResponseCaptureWriteHeaderAndWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	cap.WriteHeader(http.StatusAccepted)
	_, err := cap.Write([]byte("accepted"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	if cap.Status() != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, cap.Status())
	}

	if cap.BytesWritten() != len("accepted") {
		t.Fatalf("expected bytes %d, got %d", len("accepted"), cap.BytesWritten())
	}
}

func TestResponseCaptureMultipleWritesAccumulatesBytes(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	_, _ = cap.Write([]byte("hello"))
	_, _ = cap.Write([]byte("world"))

	if cap.BytesWritten() != len("helloworld") {
		t.Fatalf("expected accumulated bytes %d, got %d", len("helloworld"), cap.BytesWritten())
	}
}

func TestResponseCaptureZeroByteWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	_, err := cap.Write([]byte(""))
	if err != nil {
		t.Fatalf("unexpected zero-byte write error: %v", err)
	}

	if cap.BytesWritten() != 0 {
		t.Fatalf("expected zero bytes written, got %d", cap.BytesWritten())
	}

	if cap.Status() != http.StatusOK {
		t.Fatalf("expected implicit status %d on zero-byte write, got %d", http.StatusOK, cap.Status())
	}
}

func TestResponseCapturePreservesHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	header := cap.Header()
	header.Set("X-Test", "value")

	if got := rec.Header().Get("X-Test"); got != "value" {
		t.Fatalf("expected header to be forwarded, got %q", got)
	}
}

func TestResponseCaptureRecordsRecoveredPanicStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	wrapped := Recovery(handler)

	wrapped.ServeHTTP(cap, httptest.NewRequest(http.MethodGet, "/", nil))

	if cap.Status() != http.StatusInternalServerError {
		t.Fatalf("expected recovered status %d, got %d", http.StatusInternalServerError, cap.Status())
	}
}

func TestResponseCaptureSupportsResponseController(t *testing.T) {
	rec := httptest.NewRecorder()
	cap := NewResponseCapture(rec)

	controller := http.NewResponseController(cap)
	if err := controller.Flush(); err != nil {
		t.Fatalf("expected Flush to work with captured response writer, got %v", err)
	}
}

type partialWriteResponseWriter struct {
	head http.Header
	buf  bytes.Buffer
}

func (w *partialWriteResponseWriter) Header() http.Header {
	if w.head == nil {
		w.head = make(http.Header)
	}
	return w.head
}

func (w *partialWriteResponseWriter) WriteHeader(statusCode int) {
	if w.head == nil {
		w.head = make(http.Header)
	}
	w.head.Set("X-Status", http.StatusText(statusCode))
}

func (w *partialWriteResponseWriter) Write(body []byte) (int, error) {
	n := len(body) / 2
	if n == 0 && len(body) > 0 {
		n = 1
	}
	w.buf.Write(body[:n])
	return n, io.ErrUnexpectedEOF
}

func TestResponseCaptureCountsPartialWritesAndReturnsError(t *testing.T) {
	w := &partialWriteResponseWriter{}
	cap := NewResponseCapture(w)

	n, err := cap.Write([]byte("hello"))
	if err == nil {
		t.Fatal("expected write error")
	}

	if n != 2 && n != 3 {
		t.Fatalf("expected partial write count, got %d", n)
	}

	if cap.BytesWritten() != n {
		t.Fatalf("expected bytes written %d, got %d", n, cap.BytesWritten())
	}

	if cap.Status() != http.StatusOK {
		t.Fatalf("expected implicit status %d, got %d", http.StatusOK, cap.Status())
	}
}
