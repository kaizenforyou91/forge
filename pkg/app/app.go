package app

import (
	"context"

	"github.com/kaizenforyou91/forge/pkg/container"
)

// App is the root application host.
type App struct {
	container *container.Container

	modules []Module

	runtime *Runtime

	started bool
}

// New creates a new application host.
func New() *App {

	ctx, cancel := context.WithCancel(context.Background())

	return &App{

		container: container.New(),

		modules: make([]Module, 0),

		runtime: &Runtime{

			context: ctx,

			cancel: cancel,
		},
	}
}

// Container returns the DI container.
func (a *App) Container() *container.Container {

	return a.container

}

// Started reports whether the application has started.
func (a *App) Started() bool {

	return a.started

}

func (a *App) Modules() []Module {

	return a.modules

}

func (a *App) Context() context.Context {

	return a.runtime.context

}

func (a *App) Cancel() {

	a.runtime.cancel()

}
