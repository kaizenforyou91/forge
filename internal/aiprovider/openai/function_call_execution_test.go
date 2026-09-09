package openai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func TestFunctionCallExecutionAuthority(t *testing.T) {
	definition := tool.Definition{Name: "propose", Parameters: []tool.Parameter{{Name: "value", Type: tool.String, Required: true}}}
	request, err := BuildFunctionCallRequest(ai.Request{Text: "offline fixture", Model: "fixture-model", MaxOutputTokens: 123}, []tool.Definition{definition})
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(request.Body(), &body); err != nil || len(body.Tools) != 1 || body.Tools[0].Name != definition.Name {
		t.Fatal("B2 serialized declaration does not match definition")
	}
	for _, tc := range []struct {
		name, executionName string
		kind                tool.ValueType
		arguments           string
		b1Reason            tool.Reason
		wantError           error
		wantCalls           int
	}{
		{"aligned authority", "propose", tool.String, `{"value":"text"}`, tool.Allowed, nil, 1},
		{"B1 admitted C1 incompatible", "propose", tool.Number, `{"value":"text"}`, tool.Allowed, tool.ErrExecutionDenied, 0},
		{"C1 missing tool", "other", tool.String, `{"value":"text"}`, tool.Allowed, tool.ErrExecutionDenied, 0},
		{"B1 rejected", "propose", tool.String, `{"value":123}`, tool.InvalidArguments, tool.ErrExecutionDenied, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture, err := json.Marshal(map[string]any{
				"status": "completed", "error": nil, "incomplete_details": nil,
				"output": []any{map[string]any{
					"type": "function_call", "status": "completed", "id": "item_fixture", "call_id": "call_fixture",
					"name": body.Tools[0].Name, "arguments": tc.arguments,
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			// B1 uses the SAME immutable authority returned by B2. Correlation
			// fields stay with B1; only the Stage A admission enters C1.
			decision, err := MapFunctionCallResponse(fixture, request.Catalog())
			if err != nil || decision.Admission().Reason() != tc.b1Reason {
				t.Fatal("unexpected B1 admission")
			}
			wantStatus := tool.Rejected
			if tc.b1Reason == tool.Allowed {
				wantStatus = tool.Admitted
			}
			if decision.Admission().Status() != wantStatus {
				t.Fatal("unexpected B1 status")
			}
			calls := 0
			ctx := context.Background()
			executor, err := tool.NewExecutor([]tool.Binding{{
				Definition: tool.Definition{Name: tc.executionName, Parameters: []tool.Parameter{{Name: "value", Type: tc.kind, Required: true}}},
				Handler: func(received context.Context, call tool.Call) (string, error) {
					calls++
					if received != ctx || call.Name != definition.Name || string(call.Arguments) != tc.arguments {
						t.Fatal("handler received substituted context or call")
					}
					return "local result", nil
				},
			}})
			if err != nil {
				t.Fatal(err)
			}
			result, err := executor.Execute(ctx, decision.Admission())
			if err != tc.wantError || calls != tc.wantCalls {
				t.Fatal("execution authority gate or single dispatch failed")
			}
			if err != nil && result.Output() != "" {
				t.Fatal("denied call exposed result")
			}
			if err == nil && result.Output() != "local result" {
				t.Fatal("successful execution lost output")
			}
		})
	}
}
