package container

// Freeze prevents further registrations.
func (c *Container) Freeze() {
	c.frozen = true
}

// Frozen reports whether the container is frozen.
func (c *Container) Frozen() bool {
	return c.frozen
}
