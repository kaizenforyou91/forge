package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func fcrRequest() ai.Request {
	return ai.Request{Text: "  offline proposal\n", Model: "fixture-model", MaxOutputTokens: 123}
}

func fcrBuild(t *testing.T, request ai.Request, definitions []tool.Definition) FunctionCallRequest {
	t.Helper()
	r, err := BuildFunctionCallRequest(request, definitions)
	if err != nil || r.Catalog() == nil || len(r.Body()) == 0 {
		t.Fatalf("build failed: %v", err)
	}
	return r
}

func fcrJSON(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal("invalid serialized JSON")
	}
	return value
}

func fcrReject(t *testing.T, request ai.Request, definitions []tool.Definition, want error) {
	t.Helper()
	r, err := BuildFunctionCallRequest(request, definitions)
	if !errors.Is(err, want) {
		t.Fatalf("error = %v; want %v", err, want)
	}
	if !reflect.DeepEqual(r, FunctionCallRequest{}) || r.Body() != nil || r.Catalog() != nil {
		t.Fatal("error exposed partial request or authority")
	}
}

func TestFunctionCallRequestPayload(t *testing.T) {
	definitions := []tool.Definition{
		{Name: "z_tool", Parameters: []tool.Parameter{
			{Name: "z_null", Type: tool.Null, Required: true},
			{Name: "s_string", Type: tool.String},
			{Name: "o_object", Type: tool.Object},
			{Name: "n_number", Type: tool.Number},
			{Name: "b_array", Type: tool.Array},
			{Name: "a_boolean", Type: tool.Boolean, Required: true},
		}},
		{Name: "a_empty"},
	}
	r := fcrBuild(t, fcrRequest(), definitions)
	// Exact shape comparison also forbids descriptions, invented nested schemas,
	// nullable-required rewrites, additional request controls, and non-function tools.
	want := []byte(`{
		"model":"fixture-model","input":"  offline proposal\n","max_output_tokens":123,
		"stream":false,"background":false,"store":false,
		"tool_choice":"auto","parallel_tool_calls":false,
		"tools":[
			{"type":"function","name":"a_empty","strict":false,"parameters":{
				"type":"object","properties":{},"required":[],"additionalProperties":false}},
			{"type":"function","name":"z_tool","strict":false,"parameters":{
				"type":"object","additionalProperties":false,"required":["a_boolean","z_null"],
				"properties":{
					"s_string":{"type":"string"},"n_number":{"type":"number"},
					"a_boolean":{"type":"boolean"},"o_object":{"type":"object"},
					"b_array":{"type":"array"},"z_null":{"type":"null"}}}}
		]}`)
	if !reflect.DeepEqual(fcrJSON(t, r.Body()), fcrJSON(t, want)) {
		t.Fatal("payload differs from bounded function-tool contract")
	}
	// Reusing the text-only payload must preserve every original request field.
	textOnly, err := json.Marshal(payload{Model: fcrRequest().Model, Input: fcrRequest().Text, MaxOutputTokens: fcrRequest().MaxOutputTokens})
	if err != nil {
		t.Fatal(err)
	}
	full := fcrJSON(t, r.Body()).(map[string]any)
	delete(full, "tools")
	delete(full, "tool_choice")
	delete(full, "parallel_tool_calls")
	if !reflect.DeepEqual(full, fcrJSON(t, textOnly)) {
		t.Fatal("text request fields changed")
	}
}

func TestFunctionCallRequestValidation(t *testing.T) {
	valid := fcrRequest()
	cases := []struct {
		name string
		r    ai.Request
	}{
		{"zero", ai.Request{}},
		{"blank text", ai.Request{Text: " \t\n", Model: valid.Model, MaxOutputTokens: valid.MaxOutputTokens}},
		{"invalid UTF8", ai.Request{Text: string([]byte{0xff}), Model: valid.Model, MaxOutputTokens: valid.MaxOutputTokens}},
		{"oversized text", ai.Request{Text: strings.Repeat("x", ai.MaxInputBytes+1), Model: valid.Model, MaxOutputTokens: valid.MaxOutputTokens}},
		{"blank model", ai.Request{Text: valid.Text, MaxOutputTokens: valid.MaxOutputTokens}},
		{"invalid model", ai.Request{Text: valid.Text, Model: "bad model", MaxOutputTokens: valid.MaxOutputTokens}},
		{"oversized model", ai.Request{Text: valid.Text, Model: strings.Repeat("m", ai.MaxModelBytes+1), MaxOutputTokens: valid.MaxOutputTokens}},
		{"tokens below minimum", ai.Request{Text: valid.Text, Model: valid.Model, MaxOutputTokens: ai.MinOutputTokens - 1}},
		{"tokens above maximum", ai.Request{Text: valid.Text, Model: valid.Model, MaxOutputTokens: ai.MaxOutputTokens + 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fcrReject(t, tc.r, []tool.Definition{{Name: "propose"}}, ai.ErrInvalidRequest)
		})
	}
	// Request validation precedes both nonempty and Stage A catalog validation.
	fcrReject(t, ai.Request{}, nil, ai.ErrInvalidRequest)
	fcrReject(t, ai.Request{}, []tool.Definition{{Name: "invalid name"}}, ai.ErrInvalidRequest)
	for _, tokens := range []int{ai.MinOutputTokens, ai.MaxOutputTokens} {
		r := valid
		r.MaxOutputTokens = tokens
		built := fcrBuild(t, r, []tool.Definition{{Name: "propose"}})
		if fcrJSON(t, built.Body()).(map[string]any)["max_output_tokens"] != float64(tokens) {
			t.Fatal("token boundary was rewritten")
		}
	}
}

func fcrDefinitions(tools, parameters int) []tool.Definition {
	definitions := make([]tool.Definition, tools)
	for i := range definitions {
		definitions[i] = tool.Definition{Name: fmt.Sprintf("tool_%d", i), Parameters: make([]tool.Parameter, parameters)}
		for j := range definitions[i].Parameters {
			definitions[i].Parameters[j] = tool.Parameter{Name: fmt.Sprintf("p_%d", j), Type: tool.String, Required: true}
		}
	}
	return definitions
}

func TestFunctionCallRequestDefinitions(t *testing.T) {
	fcrReject(t, fcrRequest(), nil, ai.ErrInvalidRequest)
	fcrReject(t, fcrRequest(), []tool.Definition{}, ai.ErrInvalidRequest)
	cases := []struct {
		name string
		defs []tool.Definition
	}{
		{"invalid tool", []tool.Definition{{Name: "bad name"}}},
		{"long tool", []tool.Definition{{Name: strings.Repeat("t", 65)}}},
		{"duplicate tools", []tool.Definition{{Name: "same"}, {Name: "same"}}},
		{"too many tools", fcrDefinitions(65, 0)},
		{"invalid parameter", []tool.Definition{{Name: "valid", Parameters: []tool.Parameter{{Name: "0bad", Type: tool.String}}}}},
		{"long parameter", []tool.Definition{{Name: "valid", Parameters: []tool.Parameter{{Name: strings.Repeat("p", 65), Type: tool.String}}}}},
		{"duplicate parameters", []tool.Definition{{Name: "valid", Parameters: []tool.Parameter{{Name: "p", Type: tool.String}, {Name: "p", Type: tool.Number}}}}},
		{"invalid type", []tool.Definition{{Name: "valid", Parameters: []tool.Parameter{{Name: "p", Type: tool.InvalidType}}}}},
		{"unknown type", []tool.Definition{{Name: "valid", Parameters: []tool.Parameter{{Name: "p", Type: tool.ValueType(7)}}}}},
		{"too many parameters", fcrDefinitions(1, 33)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, catalogErr := tool.NewCatalog(tc.defs)
			if catalogErr != tool.ErrInvalidCatalog {
				t.Fatal("fixture does not violate Stage A contract")
			}
			fcrReject(t, fcrRequest(), tc.defs, catalogErr)
		})
	}
}

func TestFunctionCallRequestOwnershipAndZeroValue(t *testing.T) {
	var zero FunctionCallRequest
	if zero.Body() != nil || zero.Catalog() != nil {
		t.Fatal("zero value exposes request or authority")
	}
	if _, err := MapFunctionCallResponse([]byte(`{}`), zero.Catalog()); err != tool.ErrInvalidCatalog {
		t.Fatal("zero catalog did not fail closed")
	}
	parameters := []tool.Parameter{{Name: "text", Type: tool.String, Required: true}}
	definitions := []tool.Definition{{Name: "propose", Parameters: parameters}}
	request := fcrRequest()
	r := fcrBuild(t, request, definitions)
	want := r.Body()
	request.Text = "replaced"
	request.Model = "replaced"
	request.MaxOutputTokens = 16
	parameters[0].Name = "replacement"
	parameters[0].Type = tool.Number
	parameters[0].Required = false
	definitions[0].Name = "replacement"
	definitions[0].Parameters = []tool.Parameter{{Name: "new", Type: tool.Null}}
	definitions = append(definitions, tool.Definition{Name: "extra"})
	if !bytes.Equal(r.Body(), want) {
		t.Fatal("caller mutation changed serialized body")
	}
	copyOfBody := r.Body()
	for i := range copyOfBody {
		copyOfBody[i] = 'x'
	}
	for range 3 {
		if !bytes.Equal(r.Body(), want) {
			t.Fatal("Body returned writable internal storage")
		}
	}
	for _, tc := range []struct {
		name, args string
		want       tool.Reason
	}{
		{"propose", `{"text":"value"}`, tool.Allowed},
		{"propose", `{}`, tool.InvalidArguments},
		{"propose", `{"text":1}`, tool.InvalidArguments},
		{"propose", `{"replacement":1}`, tool.InvalidArguments},
		{"replacement", `{}`, tool.UnknownTool},
		{"extra", `{}`, tool.UnknownTool},
	} {
		d, err := r.Catalog().Admit(tool.Call{Name: tc.name, Arguments: []byte(tc.args)})
		if err != nil || d.Reason() != tc.want {
			t.Fatal("caller mutation changed catalog admission")
		}
	}
	for _, value := range []FunctionCallRequest{zero, r} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			for _, rendered := range []string{fmt.Sprintf(format, value), fmt.Sprintf(format, &value)} {
				for _, private := range []string{"offline proposal", "fixture-model", "propose", "properties", "text"} {
					if strings.Contains(rendered, private) {
						t.Fatal("diagnostic exposed request or definitions")
					}
				}
			}
		}
	}
}

func TestFunctionCallRequestDeterminism(t *testing.T) {
	definitions := []tool.Definition{
		{Name: "z_tool", Parameters: []tool.Parameter{{Name: "z", Type: tool.Null, Required: true}, {Name: "m", Type: tool.Object}, {Name: "a", Type: tool.Array, Required: true}}},
		{Name: "a_tool", Parameters: []tool.Parameter{{Name: "b", Type: tool.Boolean}, {Name: "a", Type: tool.Number}}},
	}
	want := fcrBuild(t, fcrRequest(), definitions).Body()
	for range 5 {
		if got := fcrBuild(t, fcrRequest(), definitions).Body(); !bytes.Equal(got, want) {
			t.Fatal("repeated build changed canonical bytes")
		}
	}
	slices.Reverse(definitions)
	if got := fcrBuild(t, fcrRequest(), definitions).Body(); !bytes.Equal(got, want) {
		t.Fatal("definition order changed canonical bytes")
	}
	for i := range definitions {
		slices.Reverse(definitions[i].Parameters)
	}
	if got := fcrBuild(t, fcrRequest(), definitions).Body(); !bytes.Equal(got, want) {
		t.Fatal("parameter order changed canonical bytes")
	}
}

func TestFunctionCallRequestBounds(t *testing.T) {
	// Stage A's documented maxima plus JSON escape expansion remain bounded.
	definitions := fcrDefinitions(64, 32)
	for i := range definitions {
		definitions[i].Name += strings.Repeat("t", 64-len(definitions[i].Name))
		for j := range definitions[i].Parameters {
			p := &definitions[i].Parameters[j]
			p.Name += strings.Repeat("p", 64-len(p.Name))
			p.Type = tool.Boolean // longest serialized JSON type name
		}
	}
	request := ai.Request{Text: strings.Repeat("\x00", ai.MaxInputBytes-1) + "x", Model: strings.Repeat("m", ai.MaxModelBytes), MaxOutputTokens: ai.MaxOutputTokens}
	r := fcrBuild(t, request, definitions)
	if len(r.Body()) > maxBody {
		t.Fatal("serialized request exceeds adapter bound")
	}
	body := fcrJSON(t, r.Body()).(map[string]any)
	if body["input"] != request.Text || body["model"] != request.Model || len(body["tools"].([]any)) != 64 {
		t.Fatal("bounded request was truncated or rewritten")
	}
}

func TestFunctionCallRequestB1Contract(t *testing.T) {
	definitions := []tool.Definition{{Name: "propose", Parameters: []tool.Parameter{
		{Name: "required", Type: tool.String, Required: true},
		{Name: "optional", Type: tool.Number},
		{Name: "object", Type: tool.Object},
		{Name: "array", Type: tool.Array},
		{Name: "null", Type: tool.Null},
	}}}
	r := fcrBuild(t, fcrRequest(), definitions)
	declaration := fcrJSON(t, r.Body()).(map[string]any)["tools"].([]any)[0].(map[string]any)
	schema := declaration["parameters"].(map[string]any)
	if !reflect.DeepEqual(schema["required"], []any{"required"}) || schema["additionalProperties"] != false {
		t.Fatal("provider declaration differs from local required/closed contract")
	}
	for _, tc := range []struct {
		name, arguments string
		want            tool.Reason
	}{
		{"valid required", `{"required":"value"}`, tool.Allowed},
		{"optional absent", `{"required":""}`, tool.Allowed},
		{"optional present", `{"required":"value","optional":1.5}`, tool.Allowed},
		{"unknown root parameter", `{"required":"value","unknown":true}`, tool.InvalidArguments},
		{"wrong arguments root", `[]`, tool.InvalidArguments},
		{"wrong root parameter type", `{"required":123}`, tool.InvalidArguments},
		{"missing required", `{"optional":1}`, tool.InvalidArguments},
		{"nested object", `{"required":"value","object":{"open":{"nested":[true,null,1]}}}`, tool.Allowed},
		{"nested array", `{"required":"value","array":[{},[],"text",false,null,2]}`, tool.Allowed},
		{"explicit null", `{"required":"value","null":null}`, tool.Allowed},
		{"optional does not mean nullable", `{"required":"value","optional":null}`, tool.InvalidArguments},
		{"null does not coerce", `{"required":"value","null":"null"}`, tool.InvalidArguments},
		{"nested traversal remains bounded", `{"required":"value","array":[[[[[[[[]]]]]]]]}`, tool.PolicyRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture, err := json.Marshal(map[string]any{
				"status": "completed", "error": nil, "incomplete_details": nil,
				"output": []any{map[string]any{
					"type": "function_call", "status": "completed", "id": "item_fixture", "call_id": "call_fixture",
					"name": declaration["name"], "arguments": tc.arguments,
				}},
			})
			// Use the provider declaration's serialized name with the SAME catalog.
			if err != nil {
				t.Fatal(err)
			}
			d, err := MapFunctionCallResponse(fixture, r.Catalog())
			if err != nil || d.Admission().Reason() != tc.want {
				t.Fatalf("admission = %v; error = %v; want reason %d", d, err, tc.want)
			}
			wantStatus := tool.Rejected
			if tc.want == tool.Allowed {
				wantStatus = tool.Admitted
			}
			if d.Admission().Status() != wantStatus {
				t.Fatal("incorrect admission status")
			}
			call, admitted := d.Admission().AdmittedCall()
			if admitted != (wantStatus == tool.Admitted) {
				t.Fatal("incorrect admitted-call availability")
			}
			if admitted && (call.Name != declaration["name"] || string(call.Arguments) != tc.arguments) {
				t.Fatal("admission changed proposal data")
			}
		})
	}
}
