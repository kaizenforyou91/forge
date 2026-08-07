package container

// Module represents a Forge module.
type Module interface {
	Register(*Builder)
}
