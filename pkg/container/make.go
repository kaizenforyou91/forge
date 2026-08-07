package container

// Make creates or resolves a service using generics.
func Make[T any](c *Container) (T, error) {
	return ResolveAs[T](c)
}
