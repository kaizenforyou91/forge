// Package agent owns one explicit text AI operation, without tools or persistence.
package agent

import (
	"context"
	"sync"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
)

// Run is an internal, single-use operation owner. Copies share the same claim
// and completion state. Execute is synchronous; Run starts no worker goroutine.
// Providers must honor context and finish owned work before returning.
type Run struct {
	core *runState
}

type runState struct {
	mu          sync.Mutex
	initialized bool // Set only after successful construction; zero/incomplete state denies all work.
	state       State
	done        chan struct{}
	request     ai.Request
	executor    *ai.Executor
	cancel      context.CancelFunc
	canceled    bool
}

// NewRun snapshots the bounded request and reuses AI validation/execution.
// Construction performs no provider I/O, credential lookup, or background work.
func NewRun(provider ai.Provider, request ai.Request, timeout time.Duration) (*Run, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	executor, err := ai.NewExecutor(provider, timeout)
	if err != nil {
		return nil, err
	}
	return &Run{core: &runState{
		initialized: true, state: StateReady, done: make(chan struct{}), request: request, executor: executor,
	}}, nil
}

// String and GoString redact both pointer and value formatting, without reading
// mutable state. Neither exposes the retained request or provider configuration.
func (r Run) String() string   { return "agent run (data redacted)" }
func (r Run) GoString() string { return r.String() }

// Execute atomically consumes Ready, then delegates once on the caller's
// goroutine. A nil context does not consume the Run; a pre-canceled non-nil
// context does, with ai.Executor preventing the provider call.
func (r *Run) Execute(ctx context.Context) (ai.Result, error) {
	if ctx == nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	if r == nil || r.core == nil {
		return ai.Result{}, ErrInvalidRun
	}
	s := r.core
	s.mu.Lock()
	if !s.validLocked() {
		s.mu.Unlock()
		return ai.Result{}, ErrInvalidRun
	}
	if s.state != StateReady {
		s.mu.Unlock()
		return ai.Result{}, ErrRunConsumed
	}
	operationCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.state = StateRunning
	executor, request := s.executor, s.request
	s.mu.Unlock()
	defer cancel()

	result, err := executor.Execute(operationCtx, request)
	return s.complete(operationCtx, result, err)
}

// complete linearizes terminal publication against explicit Cancel. A context
// checked here is authoritative at this decision point, not after publication.
func (s *runState) complete(ctx context.Context, result ai.Result, err error) (ai.Result, error) {
	err = ai.SafeError(err)
	s.mu.Lock()
	defer s.mu.Unlock()
	state := StateSucceeded
	if err != nil {
		state = StateFailed
		if contextOnly(err) {
			state = StateCanceled
		}
	} else if contextErr := ctx.Err(); contextErr != nil {
		state, err = StateCanceled, contextErr
	} else if s.canceled {
		state, err = StateCanceled, context.Canceled
	}
	if err != nil {
		result = ai.Result{}
	}
	s.finishLocked(state)
	return result, err
}

// contextOnly walks only the already-sanitized tree. Matching one context
// category via errors.Is would incorrectly accept mixed provider failures.
func contextOnly(err error) bool {
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !contextOnly(child) {
				return false
			}
		}
		return true
	}
	return false
}

// Cancel never initiates execution. Running cancellation leaves state and Done
// nonterminal until delegated work returns. Terminal cancellation is a no-op.
func (r *Run) Cancel() {
	if r == nil || r.core == nil {
		return
	}
	s := r.core
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validLocked() {
		return
	}
	switch s.state {
	case StateReady:
		s.finishLocked(StateCanceled)
	case StateRunning:
		s.canceled = true
		s.cancel()
	}
}

func (r *Run) State() State {
	if r == nil || r.core == nil {
		return StateUnknown
	}
	s := r.core
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validLocked() {
		return StateUnknown
	}
	return s.state
}

// Done returns the stable completion channel, or nil for an unconstructed Run.
// Terminal state is published before closure; no result is sent on the channel.
func (r *Run) Done() <-chan struct{} {
	if r == nil || r.core == nil {
		return nil
	}
	s := r.core
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validLocked() {
		return nil
	}
	return s.done
}

func (s *runState) validLocked() bool {
	if !s.initialized || s.done == nil {
		return false
	}
	switch s.state {
	case StateReady:
		return s.executor != nil && s.cancel == nil
	case StateRunning:
		return s.executor != nil && s.cancel != nil
	case StateSucceeded, StateFailed, StateCanceled:
		return s.executor == nil && s.cancel == nil && s.request == (ai.Request{})
	default:
		return false
	}
}

// Called only for Ready cancellation or by the sole executing caller. Release
// references before exposing completion; this is not secure memory erasure.
func (s *runState) finishLocked(state State) {
	s.state = state
	if s.cancel != nil {
		s.cancel()
	}
	s.request = ai.Request{}
	s.executor = nil
	s.cancel = nil
	close(s.done)
}
