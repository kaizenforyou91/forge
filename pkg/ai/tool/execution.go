package tool

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidExecutor         = errors.New("ai/tool: invalid executor")
	ErrExecutionDenied         = errors.New("ai/tool: execution denied")
	ErrHandlerFailed           = errors.New("ai/tool: handler failed")
	ErrInvalidExecutionOutput  = errors.New("ai/tool: invalid execution output")
	ErrExecutionOutputTooLarge = errors.New("ai/tool: execution output too large")
)

const maxExecutionOutputBytes = 64 * 1024

// Handler is explicit caller-supplied execution authority. It must honor ctx
// and finish its own work. The executor invokes it synchronously, without a
// timeout, retry, or internal goroutine. Handler thread-safety and any mutable
// state captured by the handler remain caller-owned.
type Handler func(context.Context, Call) (string, error)

// Binding couples a handler to the exact argument contract it may execute.
type Binding struct {
	Definition Definition
	Handler    Handler
}

// Executor owns immutable handler-coupled authority, separate from Stage A
// admission. Independent concurrent Execute calls are supported; they may call
// the same handler concurrently. There is no replay or at-most-once guarantee.
// The zero value and nil receiver deny execution.
type Executor struct {
	catalog  *Catalog
	handlers map[string]Handler
}

// NewExecutor validates and snapshots definitions through NewCatalog, and
// retains the supplied handler function values. Nil/empty bindings create a
// valid deny-all executor. Callers must not mutate bindings or their parameter
// slices during construction; mutation after return cannot change authority.
func NewExecutor(bindings []Binding) (*Executor, error) {
	// Reuse Stage A's bound before allocating the temporary definition slice.
	if len(bindings) > maxTools {
		return nil, ErrInvalidExecutor
	}
	definitions := make([]Definition, len(bindings))
	for i, binding := range bindings {
		if binding.Handler == nil {
			return nil, ErrInvalidExecutor
		}
		definitions[i] = binding.Definition
	}
	// NewCatalog independently copies names, parameter slices, and fields.
	catalog, err := NewCatalog(definitions)
	if err != nil {
		return nil, ErrInvalidExecutor
	}
	handlers := make(map[string]Handler, len(bindings))
	for _, binding := range bindings {
		handlers[strings.Clone(binding.Definition.Name)] = binding.Handler
	}
	return &Executor{catalog: catalog, handlers: handlers}, nil
}

// ExecutionResult retains only owned raw UTF-8 output. Its zero value is empty
// and does not prove execution success; callers must check the returned error.
type ExecutionResult struct {
	output string
}

// Output returns the immutable output string without interpreting its contents.
func (r ExecutionResult) Output() string { return r.output }

// String and GoString expose no output, definitions, arguments, or handlers.
func (r ExecutionResult) String() string   { return "tool execution result (output redacted)" }
func (r ExecutionResult) GoString() string { return r.String() }
func (e Executor) String() string          { return "tool executor (authority redacted)" }
func (e Executor) GoString() string        { return e.String() }

// Execute requires Stage A ADMITTED/ALLOWED and independently re-admits the
// exact call against the handler-coupled catalog. ADMITTED != EXECUTED.
// A permitted dispatch calls exactly one matching handler once, directly.
// Repeating Execute may execute again; no correlation or replay state is kept.
// The caller's context is checked before invocation and after return, including
// panic recovery. Its cancellation discards output and takes precedence over
// handler failure. Other handler errors are redacted, even context-shaped ones
// when the caller context is still live. No output is retained on any error.
func (e *Executor) Execute(ctx context.Context, admission Admission) (ExecutionResult, error) {
	if e == nil || e.catalog == nil || e.handlers == nil || ctx == nil {
		return ExecutionResult{}, ErrExecutionDenied
	}
	call, ok := admission.AdmittedCall()
	if !ok {
		return ExecutionResult{}, ErrExecutionDenied
	}
	reauthorized, err := e.catalog.Admit(call)
	if err != nil {
		return ExecutionResult{}, ErrExecutionDenied
	}
	call, ok = reauthorized.AdmittedCall()
	if !ok {
		return ExecutionResult{}, ErrExecutionDenied
	}
	handler := e.handlers[call.Name]
	if handler == nil {
		return ExecutionResult{}, ErrExecutionDenied
	}
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	output, err := invokeHandler(ctx, handler, call)
	if contextErr := ctx.Err(); contextErr != nil {
		return ExecutionResult{}, contextErr
	}
	if err != nil {
		return ExecutionResult{}, ErrHandlerFailed
	}
	if len(output) > maxExecutionOutputBytes {
		return ExecutionResult{}, ErrExecutionOutputTooLarge
	}
	if !utf8.ValidString(output) {
		return ExecutionResult{}, ErrInvalidExecutionOutput
	}
	return ExecutionResult{output: strings.Clone(output)}, nil
}

// Containment is limited to recoverable handler panics, not process-fatal
// behavior. The panic value and original error are never inspected or retained.
func invokeHandler(ctx context.Context, handler Handler, call Call) (output string, err error) {
	returned := false
	defer func() {
		if !returned {
			_ = recover()
			output, err = "", ErrHandlerFailed
		}
	}()
	output, err = handler(ctx, call)
	returned = true
	if err != nil {
		return "", ErrHandlerFailed
	}
	return output, nil
}
