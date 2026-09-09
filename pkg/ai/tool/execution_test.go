package tool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func executionDefinition() Definition {
	return Definition{Name: "private_tool", Parameters: []Parameter{{Name: "value", Type: String, Required: true}}}
}

func executionAdmission(t *testing.T) Admission {
	t.Helper()
	return assertDecision(t, mustCatalog(t, executionDefinition()), Call{
		Name: "private_tool", Arguments: []byte(" {\"value\":\"private_argument\"} \n"),
	}, Allowed)
}

func executionExecutor(t *testing.T, bindings ...Binding) *Executor {
	t.Helper()
	e, err := NewExecutor(bindings)
	if err != nil || e == nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}
	return e
}

func executionFailure(t *testing.T, e *Executor, ctx context.Context, a Admission, want error) {
	t.Helper()
	r, err := e.Execute(ctx, a)
	if err != want || r != (ExecutionResult{}) || r.Output() != "" {
		t.Fatalf("failure classification/result mismatch: %v", err)
	}
}

func TestExecutorConstructor(t *testing.T) {
	a := executionAdmission(t)
	for _, bindings := range [][]Binding{nil, {}} {
		e := executionExecutor(t, bindings...)
		executionFailure(t, e, context.Background(), a, ErrExecutionDenied)
	}
	handler := func(context.Context, Call) (string, error) { return "", nil }
	valid := Binding{Definition: executionDefinition(), Handler: handler}
	executionExecutor(t, valid)
	executionExecutor(t, valid, Binding{Definition: Definition{Name: "other"}, Handler: handler})
	tooManyTools := make([]Binding, 65)
	for i := range tooManyTools {
		tooManyTools[i] = Binding{Definition: Definition{Name: fmt.Sprintf("tool_%d", i)}, Handler: handler}
	}
	tooManyParameters := make([]Parameter, 33)
	for i := range tooManyParameters {
		tooManyParameters[i] = Parameter{Name: fmt.Sprintf("p_%d", i), Type: String}
	}
	cases := map[string][]Binding{
		"nil handler":          {{Definition: executionDefinition()}},
		"invalid name":         {{Definition: Definition{Name: "bad name"}, Handler: handler}},
		"duplicate definition": {valid, valid},
		"invalid parameter":    {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: "0bad", Type: String}}}, Handler: handler}},
		"invalid type":         {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: "value", Type: InvalidType}}}, Handler: handler}},
		"duplicate parameter":  {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: "value", Type: String}, {Name: "value", Type: Number}}}, Handler: handler}},
		"too many tools":       tooManyTools,
		"too many parameters":  {{Definition: Definition{Name: "tool", Parameters: tooManyParameters}, Handler: handler}},
	}
	for name, bindings := range cases {
		t.Run(name, func(t *testing.T) {
			e, err := NewExecutor(bindings)
			if e != nil || err != ErrInvalidExecutor {
				t.Fatal("invalid constructor exposed authority or wrong error")
			}
		})
	}
}

func TestExecutorAdmissionGate(t *testing.T) {
	catalog := mustCatalog(t, executionDefinition())
	a := executionAdmission(t)
	invocations := 0
	handler := func(context.Context, Call) (string, error) { invocations++; return "", nil }
	e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: handler})
	cases := map[string]Admission{
		"zero":              {},
		"malformed":         assertDecision(t, catalog, Call{Name: "bad name", Arguments: []byte("{}")}, MalformedCall),
		"unknown":           assertDecision(t, catalog, Call{Name: "unknown", Arguments: []byte("{}")}, UnknownTool),
		"invalid arguments": assertDecision(t, catalog, Call{Name: "private_tool", Arguments: []byte("{}")}, InvalidArguments),
		"policy":            assertDecision(t, catalog, Call{Name: "private_tool", Arguments: []byte(strings.Repeat(" ", 16385))}, PolicyRejected),
		// Same-package fixtures ensure both status and reason are necessary.
		"admitted wrong reason":   {status: Admitted, reason: InvalidArguments, call: a.call},
		"rejected allowed reason": {status: Rejected, reason: Allowed, call: a.call},
	}
	for name, admission := range cases {
		t.Run(name, func(t *testing.T) {
			executionFailure(t, e, context.Background(), admission, ErrExecutionDenied)
			if invocations != 0 {
				t.Fatal("denied admission invoked handler")
			}
		})
	}
	narrower := executionDefinition()
	narrower.Parameters[0].Type = Number
	for name, executor := range map[string]*Executor{
		"nil":                               nil,
		"zero":                              {},
		"missing catalog":                   {handlers: map[string]Handler{"private_tool": handler}},
		"missing map":                       {catalog: catalog},
		"missing handler":                   {catalog: catalog, handlers: map[string]Handler{}},
		"nil handler entry":                 {catalog: catalog, handlers: map[string]Handler{"private_tool": nil}},
		"unknown execution tool":            executionExecutor(t, Binding{Definition: Definition{Name: "other"}, Handler: handler}),
		"incompatible execution definition": executionExecutor(t, Binding{Definition: narrower, Handler: handler}),
	} {
		t.Run(name, func(t *testing.T) {
			executionFailure(t, executor, context.Background(), a, ErrExecutionDenied)
			if invocations != 0 {
				t.Fatal("denied executor invoked handler")
			}
		})
	}
	executionFailure(t, e, nil, a, ErrExecutionDenied)
	if invocations != 0 {
		t.Fatal("nil context invoked handler")
	}
}

func TestExecutorOwnershipAndSingleDispatch(t *testing.T) {
	a := executionAdmission(t)
	want, _ := a.AdmittedCall()
	var calls, otherCalls, replacementCalls int
	ctx := context.WithValue(context.Background(), struct{}{}, "caller context")
	definition := executionDefinition()
	parameters := definition.Parameters
	bindings := []Binding{
		{Definition: definition, Handler: func(received context.Context, call Call) (string, error) {
			calls++
			if received != ctx || call.Name != want.Name || string(call.Arguments) != string(want.Arguments) {
				t.Fatal("handler did not receive exact context/call")
			}
			for i := range call.Arguments {
				call.Arguments[i] = 'x'
			}
			return "raw output", nil
		}},
		{Definition: Definition{Name: "other"}, Handler: func(context.Context, Call) (string, error) { otherCalls++; return "", nil }},
	}
	e := executionExecutor(t, bindings...)
	parameters[0] = Parameter{Name: "replaced", Type: Number, Required: false}
	definition.Name = "replaced"
	bindings[0].Definition.Name = "replaced"
	bindings[0].Definition.Parameters = nil
	bindings[0].Handler = func(context.Context, Call) (string, error) { replacementCalls++; return "", nil }
	bindings = append(bindings, Binding{Definition: Definition{Name: "extra"}, Handler: bindings[0].Handler})
	// Each explicit Execute dispatches once; a repeated call is not deduplicated.
	for i := 1; i <= 2; i++ {
		r, err := e.Execute(ctx, a)
		if err != nil || r.Output() != "raw output" || calls != i || otherCalls != 0 || replacementCalls != 0 {
			t.Fatal("single dispatch or constructor ownership failed")
		}
		stored, ok := a.AdmittedCall()
		if !ok || stored.Name != want.Name || string(stored.Arguments) != string(want.Arguments) {
			t.Fatal("handler mutated stored admission")
		}
	}
	// A changed Required flag in caller memory must not widen authority.
	loose := executionDefinition()
	loose.Parameters[0].Required = false
	missing := assertDecision(t, mustCatalog(t, loose), Call{Name: "private_tool", Arguments: []byte("{}")}, Allowed)
	executionFailure(t, e, ctx, missing, ErrExecutionDenied)
	if calls != 2 || otherCalls != 0 || replacementCalls != 0 {
		t.Fatal("mutation widened authority")
	}
}

// If formatting a handler error or panic were attempted, these methods panic.
type opaqueExecutionFailure struct{}

func (opaqueExecutionFailure) Error() string  { panic("error was inspected") }
func (opaqueExecutionFailure) String() string { panic("panic was inspected") }

func TestExecutorHandlerFailures(t *testing.T) {
	for name, handler := range map[string]Handler{
		"error": func(context.Context, Call) (string, error) {
			return "private_output", errors.New("private_handler_error")
		},
		"opaque error":                    func(context.Context, Call) (string, error) { return "private_output", opaqueExecutionFailure{} },
		"panic":                           func(context.Context, Call) (string, error) { panic("private_panic") },
		"opaque panic":                    func(context.Context, Call) (string, error) { panic(opaqueExecutionFailure{}) },
		"nil panic":                       func(context.Context, Call) (string, error) { panic(nil) },
		"context error with live context": func(context.Context, Call) (string, error) { return "private_output", context.Canceled },
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: func(ctx context.Context, call Call) (string, error) {
				calls++
				return handler(ctx, call)
			}})
			executionFailure(t, e, context.Background(), executionAdmission(t), ErrHandlerFailed)
			if calls != 1 {
				t.Fatal("failure retried or invoked multiple handlers")
			}
		})
	}
}

func TestExecutorContext(t *testing.T) {
	a := executionAdmission(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer stop()
	for _, ctx := range []context.Context{canceled, expired} {
		calls := 0
		e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: func(context.Context, Call) (string, error) { calls++; return "", nil }})
		executionFailure(t, e, ctx, a, ctx.Err())
		if calls != 0 {
			t.Fatal("canceled context invoked handler")
		}
	}
	for _, mode := range []string{"success", "error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: func(received context.Context, _ Call) (string, error) {
				calls++
				if received != ctx {
					t.Fatal("context was substituted")
				}
				cancel()
				if mode == "panic" {
					panic("private_panic")
				}
				if mode == "error" {
					return "private_output", errors.New("private_error")
				}
				return "private_output", nil
			}})
			executionFailure(t, e, ctx, a, context.Canceled)
			if calls != 1 {
				t.Fatal("incorrect canceled dispatch count")
			}
		})
	}
}

func TestExecutorOutput(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		want         error
	}{
		{"empty", "", nil},
		{"UTF8", "  raw\n\u00e9\u4e16\u754c\x00", nil},
		{"64 KiB ASCII", strings.Repeat("x", 64*1024), nil},
		{"64 KiB multibyte", strings.Repeat("\u00e9", 32*1024), nil},
		{"too large", strings.Repeat("x", 64*1024+1), ErrExecutionOutputTooLarge},
		{"too large multibyte", strings.Repeat("\u00e9", 32*1024+1), ErrExecutionOutputTooLarge},
		{"invalid UTF8", string([]byte{0xff}), ErrInvalidExecutionOutput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: func(context.Context, Call) (string, error) { calls++; return tc.output, nil }})
			r, err := e.Execute(context.Background(), executionAdmission(t))
			if err != tc.want || calls != 1 {
				t.Fatal("incorrect output classification or dispatch count")
			}
			if err != nil {
				if r != (ExecutionResult{}) {
					t.Fatal("error retained output")
				}
			} else if r.Output() != tc.output {
				t.Fatal("output truncated, repaired, or normalized")
			}
		})
	}
}

func TestExecutorDiagnostics(t *testing.T) {
	e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: func(context.Context, Call) (string, error) { return "private_output", nil }})
	r, err := e.Execute(context.Background(), executionAdmission(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{e, *e, r, &r, ExecutionResult{}, Executor{}, (*Executor)(nil)} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			rendered := fmt.Sprintf(format, value)
			for _, secret := range []string{"private_tool", "private_argument", "private_output", "private_handler_error", "private_panic"} {
				if strings.Contains(rendered, secret) {
					t.Fatal("diagnostic exposed payload")
				}
			}
		}
	}
	for err, text := range map[error]string{
		ErrInvalidExecutor: "ai/tool: invalid executor", ErrExecutionDenied: "ai/tool: execution denied",
		ErrHandlerFailed: "ai/tool: handler failed", ErrInvalidExecutionOutput: "ai/tool: invalid execution output",
		ErrExecutionOutputTooLarge: "ai/tool: execution output too large",
	} {
		if err.Error() != text || errors.Unwrap(err) != nil {
			t.Fatal("error is not a fixed sentinel")
		}
	}
	if (ExecutionResult{}).Output() != "" {
		t.Fatal("zero result is not empty")
	}
}

func TestExecutorExternalConcurrentCalls(t *testing.T) {
	// Only this caller creates concurrency; immutable dispatch state must be safe.
	var calls atomic.Int32
	e := executionExecutor(t, Binding{Definition: executionDefinition(), Handler: func(context.Context, Call) (string, error) { calls.Add(1); return "ok", nil }})
	a := executionAdmission(t)
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			r, err := e.Execute(context.Background(), a)
			if err != nil || r.Output() != "ok" {
				t.Error("concurrent dispatch failed")
			}
		})
	}
	workers.Wait()
	if calls.Load() != 16 {
		t.Fatal("external calls did not dispatch once each")
	}
}
