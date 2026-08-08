package router

import "strings"

type routeMatch struct {
	route  Route
	params Params
	score  int
}

func matchRoute(route Route, path string) (Params, int, bool) {
	routeParts := splitPath(route.Path)
	pathParts := splitPath(path)

	if len(routeParts) != len(pathParts) {
		return nil, 0, false
	}

	params := make(Params)
	score := 0

	for i := range routeParts {
		pattern := routeParts[i]
		value := pathParts[i]

		if strings.HasPrefix(pattern, ":") {
			name := strings.TrimPrefix(pattern, ":")
			if name == "" {
				return nil, 0, false
			}

			params[name] = value
			continue
		}

		if pattern != value {
			return nil, 0, false
		}

		score++
	}

	return params, score, true
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return nil
	}

	path = strings.Trim(path, "/")

	if path == "" {
		return nil
	}

	return strings.Split(path, "/")
}
