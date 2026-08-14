package manifest

import (
	"fmt"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

// ResolveDependencies validates that every module and its complete
// transitive dependency closure is available in the package registry.
//
// Matching is exact by name@version.
// The input manifest is not mutated.
func ResolveDependencies(
	m Manifest,
	packages *registry.Registry,
) (Manifest, error) {
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}

	if packages == nil {
		return Manifest{}, &forgeerrors.Error{
			Code:    forgeerrors.CodeInternal,
			Message: "package registry is nil",
		}
	}

	graph, err := BuildDependencyGraph(m)
	if err != nil {
		return Manifest{}, err
	}

	// Validate every root module and its transitive dependencies.
	for _, module := range m.Modules {
		rootKey := moduleKey(module.Name, module.Version)

		if _, err := packages.Get(module.Name, module.Version); err != nil {
			return Manifest{}, dependencyNotFound(rootKey, err)
		}

		visited := make(map[string]bool)
		if err := resolveDependencyClosure(
			graph,
			packages,
			rootKey,
			visited,
		); err != nil {
			return Manifest{}, err
		}
	}

	// Preserve manifest structure and declaration order.
	resolved := m
	resolved.Modules = append([]Module(nil), m.Modules...)

	return resolved, nil
}

func resolveDependencyClosure(
	graph *DependencyGraph,
	packages *registry.Registry,
	node string,
	visited map[string]bool,
) error {
	if visited[node] {
		return nil
	}

	visited[node] = true

	for _, dependency := range graph.Nodes[node] {
		name, version, ok := splitModuleKey(dependency)
		if !ok {
			return &forgeerrors.Error{
				Code: forgeerrors.CodeInternal,
				Message: fmt.Sprintf(
					"invalid dependency identity %q",
					dependency,
				),
			}
		}

		if _, err := packages.Get(name, version); err != nil {
			return dependencyNotFound(dependency, err)
		}

		if err := resolveDependencyClosure(
			graph,
			packages,
			dependency,
			visited,
		); err != nil {
			return err
		}
	}

	return nil
}

func dependencyNotFound(
	node string,
	err error,
) error {
	return &forgeerrors.Error{
		Code: forgeerrors.CodeNotFound,
		Message: fmt.Sprintf(
			"dependency %q not found in package registry",
			node,
		),
		Err: err,
	}
}

func splitModuleKey(key string) (string, string, bool) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '@' {
			if i == 0 || i == len(key)-1 {
				return "", "", false
			}

			return key[:i], key[i+1:], true
		}
	}

	return "", "", false
}
