package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/logger"
)

func TestAccessLogCapturesRequest(t *testing.T) {
	var (
		gotMethod    string
		gotPath      string
		gotRequestID string
		gotStatus    int
		gotBytes     int
		gotDuration  time.Duration
	)

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		gotMethod = method
		gotPath = path
		gotRequestID = requestID
		gotStatus = status
		gotBytes = bytes
		gotDuration = duration
	}

	handler := AccessLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method POST, got %q", gotMethod)
	}

	if gotPath != "/users" {
		t.Fatalf("expected path /users, got %q", gotPath)
	}

	if gotRequestID != "" {
		t.Fatalf("expected empty request ID, got %q", gotRequestID)
	}

	if gotStatus != http.StatusCreated {
		t.Fatalf("expected logged status 201, got %d", gotStatus)
	}

	if gotBytes != len("created") {
		t.Fatalf("expected logged bytes %d, got %d", len("created"), gotBytes)
	}

	if gotDuration < 0 {
		t.Fatal("expected non-negative request duration")
	}
}

func TestAccessLogCapturesRequestID(t *testing.T) {
	var gotRequestID string

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		gotRequestID = requestID
	}

	handler := AccessLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	requestIDHandler := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()

	requestIDHandler.ServeHTTP(rec, req)

	if gotRequestID == "" {
		t.Fatal("expected request ID to be captured")
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestAccessLogDefaultsStatusToOK(t *testing.T) {
	var gotStatus int

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		gotStatus = status
	}

	handler := AccessLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if gotStatus != http.StatusOK {
		t.Fatalf("expected logged status 200, got %d", gotStatus)
	}
}

func TestAccessLogDoesNotModifyRequest(t *testing.T) {
	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
	}

	handler := AccessLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-Test"); got != "original" {
				t.Fatalf("expected request header original, got %q", got)
			}

			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Test", "original")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLogPreservesResponseBody(t *testing.T) {
	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
	}

	handler := AccessLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("forge response"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "forge response" {
		t.Fatalf("unexpected response body: %q", got)
	}
}

func TestAccessLogPreservesResponseControllerCapabilities(t *testing.T) {
	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
	}

	handler := AccessLog(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			controller := http.NewResponseController(w)

			if err := controller.Flush(); err != nil {
				t.Fatalf("expected response controller Flush to remain available, got %v", err)
			}

			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLogEntryFieldsAreDeterministic(t *testing.T) {
	entry := AccessLogEntry{
		Method:    "GET",
		Path:      "/health",
		RequestID: "abc123",
		Status:    http.StatusOK,
		Bytes:     42,
		Duration:  123 * time.Millisecond,
	}

	fields := entry.Fields()
	if len(fields) != 6 {
		t.Fatalf("expected 6 fields, got %d", len(fields))
	}

	expect := []logger.Field{
		{Key: "method", Value: "GET"},
		{Key: "path", Value: "/health"},
		{Key: "request_id", Value: "abc123"},
		{Key: "status", Value: http.StatusOK},
		{Key: "bytes", Value: 42},
		{Key: "duration", Value: 123 * time.Millisecond},
	}

	for i, field := range fields {
		if field != expect[i] {
			t.Fatalf("expected field %d %#v, got %#v", i, expect[i], field)
		}
	}
}

type testLegacyLogger struct {
	msg string
}

func (l *testLegacyLogger) Debug(msg string) {}
func (l *testLegacyLogger) Info(msg string)  { l.msg = msg }
func (l *testLegacyLogger) Warn(msg string)  {}
func (l *testLegacyLogger) Error(msg string) {}

type testStructuredLogger struct {
	msg    string
	fields []logger.Field
}

func (l *testStructuredLogger) Debug(msg string)                               {}
func (l *testStructuredLogger) Info(msg string)                                { l.msg = msg }
func (l *testStructuredLogger) Warn(msg string)                                {}
func (l *testStructuredLogger) Error(msg string)                               {}
func (l *testStructuredLogger) DebugFields(msg string, fields ...logger.Field) {}
func (l *testStructuredLogger) InfoFields(msg string, fields ...logger.Field) {
	l.msg = msg
	l.fields = append([]logger.Field(nil), fields...)
}
func (l *testStructuredLogger) WarnFields(msg string, fields ...logger.Field)  {}
func (l *testStructuredLogger) ErrorFields(msg string, fields ...logger.Field) {}

func TestAccessLogWithLoggerStructuredFields(t *testing.T) {
	structured := &testStructuredLogger{}

	handler := AccessLogWithLogger(structured)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ok"))
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	if structured.msg != "http request" {
		t.Fatalf("expected structured message 'http request', got %q", structured.msg)
	}

	if len(structured.fields) != 6 {
		t.Fatalf("expected 6 fields, got %d", len(structured.fields))
	}

	if structured.fields[0].Key != "method" || structured.fields[0].Value != "POST" {
		t.Fatalf("unexpected method field %#v", structured.fields[0])
	}

	if structured.fields[2].Key != "request_id" {
		t.Fatalf("expected request_id field, got %#v", structured.fields[2])
	}

	if structured.fields[3].Value != http.StatusAccepted {
		t.Fatalf("expected status field %d, got %#v", http.StatusAccepted, structured.fields[3].Value)
	}

	if structured.fields[4].Value != len("ok") {
		t.Fatalf("expected bytes field %d, got %#v", len("ok"), structured.fields[4].Value)
	}

	duration, ok := structured.fields[5].Value.(time.Duration)
	if !ok || duration < 0 {
		t.Fatalf("expected non-negative duration, got %#v", structured.fields[5].Value)
	}
}

func TestAccessLogWithLoggerPlainContractFallback(t *testing.T) {
	legacy := &testLegacyLogger{}

	handler := AccessLogWithLogger(legacy)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(legacy.msg, "http request") {
		t.Fatalf("expected fallback log message to contain http request, got %q", legacy.msg)
	}
}

func TestAccessLogWithLoggerNilFallback(t *testing.T) {
	old := log.Writer()
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	defer log.SetOutput(old)

	handler := AccessLogWithLogger(nil)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/nil", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if !strings.Contains(buf.String(), "http request") {
		t.Fatalf("expected stdlib fallback log message, got %q", buf.String())
	}
}

func TestAccessLogWithLoggerPanicRecovery(t *testing.T) {
	structured := &testStructuredLogger{}

	handler := Chain(RequestID, AccessLogWithLogger(structured), Recovery)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("boom")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	if structured.fields[3].Key != "status" || structured.fields[3].Value != http.StatusInternalServerError {
		t.Fatalf("expected status 500 field, got %#v", structured.fields[3])
	}

	if structured.fields[2].Key != "request_id" {
		t.Fatalf("expected request_id field, got %#v", structured.fields[2])
	}
}
