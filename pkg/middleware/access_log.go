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

			rec := &accessLogResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			logger(
				r.Method,
				r.URL.Path,
				RequestIDFromContext(r.Context()),
				rec.status,
				rec.bytes,
				time.Since(start),
			)
		})
	}
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *accessLogResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *accessLogResponseWriter) Write(body []byte) (int, error) {
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

func (w *accessLogResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
