package app

import (
	"context"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/container"
)

// App is the root application host.
type App struct {
	mu        sync.RWMutex
	container *container.Container
	modules   []Module
	runtime   *Runtime
}

// New creates a new application host.
func New() *App {
	return &App{
		container: container.New(),
		modules:   make([]Module, 0),
		runtime:   newRuntime(),
	}
}

// Container returns the DI container.
func (a *App) Container() *container.Container {
	return a.container
}

// Started reports whether the application has started.
func (a *App) Started() bool {
	return a.runtime.IsRunning()
}

// Modules returns the registered application modules.
func (a *App) Modules() []Module {
	a.mu.RLock()
	defer a.mu.RUnlock()

	modules := make([]Module, len(a.modules))
	copy(modules, a.modules)

	return modules
}

// Context returns the application runtime context.
func (a *App) Context() context.Context {
	return a.runtime.Context()
}

// Cancel cancels the application runtime context.
func (a *App) Cancel() {
	a.runtime.Cancel()
}
