package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStandardPipelineOrder(t *testing.T) {
	var order []string

	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		if requestID == "" {
			t.Fatal("expected access log request ID")
		}

		order = append(order, "access-log")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")

		if RequestIDFromContext(r.Context()) == "" {
			t.Fatal("expected request ID in context")
		}

		w.WriteHeader(http.StatusOK)
	})

	pipeline := Standard(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	pipeline.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	expected := []string{
		"handler",
		"access-log",
	}

	if len(order) != len(expected) {
		t.Fatalf("unexpected pipeline length: %#v", order)
	}

	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf(
				"unexpected pipeline order at index %d: got %q want %q; full order=%#v",
				i,
				order[i],
				expected[i],
				order,
			)
		}
	}
}

func TestStandardHandlerAddsRequestID(t *testing.T) {
	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
	}

	handler := StandardHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if RequestIDFromContext(r.Context()) == "" {
				t.Fatal("expected request ID in context")
			}

			w.WriteHeader(http.StatusOK)
		}),
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestStandardHandlerRecoversPanic(t *testing.T) {
	logger := func(
		method string,
		path string,
		requestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
	}

	handler := StandardHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("forge test panic")
		}),
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			rec.Code,
		)
	}

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request ID to survive recovery")
	}
}

func TestStandardHandlerPreservesExistingRequestID(t *testing.T) {
	const requestID = "forge-standard-test-id"

	logger := func(
		method string,
		path string,
		gotRequestID string,
		status int,
		bytes int,
		duration time.Duration,
	) {
		if gotRequestID != requestID {
			t.Errorf(
				"expected logger request ID %q, got %q",
				requestID,
				gotRequestID,
			)
		}
	}

	handler := StandardHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := RequestIDFromContext(r.Context()); got != requestID {
				t.Fatalf(
					"expected request ID %q, got %q",
					requestID,
					got,
				)
			}

			w.WriteHeader(http.StatusOK)
		}),
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", requestID)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != requestID {
		t.Fatalf(
			"expected response request ID %q, got %q",
			requestID,
			got,
		)
	}
}
