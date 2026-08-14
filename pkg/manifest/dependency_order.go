package manifest

import (
	"fmt"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// ResolveDependencyOrder returns a deterministic dependency-first order
// for all modules declared by the manifest.
//
// Dependencies are emitted before the modules that depend on them.
// Manifest declaration order is used as the deterministic tie-breaker.
func ResolveDependencyOrder(m Manifest) ([]string, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	graph, err := BuildDependencyGraph(m)
	if err != nil {
		return nil, err
	}

	order := make([]string, 0, len(m.Modules))
	visited := make(map[string]bool, len(m.Modules))
	visiting := make(map[string]bool, len(m.Modules))

	for _, module := range m.Modules {
		key := moduleKey(module.Name, module.Version)

		if err := graph.visitOrder(
			key,
			visited,
			visiting,
			&order,
		); err != nil {
			return nil, err
		}
	}

	return order, nil
}

func (g *DependencyGraph) visitOrder(
	node string,
	visited map[string]bool,
	visiting map[string]bool,
	order *[]string,
) error {
	if visited[node] {
		return nil
	}

	if visiting[node] {
		return &forgeerrors.Error{
			Code: forgeerrors.CodeInvalidManifest,
			Message: fmt.Sprintf(
				"circular module dependency detected at %s",
				node,
			),
		}
	}

	visiting[node] = true

	for _, dependency := range g.Nodes[node] {
		if err := g.visitOrder(
			dependency,
			visited,
			visiting,
			order,
		); err != nil {
			return err
		}
	}

	delete(visiting, node)
	visited[node] = true
	*order = append(*order, node)

	return nil
}
