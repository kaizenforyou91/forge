package manifest

import (
	"fmt"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// DependencyGraph represents module dependency relationships.
//
// Nodes are keyed by exact module identity: name@version.
// Each node contains its direct dependency identities in declaration order.
type DependencyGraph struct {
	Nodes map[string][]string
}

// BuildDependencyGraph builds a dependency graph from a manifest.
//
// Structural validation is performed before graph construction.
func BuildDependencyGraph(m Manifest) (*DependencyGraph, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	graph := &DependencyGraph{
		Nodes: make(map[string][]string, len(m.Modules)),
	}

	// Register every module as a graph node first.
	for _, module := range m.Modules {
		key := moduleKey(module.Name, module.Version)

		graph.Nodes[key] = make([]string, 0, len(module.Dependencies))
	}

	// Register dependency edges.
	for _, module := range m.Modules {
		source := moduleKey(module.Name, module.Version)

		for _, dependency := range module.Dependencies {
			target := moduleKey(dependency.Name, dependency.Version)

			if _, exists := graph.Nodes[target]; !exists {
				return nil, &forgeerrors.Error{
					Code: forgeerrors.CodeNotFound,
					Message: fmt.Sprintf(
						"module %q dependency %q not found",
						source,
						target,
					),
				}
			}

			graph.Nodes[source] = append(graph.Nodes[source], target)
		}
	}

	if err := graph.Validate(); err != nil {
		return nil, err
	}

	return graph, nil
}

// Dependencies returns direct dependencies of a module.
//
// The returned slice is an independent snapshot.
func (g *DependencyGraph) Dependencies(name, version string) []string {
	if g == nil || g.Nodes == nil {
		return nil
	}

	key := moduleKey(name, version)

	dependencies := g.Nodes[key]
	result := make([]string, len(dependencies))
	copy(result, dependencies)

	return result
}

// Validate checks the graph for circular dependencies.
func (g *DependencyGraph) Validate() error {
	if g == nil {
		return &forgeerrors.Error{
			Code:    forgeerrors.CodeInternal,
			Message: "dependency graph is nil",
		}
	}

	visited := make(map[string]bool, len(g.Nodes))
	stack := make(map[string]bool, len(g.Nodes))

	for node := range g.Nodes {
		if visited[node] {
			continue
		}

		if err := g.visit(node, visited, stack); err != nil {
			return err
		}
	}

	return nil
}

func (g *DependencyGraph) visit(
	node string,
	visited map[string]bool,
	stack map[string]bool,
) error {
	visited[node] = true
	stack[node] = true

	for _, dependency := range g.Nodes[node] {
		if !visited[dependency] {
			if err := g.visit(dependency, visited, stack); err != nil {
				return err
			}

			continue
		}

		if stack[dependency] {
			return &forgeerrors.Error{
				Code: forgeerrors.CodeInvalidManifest,
				Message: fmt.Sprintf(
					"circular module dependency detected: %s -> %s",
					node,
					dependency,
				),
			}
		}
	}

	stack[node] = false

	return nil
}

func moduleKey(name, version string) string {
	return name + "@" + version
}
