package app

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrAppStarting = errors.New("app is starting")
	ErrAppStopping = errors.New("app is stopping")
)

type runtimeState int

const (
	runtimeStateIdle runtimeState = iota
	runtimeStateStarting
	runtimeStateRunning
	runtimeStateStopping
	runtimeStateStopped
)

// Runtime represents the application runtime.
type Runtime struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	state  runtimeState
}

func newRuntime() *Runtime {
	ctx, cancel := context.WithCancel(context.Background())

	return &Runtime{
		ctx:    ctx,
		cancel: cancel,
		state:  runtimeStateIdle,
	}
}

// runtime state transitions:
//
// Idle -> Starting -> Running -> Stopping -> Stopped
//
// Restart semantics:
// Stopped -> Starting -> Running is allowed.
// Start() is idempotent when already running.
// Stop() is idempotent when already stopped.

func (r *Runtime) Context() context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ctx
}

func (r *Runtime) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runtime) BeginStart() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch r.state {
	case runtimeStateRunning:
		return false, nil
	case runtimeStateStarting:
		return false, ErrAppStarting
	case runtimeStateStopping:
		return false, ErrAppStopping
	case runtimeStateStopped:
		r.ctx, r.cancel = context.WithCancel(context.Background())
		r.state = runtimeStateStarting
		return true, nil
	case runtimeStateIdle:
		r.state = runtimeStateStarting
		return true, nil
	default:
		return false, nil
	}
}

func (r *Runtime) SetRunning() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.state = runtimeStateRunning
}

func (r *Runtime) BeginStop() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch r.state {
	case runtimeStateIdle, runtimeStateStopped:
		return false, nil
	case runtimeStateStopping:
		return false, nil
	case runtimeStateStarting:
		return false, ErrAppStarting
	case runtimeStateRunning:
		r.state = runtimeStateStopping
		return true, nil
	default:
		return false, nil
	}
}

func (r *Runtime) SetStopped() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.state = runtimeStateStopped
}

func (r *Runtime) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.state == runtimeStateRunning
}
