package router

import "net/http"

// Router is the core HTTP request router.
//
// Router intentionally depends only on the Go standard library.
// It does not depend on pkg/http or pkg/app.
type Router struct {
	routes []Route
}

// New creates a new Router.
func New() *Router {
	return &Router{
		routes: make([]Route, 0),
	}
}

// Handle registers a handler for an HTTP method and exact path.
func (r *Router) Handle(method, path string, handler http.Handler) {
	r.routes = append(r.routes, Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var best *routeMatch

	for _, route := range r.routes {
		if route.Method != req.Method {
			continue
		}

		params, score, ok := matchRoute(route, req.URL.Path)
		if !ok {
			continue
		}

		match := &routeMatch{
			route:  route,
			params: params,
			score:  score,
		}

		if best == nil || match.score > best.score {
			best = match
		}
	}

	if best == nil {
		http.NotFound(w, req)
		return
	}

	best.route.Handler.ServeHTTP(w, withParams(req, best.params))
}

// Handler returns the router as an http.Handler.
func (r *Router) Handler() http.Handler {
	return r
}

// Routes returns the registered routes.
func (r *Router) Routes() []Route {
	return append([]Route(nil), r.routes...)
}

// GET registers a GET route.
func (r *Router) GET(path string, handler http.Handler) {
	r.Handle(http.MethodGet, path, handler)
}

// POST registers a POST route.
func (r *Router) POST(path string, handler http.Handler) {
	r.Handle(http.MethodPost, path, handler)
}

// PUT registers a PUT route.
func (r *Router) PUT(path string, handler http.Handler) {
	r.Handle(http.MethodPut, path, handler)
}

// DELETE registers a DELETE route.
func (r *Router) DELETE(path string, handler http.Handler) {
	r.Handle(http.MethodDelete, path, handler)
}
