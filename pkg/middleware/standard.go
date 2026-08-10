package middleware

import "net/http"

// Standard returns Forge's default HTTP middleware pipeline.
//
// The execution order is:
//
//	RequestID -> Recovery -> AccessLog -> Handler
//
// RequestID must execute first so downstream middleware and handlers
// can access the request correlation ID.
//
// Recovery wraps the remaining pipeline so panics from downstream
// middleware and handlers are converted into HTTP 500 responses.
//
// AccessLog runs around the downstream handler and records the final
// request outcome.
func Standard(logger AccessLogger) Middleware {
	return Chain(
		RequestID,
		Recovery,
		AccessLog(logger),
	)
}

// StandardHandler applies Forge's standard middleware pipeline to a handler.
func StandardHandler(handler http.Handler, logger AccessLogger) http.Handler {
	return Standard(logger)(handler)
}
