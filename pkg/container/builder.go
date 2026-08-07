package container

// Builder builds a dependency injection container.
type Builder struct {
	container *Container
}

// NewBuilder creates a new builder.
func NewBuilder() *Builder {
	return &Builder{
		container: New(),
	}
}

// Build returns the constructed container.
func (b *Builder) Build() *Container {
	return b.container
}

// Singleton registers a singleton service.
func (b *Builder) Singleton(service any) *Builder {

	_ = b.container.RegisterSingleton(service)

	return b
}

// Transient registers a transient service.
func (b *Builder) Transient(service any) *Builder {

	_ = b.container.RegisterTransient(service)

	return b
}

// Constructor registers a constructor.
func (b *Builder) Constructor(fn any) *Builder {

	_ = b.container.RegisterConstructor(fn)

	return b
}

// Module registers an application module.
func (b *Builder) Module(m Module) *Builder {

	m.Register(b)

	return b
}
