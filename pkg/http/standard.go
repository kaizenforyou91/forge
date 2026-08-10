package http

import (
	"net/http"

	"github.com/kaizenforyou91/forge/pkg/middleware"
	"github.com/kaizenforyou91/forge/pkg/router"
)

// NewStandardModule creates an HTTP module using Forge's canonical
// middleware pipeline.
//
// The standard pipeline is:
//
//	RequestID -> AccessLog -> Recovery -> Handler
//
// Additional middleware may be supplied after the standard pipeline.
// This keeps the Forge defaults consistent while allowing application-
// specific extensions.
func NewStandardModule(
	addr string,
	handler http.Handler,
	logger middleware.AccessLogger,
	middlewares ...middleware.Middleware,
) *Module {
	standard := middleware.Standard(logger)

	all := make([]middleware.Middleware, 0, 1+len(middlewares))
	all = append(all, standard)
	all = append(all, middlewares...)

	return NewModule(addr, handler, all...)
}

// NewStandardRouterModule creates an HTTP module backed by a Forge router
// using the canonical Forge middleware pipeline.
func NewStandardRouterModule(
	addr string,
	r *router.Router,
	logger middleware.AccessLogger,
	middlewares ...middleware.Middleware,
) *Module {
	return NewStandardModule(addr, r, logger, middlewares...)
}
