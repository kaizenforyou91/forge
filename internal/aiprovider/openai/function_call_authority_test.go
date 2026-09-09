package openai

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func TestAuthorizedFunctionRoundTripInvalid(t *testing.T) {
	empty, err := tool.NewAuthority(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, authority := range []*tool.Authority{nil, {}, empty} {
		s := frClient(t)
		r, err := s.client.ExecuteAuthorizedFunctionRoundTrip(context.Background(), request(), authority)
		requireFailure(t, r, err, ai.ErrInvalidRequest)
		if len(s.wires) != 0 {
			t.Fatal("invalid authority performed HTTP")
		}
	}
}

func TestAuthorizedFunctionRoundTripSnapshot(t *testing.T) {
	for _, direct := range []bool{true, false} {
		calls, replacements := 0, 0
		def := tool.Definition{Name: "lookup", Parameters: []tool.Parameter{{Name: "value", Type: tool.String, Required: true}}}
		parameters := def.Parameters
		bindings := []tool.Binding{{Definition: def, Handler: func(_ context.Context, call tool.Call) (string, error) {
			calls++
			if call.Name != "lookup" || string(call.Arguments) != `{"value":"text"}` {
				t.Fatal("call contract changed")
			}
			return "local result", nil
		}}}
		a, err := tool.NewAuthority(bindings)
		if err != nil {
			t.Fatal(err)
		}
		canonical, err := BuildFunctionCallRequest(request(), a.Definitions())
		if err != nil {
			t.Fatal(err)
		}
		parameters[0] = tool.Parameter{Name: "changed", Type: tool.Number}
		bindings[0].Definition.Name = "changed"
		bindings[0].Definition.Parameters = nil
		bindings[0].Handler = func(context.Context, tool.Call) (string, error) { replacements++; return "changed", nil }
		returned := a.Definitions()
		returned[0].Name = "changed"
		returned[0].Parameters[0] = tool.Parameter{Name: "changed", Type: tool.Number}
		first := responseObject("direct")
		if !direct {
			first = frFunction()
			item := first["output"].([]any)[0].(map[string]any)
			item["name"], item["arguments"] = "lookup", `{"value":"text"}`
		}
		s := frClient(t, frStep{body: encoded(first)}, frStep{body: encoded(responseObject("final"))})
		r, err := s.client.ExecuteAuthorizedFunctionRoundTrip(context.Background(), request(), a)
		if err != nil {
			t.Fatal(err)
		}
		if string(s.wires[0]) != string(canonical.Body()) || replacements != 0 {
			t.Fatal("snapshot mismatch")
		}
		if direct {
			if len(s.wires) != 1 || calls != 0 || r.Text != "direct" {
				t.Fatal("direct count/result")
			}
		} else {
			if len(s.wires) != 2 || calls != 1 || r.Text != "final" {
				t.Fatal("function count/result")
			}
			var continuation, initial map[string]any
			json.Unmarshal(s.wires[1], &continuation)
			json.Unmarshal(canonical.Body(), &initial)
			if !reflect.DeepEqual(continuation["tools"], initial["tools"]) {
				t.Fatal("continuation declarations changed")
			}
			items := continuation["input"].([]any)
			if !reflect.DeepEqual(items[1], map[string]any{"type": "function_call", "call_id": "call_fixture", "name": "lookup", "arguments": `{"value":"text"}`}) || !reflect.DeepEqual(items[2], map[string]any{"type": "function_call_output", "call_id": "call_fixture", "output": "local result"}) {
				t.Fatal("pairing changed")
			}
		}
	}
}

func TestAuthorizedFunctionRoundTripSharedAuthority(t *testing.T) {
	var calls atomic.Int32
	a, err := tool.NewAuthority([]tool.Binding{{Definition: fcoDefinition(), Handler: func(context.Context, tool.Call) (string, error) { calls.Add(1); return "local result", nil }}})
	if err != nil {
		t.Fatal(err)
	}
	const count = 8
	scripts := make([]*frScript, count)
	for i := range scripts {
		scripts[i] = frClient(t, frStep{body: encoded(frFunction())}, frStep{body: encoded(responseObject("final"))})
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, script := range scripts {
		workers.Go(func() {
			<-start
			r, err := script.client.ExecuteAuthorizedFunctionRoundTrip(context.Background(), request(), a)
			if err != nil || r.Text != "final" || len(script.wires) != 2 {
				t.Error("shared authority round-trip failed")
			}
		})
	}
	close(start)
	workers.Wait()
	// Same call_id in independent invocations must not share replay state.
	if calls.Load() != count {
		t.Fatal("authority introduced shared replay or duplicate dispatch")
	}
}
