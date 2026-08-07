package app

// Lifecycle represents application lifecycle.
type Lifecycle interface {
	Start(*App) error

	Stop(*App) error
}
