package router

import (
	"context"
	"net/http"
)

type paramsKey struct{}

// Params contains route parameters.
type Params map[string]string

// Param returns a route parameter from the request.
//
// It returns an empty string when the parameter does not exist.
func Param(r *http.Request, name string) string {
	if r == nil {
		return ""
	}

	params, ok := r.Context().Value(paramsKey{}).(Params)
	if !ok {
		return ""
	}

	return params[name]
}

func withParams(r *http.Request, params Params) *http.Request {
	ctx := context.WithValue(r.Context(), paramsKey{}, params)
	return r.WithContext(ctx)
}
