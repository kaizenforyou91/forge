package container

// Factory creates a service instance.
//
// Factory is used by the dependency injection container
// to lazily construct services.
type Factory func() any
