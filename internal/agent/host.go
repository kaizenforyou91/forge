package agent

import (
	"context"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/app"
)

// RunHost binds explicit Run or Sequence execution to an application's lifetime.
// Register and Start never execute work. Copies share ownership, like Run.
// Install with App.Add before starting the App; lifecycle methods are Module
// callbacks. Callers retain the host; no DI or global registry is created.
type RunHost struct {
	core *hostState
}

var _ app.Module = (*RunHost)(nil)

type hostPhase uint8

const (
	hostReady hostPhase = iota
	hostRunning
	hostStopping
	hostStopped
)

type hostState struct {
	mu       sync.Mutex
	phase    hostPhase
	app      *app.App
	appCtx   context.Context
	active   map[*hostExecution]struct{}
	wg       sync.WaitGroup
	stopDone chan struct{}
}

// Each admitted call gets its own identity, even when owners share a claim.
// The only retained capability is cancellation, never execution or results.
type hostExecution struct{ cancel func() }

func NewRunHost() *RunHost {
	return &RunHost{core: &hostState{phase: hostReady, active: make(map[*hostExecution]struct{})}}
}

func (*RunHost) Name() string      { return "agent-run-host" }
func (RunHost) String() string     { return "agent run host (data redacted)" }
func (h RunHost) GoString() string { return h.String() }

func (h *RunHost) valid() bool {
	return h != nil && h.core != nil && h.core.active != nil
}

// Register binds once, rejecting even repeated same-App registration so Add
// cannot append this module twice. The zero App is not a usable application.
func (h *RunHost) Register(a *app.App) error {
	if !h.valid() || a == nil || a.Container() == nil {
		return ErrInvalidHost
	}
	s := h.core
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app != nil || s.phase != hostReady {
		return ErrInvalidHost
	}
	s.app = a
	return nil
}

// Start snapshots this generation's context; App itself may still be Starting.
// Execute separately requires App.Started before admitting any operation.
func (h *RunHost) Start(a *app.App) error {
	if !h.valid() || a == nil {
		return ErrInvalidHost
	}
	s := h.core
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app != a || (s.phase != hostReady && s.phase != hostStopped) || len(s.active) != 0 {
		return ErrInvalidHost
	}
	ctx := a.Context()
	if ctx == nil || ctx.Err() != nil {
		return ErrHostNotRunning
	}
	s.appCtx, s.phase = ctx, hostRunning
	return nil
}

// Execute admits one call while both host and App are Running. Run remains the
// sole claim/operation owner; this method adds no timeout, result validation,
// retry, or worker goroutine. Caller values and deadlines remain authoritative.
func (h *RunHost) Execute(ctx context.Context, run *Run) (ai.Result, error) {
	linkedCtx, release, err := h.admit(ctx, run.Cancel)
	if err != nil {
		return ai.Result{}, err
	}
	defer release()
	return run.Execute(linkedCtx)
}

// ExecuteSequence owns the entire synchronous Sequence call as one admission,
// including transitions and panic unwind. Sequence alone owns child execution,
// timeouts, handoff, accounting, and result/error classification.
func (h *RunHost) ExecuteSequence(ctx context.Context, sequence *Sequence) (SequenceResult, error) {
	linkedCtx, release, err := h.admit(ctx, sequence.Cancel)
	if err != nil {
		return SequenceResult{}, err
	}
	defer release()
	return sequence.Execute(linkedCtx)
}

// admit links lifetimes and registers one cancellation owner. It does not
// execute work. The narrow entry points must defer the returned release once.
func (h *RunHost) admit(ctx context.Context, cancelOwner func()) (context.Context, func(), error) {
	if ctx == nil {
		return nil, nil, ai.ErrInvalidRequest
	}
	if !h.valid() {
		return nil, nil, ErrInvalidHost
	}
	s := h.core
	s.mu.Lock()
	if s.app == nil {
		s.mu.Unlock()
		return nil, nil, ErrInvalidHost
	}
	if s.phase != hostRunning || !s.app.Started() || s.appCtx == nil || s.appCtx.Err() != nil || s.app.Context() != s.appCtx {
		s.mu.Unlock()
		return nil, nil, ErrHostNotRunning
	}
	entry := &hostExecution{cancel: cancelOwner}
	s.active[entry] = struct{}{}
	s.wg.Add(1) // Stop closes admission under this same lock before Wait.
	appCtx := s.appCtx
	s.mu.Unlock()
	releaseEntry := func() {
		s.mu.Lock()
		delete(s.active, entry)
		entry.cancel = nil
		s.mu.Unlock()
		s.wg.Done()
	}
	linked := false
	defer func() {
		// Preserve bookkeeping even if a caller's custom Context panics while
		// linking. No recovery or translation of the original panic.
		if !linked {
			releaseEntry()
		}
	}()

	linkedCtx, cancel := context.WithCancel(ctx)
	// The standard-library callback performs cancellation only, never AI work.
	unlink := context.AfterFunc(appCtx, cancel)
	if appCtx.Err() != nil {
		cancel()
	}
	linked = true
	return linkedCtx, func() {
		unlink()
		cancel()
		releaseEntry()
	}, nil
}

// Stop closes admission, cancels every active owner, then drains admitted calls.
// A non-cooperative provider can block shutdown: owned work is never detached
// or abandoned. Operation results/errors belong only to Execute callers.
func (h *RunHost) Stop(a *app.App) error {
	if !h.valid() || a == nil {
		return ErrInvalidHost
	}
	s := h.core
	s.mu.Lock()
	if s.app != a {
		s.mu.Unlock()
		return ErrInvalidHost
	}
	switch s.phase {
	case hostStopped:
		s.mu.Unlock()
		return nil
	case hostStopping:
		done := s.stopDone
		s.mu.Unlock()
		<-done
		return nil
	case hostReady, hostRunning:
		s.phase = hostStopping
	default:
		s.mu.Unlock()
		return ErrInvalidHost
	}
	s.stopDone = make(chan struct{})
	cancels := make([]func(), 0, len(s.active))
	for entry := range s.active {
		cancels = append(cancels, entry.cancel)
	}
	s.mu.Unlock()
	for i, cancel := range cancels {
		cancel()
		cancels[i] = nil
	}
	s.wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.active) != 0 {
		return ErrInvalidHost
	}
	s.appCtx = nil
	s.phase = hostStopped
	close(s.stopDone)
	s.stopDone = nil
	return nil
}
