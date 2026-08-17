package compiler

import (
	"errors"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

var (
	ErrNilCompilerApp    = errors.New("compiler app is nil")
	ErrNilCompilerModule = errors.New("compiler module is nil")
)

// Module integrates the Forge compiler into the application lifecycle.
type Module struct {
	registry *registry.Registry
	runner   *OSCommandRunner
	executor *ToolchainExecutor
	engine   *Engine
}

// NewModule creates a compiler application module
// with the default Forge compiler stack.
func NewModule() *Module {
	registryInstance := registry.New()
	runner := NewOSCommandRunner()

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		panic(err)
	}

	engine, err := NewEngine(executor)
	if err != nil {
		panic(err)
	}

	return &Module{
		registry: registryInstance,
		runner:   runner,
		executor: executor,
		engine:   engine,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "compiler"
}

// Register registers the compiler services into the application container.
func (m *Module) Register(a *app.App) error {
	if m == nil {
		return ErrNilCompilerModule
	}

	if a == nil {
		return ErrNilCompilerApp
	}

	container := a.Container()

	if err := container.RegisterSingleton(m.registry); err != nil {
		return err
	}

	if err := container.RegisterSingleton(m.runner); err != nil {
		return err
	}

	if err := container.RegisterSingleton(m.executor); err != nil {
		return err
	}

	if err := container.RegisterSingleton(m.engine); err != nil {
		return err
	}

	return nil
}

// Start starts the compiler module.
//
// The compiler currently has no background process to start.
func (m *Module) Start(a *app.App) error {
	return nil
}

// Stop stops the compiler module.
//
// The compiler currently has no background process to stop.
func (m *Module) Stop(a *app.App) error {
	return nil
}

// Registry returns the compiler package registry.
func (m *Module) Registry() *registry.Registry {
	if m == nil {
		return nil
	}

	return m.registry
}

// Runner returns the operating-system command runner.
func (m *Module) Runner() *OSCommandRunner {
	if m == nil {
		return nil
	}

	return m.runner
}

// Executor returns the compiler toolchain executor.
func (m *Module) Executor() *ToolchainExecutor {
	if m == nil {
		return nil
	}

	return m.executor
}

// Engine returns the compiler execution engine.
func (m *Module) Engine() *Engine {
	if m == nil {
		return nil
	}

	return m.engine
}
