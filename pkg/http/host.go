package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
)

// Host represents the Forge HTTP host.
type Host struct {
	server *http.Server

	mu       sync.Mutex
	listener net.Listener
	ready    chan struct{}
	started  bool
}

// New creates a new HTTP host.
func New(addr string, handler http.Handler) *Host {
	if handler == nil {
		handler = http.DefaultServeMux
	}

	return &Host{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		ready: make(chan struct{}),
	}
}

// Start starts the HTTP server.
func (h *Host) Start() error {
	h.mu.Lock()

	if h.started {
		h.mu.Unlock()
		return errors.New("http host already started")
	}

	h.started = true

	h.mu.Unlock()

	listener, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		h.mu.Lock()
		h.started = false
		h.mu.Unlock()

		return err
	}

	h.mu.Lock()
	h.listener = listener
	close(h.ready)
	h.mu.Unlock()

	err = h.server.Serve(listener)

	h.mu.Lock()
	h.listener = nil
	h.started = false
	h.mu.Unlock()

	if err == http.ErrServerClosed {
		return nil
	}

	return err
}

// Stop gracefully shuts down the HTTP server.
func (h *Host) Stop(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// Addr returns the configured server address.
func (h *Host) Addr() string {
	return h.server.Addr
}

// Handler returns the underlying HTTP handler.
func (h *Host) Handler() http.Handler {
	return h.server.Handler
}

// ListenerAddr returns the actual listener address.
func (h *Host) ListenerAddr() net.Addr {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener == nil {
		return nil
	}

	return h.listener.Addr()
}

// Listener returns the active listener.
func (h *Host) Listener() net.Listener {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.listener
}

// Ready returns a channel that is closed when the listener is ready.
func (h *Host) Ready() <-chan struct{} {
	return h.ready
}
