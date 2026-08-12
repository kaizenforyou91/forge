package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kaizenforyou91/forge/pkg/logger"
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

// AccessLogEntry represents a single HTTP access log event.
//
// The fields are deterministic and designed for structured logging.
type AccessLogEntry struct {
	Method    string
	Path      string
	RequestID string
	Status    int
	Bytes     int
	Duration  time.Duration
}

// Fields returns structured logging fields for the access log entry.
//
// The field order is deterministic and preserves duplicate keys if used.
func (e AccessLogEntry) Fields() []logger.Field {
	return []logger.Field{
		{Key: "method", Value: e.Method},
		{Key: "path", Value: e.Path},
		{Key: "request_id", Value: e.RequestID},
		{Key: "status", Value: e.Status},
		{Key: "bytes", Value: e.Bytes},
		{Key: "duration", Value: e.Duration},
	}
}

// Message returns the default textual access log representation.
func (e AccessLogEntry) Message() string {
	return fmt.Sprintf(
		"http request method=%s path=%s request_id=%s status=%d bytes=%d duration=%s",
		e.Method,
		e.Path,
		e.RequestID,
		e.Status,
		e.Bytes,
		e.Duration,
	)
}

func newAccessLogEntry(r *http.Request, rec *ResponseCapture, duration time.Duration) AccessLogEntry {
	return AccessLogEntry{
		Method:    r.Method,
		Path:      r.URL.Path,
		RequestID: RequestIDFromContext(r.Context()),
		Status:    rec.Status(),
		Bytes:     rec.BytesWritten(),
		Duration:  duration,
	}
}

func isNilLoggerContract(l logger.Contract) bool {
	return l == nil
}

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

// AccessLogWithLogger creates HTTP access logging middleware using a logger.Contract.
//
// If the contract supports StructuredContract, the access event is emitted with
// structured fields. Otherwise the entry falls back to a plain Info log.
//
// If logger is nil, the standard library logger is used.
func AccessLogWithLogger(l logger.Contract) Middleware {
	if isNilLoggerContract(l) {
		return AccessLog(nil)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := NewResponseCapture(w)

			next.ServeHTTP(rec, r)

			entry := newAccessLogEntry(r, rec, time.Since(start))

			if sc, ok := l.(logger.StructuredContract); ok {
				sc.InfoFields("http request", entry.Fields()...)
				return
			}

			l.Info(entry.Message())
		})
	}
}
