package container

import "reflect"

// Provider represents a registered dependency.
type Provider struct {
	Type reflect.Type

	Lifetime Lifetime

	Factory Factory

	Instance any
}
