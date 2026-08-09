package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
