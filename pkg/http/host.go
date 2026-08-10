package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
)

type hostState int

const (
	hostStateIdle hostState = iota
	hostStateStarting
	hostStateRunning
	hostStateStopping
)

// Host represents the Forge HTTP host.
type Host struct {
	addr    string
	handler http.Handler

	mu        sync.Mutex
	state     hostState
	server    *http.Server
	listener  net.Listener
	ready     chan struct{}
	startDone chan struct{}
}

// New creates a new HTTP host.
func New(addr string, handler http.Handler) *Host {
	if handler == nil {
		handler = http.DefaultServeMux
	}

	return &Host{
		addr:    addr,
		handler: handler,
		ready:   make(chan struct{}),
		state:   hostStateIdle,
	}
}

// Start starts the HTTP server.
func (h *Host) Start() error {
	h.mu.Lock()
	if h.state != hostStateIdle {
		h.mu.Unlock()
		return errors.New("http host already started")
	}

	h.state = hostStateStarting
	if h.ready == nil || isClosed(h.ready) {
		h.ready = make(chan struct{})
	}
	if h.startDone == nil {
		h.startDone = make(chan struct{})
	}

	server := &http.Server{
		Addr:    h.addr,
		Handler: h.handler,
	}

	h.server = server
	h.mu.Unlock()

	listener, err := net.Listen("tcp", h.addr)
	if err != nil {
		h.mu.Lock()
		if h.state == hostStateStarting {
			h.state = hostStateIdle
			h.server = nil
		}
		close(h.startDone)
		h.startDone = nil
		h.mu.Unlock()
		return err
	}

	h.mu.Lock()
	if h.state != hostStateStarting {
		h.mu.Unlock()
		_ = listener.Close()
		return errors.New("http host already started")
	}

	h.listener = listener
	h.state = hostStateRunning
	close(h.ready)
	close(h.startDone)
	h.startDone = nil
	h.mu.Unlock()

	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	h.mu.Lock()
	if h.state == hostStateRunning {
		h.listener = nil
		h.server = nil
		h.state = hostStateIdle
		if h.ready != nil && isClosed(h.ready) {
			h.ready = make(chan struct{})
		}
	}
	h.mu.Unlock()

	return err
}

// Stop gracefully shuts down the HTTP server.
func (h *Host) Stop(ctx context.Context) error {
	for {
		h.mu.Lock()
		switch h.state {
		case hostStateIdle:
			if h.ready != nil && isClosed(h.ready) {
				h.ready = make(chan struct{})
			}
			h.mu.Unlock()
			return nil
		case hostStateStarting:
			startDone := h.startDone
			h.mu.Unlock()

			select {
			case <-startDone:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		case hostStateRunning:
			server := h.server
			h.state = hostStateStopping
			h.mu.Unlock()

			err := server.Shutdown(ctx)
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}

			h.mu.Lock()
			if h.server == server {
				h.server = nil
				h.listener = nil
				h.state = hostStateIdle
				if h.ready != nil && isClosed(h.ready) {
					h.ready = make(chan struct{})
				}
			}
			h.mu.Unlock()

			return err
		case hostStateStopping:
			server := h.server
			h.mu.Unlock()

			if server == nil {
				return nil
			}

			err := server.Shutdown(ctx)
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			return err
		}
	}
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Addr returns the configured server address.
func (h *Host) Addr() string {
	return h.addr
}

// Handler returns the underlying HTTP handler.
func (h *Host) Handler() http.Handler {
	return h.handler
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
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.ready
}
