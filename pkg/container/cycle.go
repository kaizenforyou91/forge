package container

import "fmt"

// Validate checks the dependency graph for circular dependencies.
func (g *DependencyGraph) Validate() error {

	visited := make(map[string]bool)
	stack := make(map[string]bool)

	for node := range g.Nodes {

		if !visited[node] {

			if err := g.visit(node, visited, stack); err != nil {
				return err
			}

		}

	}

	return nil
}

// visit performs DFS to detect dependency cycles.
func (g *DependencyGraph) visit(
	node string,
	visited map[string]bool,
	stack map[string]bool,
) error {

	visited[node] = true
	stack[node] = true

	for _, dep := range g.Nodes[node] {

		if !visited[dep] {

			if err := g.visit(dep, visited, stack); err != nil {
				return err
			}

		} else if stack[dep] {

			return fmt.Errorf(
				"circular dependency detected: %s -> %s",
				node,
				dep,
			)

		}

	}

	stack[node] = false

	return nil
}
