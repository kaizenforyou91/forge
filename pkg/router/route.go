package router

import "net/http"

// Route represents one registered HTTP route.
type Route struct {
	Method  string
	Path    string
	Handler http.Handler
}
