package tool

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestAuthorityConstruction(t *testing.T) {
	handler := Handler(func(context.Context, Call) (string, error) { return "", nil })
	valid := Binding{Definition: executionDefinition(), Handler: handler}
	for _, bindings := range [][]Binding{nil, {}, {valid}, {valid, {Definition: Definition{Name: "other"}, Handler: handler}}} {
		a, err := NewAuthority(bindings)
		if err != nil || a == nil || a.Executor() == nil || len(a.Definitions()) != len(bindings) {
			t.Fatal("valid constructor failed", err)
		}
		if len(bindings) == 0 {
			executionFailure(t, a.Executor(), context.Background(), executionAdmission(t), ErrExecutionDenied)
		}
	}
	tools := make([]Binding, maxTools+1)
	for i := range tools {
		tools[i] = Binding{Definition: Definition{Name: fmt.Sprintf("tool_%d", i)}, Handler: handler}
	}
	parameters := make([]Parameter, maxParameters+1)
	for i := range parameters {
		parameters[i] = Parameter{Name: fmt.Sprintf("p_%d", i), Type: String}
	}
	for name, bindings := range map[string][]Binding{
		"nil handler":         {{Definition: executionDefinition()}},
		"empty tool":          {{Handler: handler}},
		"invalid name":        {{Definition: Definition{Name: "bad name"}, Handler: handler}},
		"long name":           {{Definition: Definition{Name: strings.Repeat("x", maxNameBytes+1)}, Handler: handler}},
		"duplicate tool":      {valid, valid},
		"invalid parameter":   {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: "bad name", Type: String}}}, Handler: handler}},
		"long parameter":      {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: strings.Repeat("x", maxNameBytes+1), Type: String}}}, Handler: handler}},
		"duplicate parameter": {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: "value", Type: String}, {Name: "value", Type: Number}}}, Handler: handler}},
		"invalid type":        {{Definition: Definition{Name: "tool", Parameters: []Parameter{{Name: "value", Type: InvalidType}}}, Handler: handler}},
		"too many tools":      tools,
		"too many parameters": {{Definition: Definition{Name: "tool", Parameters: parameters}, Handler: handler}},
	} {
		t.Run(name, func(t *testing.T) {
			a, err := NewAuthority(bindings)
			if a != nil || err != ErrInvalidExecutor {
				t.Fatal("invalid configuration exposed authority or wrong error")
			}
		})
	}
	// Inclusive boundaries remain accepted, using the existing Stage A limits.
	tools = tools[:maxTools]
	tools[0].Definition.Parameters = parameters[:maxParameters]
	if _, err := NewAuthority(tools); err != nil {
		t.Fatal("valid bounds rejected", err)
	}
}

func TestAuthorityOwnershipAndAlignment(t *testing.T) {
	calls, replacements := 0, 0
	definition := executionDefinition()
	parameters := definition.Parameters
	bindings := []Binding{{Definition: definition, Handler: func(context.Context, Call) (string, error) {
		calls++
		return "original output", nil
	}}}
	a, err := NewAuthority(bindings)
	if err != nil {
		t.Fatal(err)
	}
	executor := a.Executor()
	parameters[0].Name, parameters[0].Type, parameters[0].Required = "changed", Number, false
	definition.Name = "changed"
	bindings[0].Definition.Name = "changed"
	bindings[0].Definition.Parameters = nil
	bindings[0].Handler = func(context.Context, Call) (string, error) { replacements++; return "replacement", nil }
	bindings[0] = Binding{}
	bindings = append(bindings, Binding{})
	returned := a.Definitions()
	returned[0].Name = "changed"
	returned[0].Parameters[0] = Parameter{Name: "changed", Type: Number}
	returned[0].Parameters = nil
	for range 2 {
		defs := a.Definitions()
		if !reflect.DeepEqual(defs, []Definition{executionDefinition()}) || a.Executor() != executor {
			t.Fatal("mutation changed authority")
		}
		admission := assertDecision(t, mustCatalog(t, defs...), Call{Name: "private_tool", Arguments: []byte(`{"value":"text"}`)}, Allowed)
		result, err := executor.Execute(context.Background(), admission)
		if err != nil || result.Output() != "original output" {
			t.Fatal("alignment failed", err)
		}
		defs[0].Parameters[0].Required = false
	}
	if calls != 2 || replacements != 0 {
		t.Fatal("captured handler changed")
	}
	// Independently admitted unknown/wrong-type/missing-required calls must not
	// execute through the bundled C1 authority, even after all caller mutations.
	for _, tc := range []struct {
		definition Definition
		call       Call
	}{
		{Definition{Name: "unknown"}, Call{Name: "unknown", Arguments: []byte(`{}`)}},
		{Definition{Name: "private_tool", Parameters: []Parameter{{Name: "value", Type: Number, Required: true}}}, Call{Name: "private_tool", Arguments: []byte(`{"value":1}`)}},
		{Definition{Name: "private_tool", Parameters: []Parameter{{Name: "value", Type: String}}}, Call{Name: "private_tool", Arguments: []byte(`{}`)}},
	} {
		admission := assertDecision(t, mustCatalog(t, tc.definition), tc.call, Allowed)
		executionFailure(t, executor, context.Background(), admission, ErrExecutionDenied)
	}
	if calls != 2 || replacements != 0 {
		t.Fatal("denied call executed")
	}
}

func TestAuthorityZeroAndDiagnostics(t *testing.T) {
	for _, a := range []*Authority{nil, {}} {
		if a.Definitions() != nil || a.Executor() != nil {
			t.Fatal("zero authority exposed state")
		}
	}
	a, err := NewAuthority([]Binding{{Definition: executionDefinition(), Handler: func(context.Context, Call) (string, error) { return "private_output", nil }}})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{a, *a, Authority{}, (*Authority)(nil)} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			rendered := fmt.Sprintf(format, value)
			for _, secret := range []string{"private_tool", "value", "private_output", "Handler", "definitions"} {
				if strings.Contains(rendered, secret) {
					t.Fatal("diagnostics exposed configuration")
				}
			}
		}
	}
	if a.String() != "tool authority (configuration redacted)" || a.GoString() != a.String() {
		t.Fatal("diagnostic changed")
	}
}

func TestAuthorityConcurrentAccess(t *testing.T) {
	var calls atomic.Int32
	a, err := NewAuthority([]Binding{{Definition: executionDefinition(), Handler: func(context.Context, Call) (string, error) { calls.Add(1); return "ok", nil }}})
	if err != nil {
		t.Fatal(err)
	}
	admission := executionAdmission(t)
	executor := a.Executor()
	start := make(chan struct{})
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			<-start
			defs := a.Definitions()
			if !reflect.DeepEqual(defs, []Definition{executionDefinition()}) || a.Executor() != executor {
				t.Error("concurrent access changed authority")
				return
			}
			defs[0].Name = "changed"
			defs[0].Parameters[0].Type = Number
			r, err := a.Executor().Execute(context.Background(), admission)
			if err != nil || r.Output() != "ok" {
				t.Error("concurrent execution failed")
			}
		})
	}
	close(start)
	workers.Wait()
	if calls.Load() != 16 {
		t.Fatal("incorrect dispatch count")
	}
}
