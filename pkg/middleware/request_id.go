package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

const requestIDHeader = "X-Request-ID"

// RequestID adds a request ID to the request context and response header.
//
// If the incoming request already contains X-Request-ID, that ID is preserved.
// Otherwise, a new request ID is generated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)

		if id == "" {
			id = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		r = r.WithContext(ctx)

		w.Header().Set(requestIDHeader, id)

		next.ServeHTTP(w, r)
	})
}

// RequestIDFromContext returns the request ID stored in the context.
//
// It returns an empty string when no request ID exists.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)

	return id
}

func generateRequestID() string {
	var b [16]byte

	if _, err := rand.Read(b[:]); err != nil {
		panic("middleware: failed to generate request ID")
	}

	return hex.EncodeToString(b[:])
}
