package app

// Module represents an application module.
type Module interface {
	Name() string

	Register(*App) error

	Lifecycle
}
