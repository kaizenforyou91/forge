package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func fcoDefinition() tool.Definition {
	return tool.Definition{Name: "private_tool", Parameters: []tool.Parameter{{Name: "value", Type: tool.String, Required: true}}}
}

func fcoDecision(t *testing.T, name, arguments, itemID, callID string) FunctionCallDecision {
	t.Helper()
	request, err := BuildFunctionCallRequest(ai.Request{Text: "offline fixture", Model: "fixture-model", MaxOutputTokens: 123}, []tool.Definition{fcoDefinition()})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := json.Marshal(map[string]any{
		"status": "completed", "error": nil, "incomplete_details": nil,
		"output": []any{map[string]any{
			"type": "function_call", "status": "completed", "id": itemID, "call_id": callID,
			"name": name, "arguments": arguments,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := MapFunctionCallResponse(fixture, request.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func fcoAdmitted(t *testing.T) FunctionCallDecision {
	t.Helper()
	d := fcoDecision(t, "private_tool", `{"value":"private_argument"}`, "item_private", "call_fixture")
	if d.Admission().Status() != tool.Admitted || d.Admission().Reason() != tool.Allowed {
		t.Fatal("fixture not admitted")
	}
	return d
}

func fcoExecute(t *testing.T, decision FunctionCallDecision, output string) tool.ExecutionResult {
	t.Helper()
	calls := 0
	executor, err := tool.NewExecutor([]tool.Binding{{
		Definition: fcoDefinition(),
		Handler:    func(context.Context, tool.Call) (string, error) { calls++; return output, nil },
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(context.Background(), decision.Admission())
	if err != nil || calls != 1 {
		t.Fatalf("C1 fixture execution failed: %v", err)
	}
	return result
}

func fcoBuild(t *testing.T, decision FunctionCallDecision, result tool.ExecutionResult) FunctionCallOutput {
	t.Helper()
	item, err := BuildFunctionCallOutput(decision, result)
	if err != nil || len(item.Body()) == 0 {
		t.Fatalf("C2 build failed: %v", err)
	}
	return item
}

func TestFunctionCallOutputEndToEnd(t *testing.T) {
	// Real public B2 -> B1 -> C1 contracts, entirely offline. Only this test's
	// explicit C1 call invokes a handler; serialization cannot invoke it again.
	decision := fcoAdmitted(t)
	calls := 0
	executor, err := tool.NewExecutor([]tool.Binding{{Definition: fcoDefinition(), Handler: func(context.Context, tool.Call) (string, error) {
		calls++
		return "local result", nil
	}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(context.Background(), decision.Admission())
	if err != nil {
		t.Fatal(err)
	} // caller checks success before C2
	item := fcoBuild(t, decision, result)
	want := `{"type":"function_call_output","call_id":"call_fixture","output":"local result"}`
	if string(item.Body()) != want || calls != 1 {
		t.Fatal("end-to-end output or invocation count mismatch")
	}
	for range 3 {
		if !bytes.Equal(item.Body(), fcoBuild(t, decision, result).Body()) {
			t.Fatal("repeated build not deterministic")
		}
	}
	if calls != 1 {
		t.Fatal("serialization invoked handler")
	}
}

func TestFunctionCallOutputTextAndShape(t *testing.T) {
	decision := fcoAdmitted(t)
	for name, text := range map[string]string{
		"normal": "local result", "empty": "", "whitespace": " \n\t raw \r\n ",
		"quotes and slashes": "\"quoted\"\\path/", "Unicode": "\u00e9 e\u0301 \u4e16\u754c \U0001f642",
		"not JSON": "{unfinished", "JSON stays string": `{"value":123}`, "HTML and separators": "<>&\u2028\u2029",
		"64 KiB ASCII":    strings.Repeat("x", 64*1024),
		"64 KiB Unicode":  strings.Repeat("\u00e9", 32*1024),
		"64 KiB escaping": strings.Repeat("\x00", 64*1024),
	} {
		t.Run(name, func(t *testing.T) {
			result := fcoExecute(t, decision, text)
			item := fcoBuild(t, decision, result)
			var got map[string]any
			if err := json.Unmarshal(item.Body(), &got); err != nil {
				t.Fatal(err)
			}
			// Exact map equality forbids ALL extra fields, including id, name,
			// arguments, status, response IDs, and execution metadata.
			want := map[string]any{"type": "function_call_output", "call_id": "call_fixture", "output": text}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("shape changed or raw output rewritten")
			}
			if len(item.Body()) > maxBody {
				t.Fatal("item exceeds adapter bound")
			}
		})
	}
}

func TestFunctionCallOutputCorrelation(t *testing.T) {
	admitted := fcoAdmitted(t)
	for name, decision := range map[string]FunctionCallDecision{
		"zero":              {},
		"malformed name":    fcoDecision(t, "bad name", `{}`, "item_private", "call_private"),
		"unknown tool":      fcoDecision(t, "unknown", `{}`, "item_private", "call_private"),
		"invalid arguments": fcoDecision(t, "private_tool", `{}`, "item_private", "call_private"),
		"policy rejection":  fcoDecision(t, "private_tool", strings.Repeat(" ", 16385), "item_private", "call_private"),
		// Same-package fixtures exercise impossible/incomplete B1 state without
		// weakening the real mapper or inventing a second admission API.
		"admitted no correlation": {admission: admitted.Admission()},
		"item ID only":            {admission: admitted.Admission(), itemID: "item_private"},
		"IDs without admission":   {itemID: "item_private", callID: "call_private"},
		"invalid call ID":         {admission: admitted.Admission(), callID: "call with space"},
		"oversized call ID":       {admission: admitted.Admission(), callID: strings.Repeat("c", 257)},
	} {
		t.Run(name, func(t *testing.T) {
			item, err := BuildFunctionCallOutput(decision, tool.ExecutionResult{})
			if err != ai.ErrInvalidRequest || item.Body() != nil || !reflect.DeepEqual(item, FunctionCallOutput{}) {
				t.Fatal("invalid correlation did not fail closed")
			}
			if errors.Unwrap(err) != nil {
				t.Fatal("failure exposed an underlying error")
			}
		})
	}
	for _, callID := range []string{"call_fixture", "opaque-\u00e9_\"\\<>&", strings.Repeat("c", 256)} {
		d := fcoDecision(t, "private_tool", `{"value":"x"}`, "different_item_ID", callID)
		item := fcoBuild(t, d, fcoExecute(t, d, "output"))
		var got map[string]any
		if err := json.Unmarshal(item.Body(), &got); err != nil {
			t.Fatal(err)
		}
		if got["call_id"] != callID || len(got) != 3 {
			t.Fatal("call ID normalized, substituted, or extra fields emitted")
		}
	}
}

func TestFunctionCallOutputOwnershipAndDiagnostics(t *testing.T) {
	var zero FunctionCallOutput
	if zero.Body() != nil {
		t.Fatal("zero value exposes body")
	}
	d := fcoAdmitted(t)
	result := fcoExecute(t, d, "private_output")
	item := fcoBuild(t, d, result)
	want := item.Body()
	modified := item.Body()
	for i := range modified {
		modified[i] = 'x'
	}
	for range 3 {
		if !bytes.Equal(item.Body(), want) || !bytes.Equal(fcoBuild(t, d, result).Body(), want) {
			t.Fatal("mutable body alias or nondeterminism")
		}
	}
	// The builder retains only serialized bytes, not a reference to decision.
	d.callID = "changed_call"
	d.itemID = "changed_item"
	d.admission = tool.Admission{}
	result = tool.ExecutionResult{}
	if !bytes.Equal(item.Body(), want) {
		t.Fatal("caller reassignment altered item")
	}
	_, err := BuildFunctionCallOutput(d, result)
	if err != ai.ErrInvalidRequest {
		t.Fatal("expected safe failure")
	}
	for _, value := range []any{item, &item, zero, &zero, err} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			rendered := fmt.Sprintf(format, value)
			for _, secret := range []string{"call_fixture", "item_private", "private_tool", "private_argument", "private_output", "changed_call", "changed_item"} {
				if strings.Contains(rendered, secret) {
					t.Fatal("diagnostic exposed correlation or payload")
				}
			}
		}
	}
}

func TestFunctionCallOutputZeroResultIsNotProvenance(t *testing.T) {
	// No result-only API can distinguish a zero result from successful empty
	// output. Checking C1's error and pairing the result is caller responsibility.
	d := fcoAdmitted(t)
	empty := fcoExecute(t, d, "")
	if !bytes.Equal(fcoBuild(t, d, empty).Body(), fcoBuild(t, d, tool.ExecutionResult{}).Body()) {
		t.Fatal("serializer invented success provenance or rejected valid empty output")
	}
}
