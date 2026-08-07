package container

import "reflect"

// resolveArguments resolves constructor dependencies.
func (c *Container) resolveArguments(constructor Constructor) ([]reflect.Value, error) {

	args := make([]reflect.Value, 0, len(constructor.Input))

	for _, t := range constructor.Input {

		provider, ok := c.services[t]
		if !ok {
			return nil, ErrServiceNotFound
		}

		args = append(args, reflect.ValueOf(provider.Instance))
	}

	return args, nil
}
