package container

// ResolveAs resolves a service using generics.
func ResolveAs[T any](c *Container) (T, error) {

	var value T

	err := c.Resolve(&value)

	return value, err
}
