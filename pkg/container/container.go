package container

import "reflect"

// Container stores registered services.
type Container struct {
	services map[reflect.Type]Provider

	constructors map[reflect.Type]Constructor

	graph *Graph

	frozen bool
}

// New creates a new dependency injection container.
func New() *Container {

	return &Container{

		services: make(map[reflect.Type]Provider),

		constructors: make(map[reflect.Type]Constructor),

		graph: NewGraph(),
	}
}

// Register registers a singleton service.
//
// This method is kept for backward compatibility.
func (c *Container) Register(service any) error {
	return c.RegisterSingleton(service)
}

// RegisterSingleton registers a singleton service.
func (c *Container) RegisterSingleton(service any) error {
	return c.register(service, Singleton)
}

// RegisterTransient registers a transient service.
func (c *Container) RegisterTransient(service any) error {
	return c.register(service, Transient)
}

// register stores a service with the specified lifetime.
func (c *Container) register(service any, lifetime Lifetime) error {

	if c.frozen {
		return ErrContainerFrozen
	}

	t := reflect.TypeOf(service)

	c.services[t] = Provider{
		Type:     t,
		Instance: service,
		Lifetime: lifetime,
	}

	return nil
}

// Count returns the number of registered services.
func (c *Container) Count() int {
	return len(c.services)
}

// Resolve retrieves a registered service into target.
func (c *Container) Resolve(target any) error {

	v := reflect.ValueOf(target)

	if v.Kind() != reflect.Pointer || v.IsNil() {
		return ErrInvalidTarget
	}

	t := v.Elem().Type()

	provider, ok := c.services[t]
	if !ok {

		value, err := c.resolveRecursive(t)
		if err != nil {
			return err
		}

		v.Elem().Set(value)

		return nil
	}

	switch provider.Lifetime {

	case Singleton:
		v.Elem().Set(reflect.ValueOf(provider.Instance))

	case Transient:
		if provider.Factory != nil {
			instance := provider.Factory()
			v.Elem().Set(reflect.ValueOf(instance))
		} else {
			v.Elem().Set(reflect.ValueOf(provider.Instance))
		}

	default:
		v.Elem().Set(reflect.ValueOf(provider.Instance))
	}

	return nil
}

// RegisterFactory registers a singleton factory.
func (c *Container) RegisterFactory(factory Factory) error {

	return c.registerFactory(factory, Singleton)

}

// RegisterTransientFactory registers a transient factory.
func (c *Container) RegisterTransientFactory(factory Factory) error {

	return c.registerFactory(factory, Transient)

}

// RegisterConstructor registers a constructor function.
func (c *Container) RegisterConstructor(fn any) error {

	if c.frozen {
		return ErrContainerFrozen
	}

	constructor, err := ParseConstructor(fn)
	if err != nil {
		return err
	}

	c.constructors[constructor.Output] = constructor

	c.graph.Add(constructor)

	return nil
}

// ConstructorCount returns the number of registered constructors.
func (c *Container) ConstructorCount() int {
	return len(c.constructors)
}
func (c *Container) registerFactory(factory Factory, lifetime Lifetime) error {

	if c.frozen {
		return ErrContainerFrozen
	}

	instance := factory()

	t := reflect.TypeOf(instance)

	c.services[t] = Provider{
		Type:     t,
		Instance: instance,
		Factory:  factory,
		Lifetime: lifetime,
	}

	return nil
}
