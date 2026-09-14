package agent

import (
	"context"
	"sync"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

const maxSequenceSteps = 8

type sequenceStepKind uint8

const (
	sequenceText sequenceStepKind = iota + 1
	sequenceAuthorizedTool
)

// SequenceStep is an immutable literal request with caller-fixed authority.
// Provider implementations remain trusted caller-owned references. No Run is
// constructed until execution, and output never becomes another step's input.
type SequenceStep struct {
	kind         sequenceStepKind
	request      ai.Request
	timeout      time.Duration
	provider     ai.Provider
	roundTripper AuthorizedToolRoundTripper
	authority    *tool.Authority
}

func (s SequenceStep) String() string   { return "agent sequence step (data redacted)" }
func (s SequenceStep) GoString() string { return s.String() }

// NewTextSequenceStep validates without executing or retaining an executor.
func NewTextSequenceStep(provider ai.Provider, request ai.Request, timeout time.Duration) (SequenceStep, error) {
	step := SequenceStep{kind: sequenceText, request: request, timeout: timeout, provider: provider}
	if err := step.validate(); err != nil {
		return SequenceStep{}, err
	}
	return step, nil
}

// NewAuthorizedToolSequenceStep preserves the existing immutable authority.
// Construction performs no provider or handler work.
func NewAuthorizedToolSequenceStep(roundTripper AuthorizedToolRoundTripper, request ai.Request, authority *tool.Authority, timeout time.Duration) (SequenceStep, error) {
	step := SequenceStep{kind: sequenceAuthorizedTool, request: request, timeout: timeout, roundTripper: roundTripper, authority: authority}
	if err := step.validate(); err != nil {
		return SequenceStep{}, err
	}
	return step, nil
}

func (s SequenceStep) validate() error {
	if err := s.request.Validate(); err != nil {
		return err
	}
	if err := ai.ValidateTimeout(s.timeout); err != nil {
		return err
	}
	switch s.kind {
	case sequenceText:
		if s.roundTripper != nil || s.authority != nil {
			return ErrInvalidSequence
		}
		_, err := ai.NewExecutor(s.provider, s.timeout)
		return err
	case sequenceAuthorizedTool:
		if s.provider != nil || nilRoundTripper(s.roundTripper) || s.authority == nil || s.authority.Executor() == nil || len(s.authority.Definitions()) == 0 {
			return ai.ErrInvalidRequest
		}
		return nil
	default:
		return ErrInvalidSequence
	}
}

// SequenceResult exposes only the final child result and bounded completion
// metadata. Final.Usage is not sequence aggregate usage. Failure returns zero.
type SequenceResult struct {
	Final          ai.Result
	CompletedSteps int
}

// Sequence owns 1..8 literal steps synchronously on the caller goroutine.
// Copies share one single-use claim. It has no host admission, workers, handoff,
// persistence, or aggregate usage. Providers must finish owned work on return.
type Sequence struct {
	core *sequenceState
}

type sequenceState struct {
	mu             sync.Mutex
	initialized    bool
	state          State
	done           chan struct{}
	steps          []SequenceStep
	overallTimeout time.Duration
	active         *Run
	cancel         context.CancelFunc
	canceled       bool
}

// NewSequence snapshots the declaration slice and request values without I/O.
func NewSequence(steps []SequenceStep, overallTimeout time.Duration) (*Sequence, error) {
	if len(steps) < 1 || len(steps) > maxSequenceSteps {
		return nil, ErrInvalidSequence
	}
	if err := ai.ValidateTimeout(overallTimeout); err != nil {
		return nil, err
	}
	for _, step := range steps {
		if err := step.validate(); err != nil {
			return nil, err
		}
	}
	return &Sequence{core: &sequenceState{
		initialized: true, state: StateReady, done: make(chan struct{}),
		steps: append([]SequenceStep(nil), steps...), overallTimeout: overallTimeout,
	}}, nil
}

func (s Sequence) String() string   { return "agent sequence (data redacted)" }
func (s Sequence) GoString() string { return s.String() }

// Execute claims once, creates one overall deadline, and executes fresh child
// Runs in declaration order. Nil context does not consume; pre-cancellation does.
func (s *Sequence) Execute(ctx context.Context) (SequenceResult, error) {
	if ctx == nil {
		return SequenceResult{}, ai.ErrInvalidRequest
	}
	if s == nil || s.core == nil {
		return SequenceResult{}, ErrInvalidSequence
	}
	c := s.core
	c.mu.Lock()
	if !c.validLocked() {
		c.mu.Unlock()
		return SequenceResult{}, ErrInvalidSequence
	}
	if c.state != StateReady {
		c.mu.Unlock()
		return SequenceResult{}, ErrSequenceConsumed
	}
	operationCtx, cancel := context.WithTimeout(ctx, c.overallTimeout)
	c.cancel, c.state = cancel, StateRunning
	count := len(c.steps)
	c.mu.Unlock()

	finished := false
	defer func() {
		// No recover: the original panic propagates unchanged. The Sequence's
		// bookkeeping still completes, independently of Run's panic behavior.
		if !finished {
			c.mu.Lock()
			c.finishLocked(StateFailed)
			c.mu.Unlock()
		}
		cancel()
	}()
	result, err := c.executeSteps(operationCtx, count)
	result, err = c.complete(operationCtx, result, err)
	finished = true
	return result, err
}

func (c *sequenceState) executeSteps(ctx context.Context, count int) (SequenceResult, error) {
	var final ai.Result
	for i := range count {
		// Discard the previous result before constructing another literal Run.
		final = ai.Result{}
		run, err := c.nextRun(ctx, i)
		if err != nil {
			return SequenceResult{}, err
		}
		final, err = run.Execute(ctx)
		c.mu.Lock()
		c.active = nil
		c.mu.Unlock()
		if err != nil {
			return SequenceResult{}, err
		}
	}
	return SequenceResult{Final: final, CompletedSteps: count}, nil
}

// nextRun linearizes cancellation and step admission under one lock. The
// constructors perform validation only. Cancellation after admission propagates
// through ctx before child provider work; no independent host entry is used.
func (c *sequenceState) nextRun(ctx context.Context, index int) (*Run, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.canceled {
		return nil, context.Canceled
	}
	step := c.steps[index]
	var run *Run
	var err error
	switch step.kind {
	case sequenceText:
		run, err = NewRun(step.provider, step.request, step.timeout)
	case sequenceAuthorizedTool:
		run, err = NewAuthorizedToolRun(step.roundTripper, step.request, step.authority, step.timeout)
	default:
		return nil, ErrInvalidSequence
	}
	if err != nil {
		return nil, err
	}
	c.active = run
	return run, nil
}

func (c *sequenceState) complete(ctx context.Context, result SequenceResult, err error) (SequenceResult, error) {
	err = ai.SafeError(err)
	c.mu.Lock()
	defer c.mu.Unlock()
	state := StateSucceeded
	if err != nil {
		state = StateFailed
		if contextOnly(err) {
			state = StateCanceled
		}
	} else if contextErr := ctx.Err(); contextErr != nil {
		state, err = StateCanceled, contextErr
	} else if c.canceled {
		state, err = StateCanceled, context.Canceled
	}
	if err != nil {
		result = SequenceResult{}
	}
	c.finishLocked(state)
	return result, err
}

// Cancel closes Ready immediately. Running stays nonterminal until owned work
// returns/unwinds. Canceling the shared parent cancels the active Run context
// and covers inter-step gaps; it never detaches a non-cooperative provider.
func (s *Sequence) Cancel() {
	if s == nil || s.core == nil {
		return
	}
	c := s.core
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.validLocked() {
		return
	}
	switch c.state {
	case StateReady:
		c.finishLocked(StateCanceled)
	case StateRunning:
		c.canceled = true
		c.cancel()
	}
}

func (s *Sequence) State() State {
	if s == nil || s.core == nil {
		return StateUnknown
	}
	c := s.core
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.validLocked() {
		return StateUnknown
	}
	return c.state
}

// Done is stable and carries no result. An unconstructed Sequence returns nil.
func (s *Sequence) Done() <-chan struct{} {
	if s == nil || s.core == nil {
		return nil
	}
	c := s.core
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.validLocked() {
		return nil
	}
	return c.done
}

func (c *sequenceState) validLocked() bool {
	if !c.initialized || c.done == nil {
		return false
	}
	switch c.state {
	case StateReady:
		return len(c.steps) >= 1 && len(c.steps) <= maxSequenceSteps && c.cancel == nil && c.active == nil
	case StateRunning:
		return len(c.steps) >= 1 && len(c.steps) <= maxSequenceSteps && c.cancel != nil
	case StateSucceeded, StateFailed, StateCanceled:
		return c.steps == nil && c.cancel == nil && c.active == nil
	default:
		return false
	}
}

// Only Ready cancellation or the sole executing caller publishes terminal
// state. Release references before closing Done; this is not secure erasure.
func (c *sequenceState) finishLocked(state State) {
	if c.cancel != nil {
		c.cancel()
	}
	c.steps, c.active, c.cancel = nil, nil, nil
	c.state = state
	close(c.done)
}
