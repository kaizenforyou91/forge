package container

import "reflect"

// resolveRecursive resolves a type recursively.
func (c *Container) resolveRecursive(t reflect.Type) (reflect.Value, error) {

	// already registered

	if provider, ok := c.services[t]; ok {

		return reflect.ValueOf(provider.Instance), nil

	}

	// constructor available?

	constructor, ok := c.constructors[t]

	if !ok {

		return reflect.Value{}, ErrServiceNotFound

	}

	// Resolve constructor arguments recursively
	args := make([]reflect.Value, 0, len(constructor.Input))

	for _, dep := range constructor.Input {

		value, err := c.resolveRecursive(dep)
		if err != nil {
			return reflect.Value{}, err
		}

		args = append(args, value)
	}

	// Call constructor
	results := constructor.Factory.Call(args)

	instance := results[0]

	// Store as singleton
	c.services[t] = Provider{
		Type:     t,
		Instance: instance.Interface(),
		Lifetime: Singleton,
	}

	return instance, nil
}
