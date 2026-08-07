package container

import "reflect"

// DependencyGraph represents dependency relationships.
type DependencyGraph struct {
	Nodes map[string][]string
}

// Graph stores constructor dependency relationships.
type Graph struct {
	nodes map[reflect.Type][]reflect.Type
}

// NewGraph creates a dependency graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[reflect.Type][]reflect.Type),
	}
}

// Add registers a constructor dependency.
func (g *Graph) Add(c Constructor) {

	g.nodes[c.Output] = c.Input

}

// Dependencies returns constructor dependencies.
func (g *Graph) Dependencies(t reflect.Type) []reflect.Type {

	return g.nodes[t]

}
