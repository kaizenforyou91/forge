package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDGeneratesID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())

		if id == "" {
			t.Fatal("expected request ID in context")
		}

		if r.Header.Get("X-Request-ID") != "" {
			t.Fatal("request header should not be modified")
		}

		w.WriteHeader(http.StatusOK)
	}))

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

func TestRequestIDPreservesExistingID(t *testing.T) {
	const requestID = "forge-test-request-id"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != requestID {
			t.Fatalf("expected request ID %q, got %q", requestID, got)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", requestID)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != requestID {
		t.Fatalf("expected response request ID %q, got %q", requestID, got)
	}
}

func TestRequestIDUnavailableFromEmptyContext(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty request ID, got %q", got)
	}
}

func TestRequestIDGeneratesDifferentIDs(t *testing.T) {
	var firstID string
	var secondID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())

		if firstID == "" {
			firstID = id
		} else {
			secondID = id
		}

		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	handler.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)

	if firstID == "" {
		t.Fatal("expected first request ID")
	}

	if secondID == "" {
		t.Fatal("expected second request ID")
	}

	if firstID == secondID {
		t.Fatal("expected different request IDs")
	}
}

func TestRequestIDDoesNotModifyRequestHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Request-ID"); got != "" {
			t.Fatalf("expected request header to remain empty, got %q", got)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
