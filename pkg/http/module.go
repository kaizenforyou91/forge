package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/router"
)

// Module integrates the HTTP host into the Forge application lifecycle.
type Module struct {
	host *Host
	err  chan error
}

// NewModule creates an HTTP application module.
func NewModule(addr string, handler http.Handler) *Module {
	return &Module{
		host: New(addr, handler),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "http"
}

// Register registers the HTTP host with the application.
func (m *Module) Register(a *app.App) error {
	return nil
}

// Start starts the HTTP server in the background.
func (m *Module) Start(a *app.App) error {
	m.err = make(chan error, 1)

	go func() {
		if err := m.host.Start(); err != nil {
			m.err <- err
		}
	}()

	select {
	case <-m.host.Ready():
		return nil

	case err := <-m.err:
		return err

	case <-time.After(time.Second):
		return errors.New("http server failed to start")
	}
}

// Stop gracefully shuts down the HTTP server.
func (m *Module) Stop(a *app.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.host.Stop(ctx)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Host returns the underlying HTTP host.
func (m *Module) Host() *Host {
	return m.host
}

// Errors returns the asynchronous server error channel.
func (m *Module) Errors() <-chan error {
	return m.err
}

// NewRouterModule creates an HTTP module backed by a Forge router.
func NewRouterModule(addr string, r *router.Router) *Module {
	return NewModule(addr, r)
}
