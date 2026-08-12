package middleware

import (
	"log"
	"net/http"
	"time"
)

// AccessLogger records basic HTTP request information.
type AccessLogger func(
	method string,
	path string,
	requestID string,
	status int,
	bytes int,
	duration time.Duration,
)

// AccessLog creates HTTP access logging middleware.
//
// The middleware captures:
//
//   - HTTP method
//   - request path
//   - request ID from the request context
//   - response status
//   - response bytes
//   - request duration
//
// If logger is nil, the standard library logger is used.
func AccessLog(logger AccessLogger) Middleware {
	if logger == nil {
		logger = func(
			method string,
			path string,
			requestID string,
			status int,
			bytes int,
			duration time.Duration,
		) {
			log.Printf(
				"http request method=%s path=%s request_id=%s status=%d bytes=%d duration=%s",
				method,
				path,
				requestID,
				status,
				bytes,
				duration,
			)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := NewResponseCapture(w)

			next.ServeHTTP(rec, r)

			logger(
				r.Method,
				r.URL.Path,
				RequestIDFromContext(r.Context()),
				rec.Status(),
				rec.BytesWritten(),
				time.Since(start),
			)
		})
	}
}
