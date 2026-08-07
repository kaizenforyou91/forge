package container

// Lifetime defines how long a service instance lives.
type Lifetime int

const (

	// Singleton creates only one instance.
	Singleton Lifetime = iota

	// Transient creates a new instance every Resolve().
	Transient

	// Scoped creates one instance per scope.
	// (implemented in next sprint)
	Scoped
)
