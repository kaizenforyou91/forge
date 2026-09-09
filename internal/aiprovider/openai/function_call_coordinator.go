package openai

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

var (
	ErrFunctionCallReplay   = errors.New("openai: function call replay denied")
	ErrFunctionCallCapacity = errors.New("openai: function call capacity reached")
)

const maxCoordinatedFunctionCalls = 1024

// FunctionCallCoordinator provides in-memory at-most-one execution attempt per
// call_id within one coordinator instance. A new coordinator or process restart
// resets protection; it is not durable, global, or an exactly-once guarantee.
// Different IDs may execute concurrently through external callers; handler
// thread-safety remains caller-owned via C1. No background work is created.
// The zero value denies all calls. Value copies share the same claim state.
type FunctionCallCoordinator struct {
	state *functionCallCoordinationState
}

type functionCallCoordinationState struct {
	executor *tool.Executor
	mu       sync.Mutex
	claimed  map[string]struct{}
}

// NewFunctionCallCoordinator retains the non-nil executor reference and starts
// an empty bounded claim set. C1 alone validates execution authority; no handler
// or catalog is extracted. Claimed IDs are never evicted, even after failure.
func NewFunctionCallCoordinator(executor *tool.Executor) (*FunctionCallCoordinator, error) {
	if executor == nil {
		return nil, ai.ErrInvalidRequest
	}
	return &FunctionCallCoordinator{state: &functionCallCoordinationState{
		executor: executor,
		claimed:  make(map[string]struct{}),
	}}, nil
}

// String and GoString disclose no IDs, counts, arguments, authority, or output.
func (c FunctionCallCoordinator) String() string {
	return "OpenAI function-call coordinator (state redacted)"
}

func (c FunctionCallCoordinator) GoString() string { return c.String() }

// Execute validates caller context and admitted B1 correlation before claiming
// call_id. Invalid decisions and pre-canceled contexts consume no capacity.
// The claim precedes C1 and is permanent for this coordinator, including C1
// denial, cancellation after claim, handler failure/panic, or C2 failure.
//
// C1 re-authorizes the exact admitted call. Only its successful result is paired
// internally with this same decision for C2 serialization; callers cannot supply
// a mismatched ExecutionResult. Success does not mean an output was sent or a
// model continued, nor does it give side effects durable transaction semantics.
func (c *FunctionCallCoordinator) Execute(ctx context.Context, decision FunctionCallDecision) (FunctionCallOutput, error) {
	if c == nil || c.state == nil || c.state.executor == nil || c.state.claimed == nil || ctx == nil {
		return FunctionCallOutput{}, ai.ErrInvalidRequest
	}
	if err := ctx.Err(); err != nil {
		return FunctionCallOutput{}, err
	}
	itemID, callID, ok := decision.Correlation()
	if !ok || !validFunctionCallID(itemID) || !validFunctionCallID(callID) {
		return FunctionCallOutput{}, ai.ErrInvalidRequest
	}
	if err := c.state.claim(callID); err != nil {
		return FunctionCallOutput{}, err
	}
	result, err := c.state.executor.Execute(ctx, decision.Admission())
	if err != nil {
		return FunctionCallOutput{}, err
	}
	return BuildFunctionCallOutput(decision, result)
}

func (s *functionCallCoordinationState) claim(callID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Replay takes precedence at capacity so old IDs never become executable.
	if _, exists := s.claimed[callID]; exists {
		return ErrFunctionCallReplay
	}
	if len(s.claimed) >= maxCoordinatedFunctionCalls {
		return ErrFunctionCallCapacity
	}
	// Only bounded, owned identity is retained, never decision/result payloads.
	s.claimed[strings.Clone(callID)] = struct{}{}
	return nil
}
