package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

const (
	fcNameCanary     = "function_name_canary"
	fcArgsCanary     = "ARGUMENT_CANARY_842"
	fcItemCanary     = "ITEM_ID_CANARY_842"
	fcCallCanary     = "CALL_ID_CANARY_842"
	fcErrorCanary    = "PROVIDER_ERROR_CANARY_842"
	fcMetadataCanary = "IGNORED_METADATA_CANARY_842"
)

func fcCatalog(t *testing.T, parameters ...tool.Parameter) *tool.Catalog {
	t.Helper()
	catalog, err := tool.NewCatalog([]tool.Definition{{Name: fcNameCanary, Parameters: parameters}})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func fcItem(arguments string) map[string]any {
	return map[string]any{
		"type": "function_call", "status": "completed",
		"id": fcItemCanary, "call_id": fcCallCanary,
		"name": fcNameCanary, "arguments": arguments,
	}
}

func fcResponse(item any) map[string]any {
	return map[string]any{
		"status": "completed", "error": nil, "incomplete_details": nil,
		"output": []any{item}, "metadata": map[string]any{"ignored": fcMetadataCanary},
	}
}

func fcBytes(value any) []byte { return []byte(encoded(value)) }

func fcCheckPrivacy(t *testing.T, d FunctionCallDecision, err error) {
	t.Helper()
	formatted := []string{d.String(), d.GoString()}
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		formatted = append(formatted, fmt.Sprintf(format, d), fmt.Sprintf(format, &d))
	}
	if err != nil {
		formatted = append(formatted, err.Error(), fmt.Sprintf("%v %+v %#v", err, err, err))
		// Mapper returns safe sentinels themselves, never raw causes/wrappers.
		if errors.Unwrap(err) != nil {
			t.Fatal("raw error cause exposed")
		}
		if _, joined := err.(interface{ Unwrap() []error }); joined {
			t.Fatal("unexpected public error tree")
		}
	}
	for _, text := range formatted {
		for _, secret := range []string{fcNameCanary, fcArgsCanary, fcItemCanary, fcCallCanary, fcErrorCanary, fcMetadataCanary} {
			if strings.Contains(text, secret) {
				t.Fatal("diagnostic leaked untrusted data")
			}
		}
	}
}

func fcAssertError(t *testing.T, data []byte, catalog *tool.Catalog, want error) {
	t.Helper()
	d, err := MapFunctionCallResponse(data, catalog)
	if !errors.Is(err, want) || err != want {
		t.Fatalf("error category = %v; want %v", err, want)
	}
	if !reflect.DeepEqual(d, FunctionCallDecision{}) ||
		d.Admission().Status() != tool.Rejected || d.Admission().Reason() != tool.NotEvaluated {
		t.Fatal("error returned partial decision")
	}
	if itemID, callID, ok := d.Correlation(); ok || itemID != "" || callID != "" || d.itemID != "" || d.callID != "" {
		t.Fatal("error retained correlation")
	}
	if call, ok := d.Admission().AdmittedCall(); ok || call.Name != "" || call.Arguments != nil {
		t.Fatal("error exposed proposal")
	}
	fcCheckPrivacy(t, d, err)
}

func fcAssertDecision(t *testing.T, data []byte, catalog *tool.Catalog, want tool.Reason) FunctionCallDecision {
	t.Helper()
	d, err := MapFunctionCallResponse(data, catalog)
	if err != nil || d.Admission().Reason() != want {
		t.Fatalf("decision = %v; error = %v; want reason %d", d, err, want)
	}
	wantStatus := tool.Rejected
	if want == tool.Allowed {
		wantStatus = tool.Admitted
	}
	if d.Admission().Status() != wantStatus {
		t.Fatal("wrong admission status")
	}
	if want != tool.Allowed {
		if itemID, callID, ok := d.Correlation(); ok || itemID != "" || callID != "" || d.itemID != "" || d.callID != "" {
			t.Fatal("rejection retained IDs")
		}
		if call, ok := d.Admission().AdmittedCall(); ok || call.Name != "" || call.Arguments != nil {
			t.Fatal("rejection exposed proposal")
		}
	}
	fcCheckPrivacy(t, d, err)
	return d
}

func TestFunctionCallDecisionBaseline(t *testing.T) {
	var zero FunctionCallDecision
	if zero.Admission().Status() != tool.Rejected || zero.Admission().Reason() != tool.NotEvaluated {
		t.Fatal("zero decision does not fail closed")
	}
	if itemID, callID, ok := zero.Correlation(); ok || itemID != "" || callID != "" {
		t.Fatal("zero correlation")
	}
	if _, ok := zero.Admission().AdmittedCall(); ok {
		t.Fatal("zero exposes call")
	}
	fcCheckPrivacy(t, zero, nil)
	if zero.String() != "OpenAI function-call mapping: tool admission: REJECTED / NOT_EVALUATED" {
		t.Fatal("zero diagnostic lost mapping/admission classification")
	}
	fcAssertError(t, bytes.Repeat([]byte{0xff}, maxBody+1), nil, tool.ErrInvalidCatalog)
	empty, err := tool.NewCatalog(nil)
	if err != nil {
		t.Fatal(err)
	}
	var zeroCatalog tool.Catalog
	for _, catalog := range []*tool.Catalog{empty, &zeroCatalog} {
		fcAssertDecision(t, fcBytes(fcResponse(fcItem("{}"))), catalog, tool.UnknownTool)
	}
	d := fcAssertDecision(t, fcBytes(fcResponse(fcItem("{}"))), fcCatalog(t), tool.Allowed)
	if itemID, callID, ok := d.Correlation(); !ok || itemID != fcItemCanary || callID != fcCallCanary {
		t.Fatal("admitted correlation mismatch")
	}
	if d.String() != "OpenAI function-call mapping: tool admission: ADMITTED / ALLOWED" {
		t.Fatal("diagnostic obscures admission-only outcome")
	}
}

func TestFunctionCallResponseStatuses(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  error
	}{
		{"failed", "failed", ai.ErrProvider}, {"cancelled", "cancelled", ai.ErrProvider},
		{"incomplete", "incomplete", ai.ErrIncompleteResponse},
		{"queued", "queued", ai.ErrIncompleteResponse}, {"progress", "in_progress", ai.ErrIncompleteResponse},
		{"missing", nil, ai.ErrMalformedResponse}, {"null", nil, ai.ErrMalformedResponse},
		{"number", 1, ai.ErrMalformedResponse}, {"object", map[string]any{}, ai.ErrMalformedResponse},
		{"unknown", "COMPLETED", ai.ErrMalformedResponse}, {"empty", "", ai.ErrMalformedResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Non-completed statuses do not require the completed-only fields.
			obj := map[string]any{"status": tc.value}
			if tc.name == "missing" {
				delete(obj, "status")
			}
			fcAssertError(t, fcBytes(obj), fcCatalog(t), tc.want)
		})
	}
	obj := fcResponse(fcItem("{}"))
	delete(obj, "status")
	obj["Status"] = "completed"
	fcAssertError(t, fcBytes(obj), fcCatalog(t), ai.ErrMalformedResponse)
}

func TestFunctionCallCompletedNullables(t *testing.T) {
	for _, field := range []string{"error", "incomplete_details"} {
		for _, tc := range []struct {
			name    string
			value   any
			missing bool
			want    error
		}{
			{"null", nil, false, nil}, {"missing", nil, true, ai.ErrMalformedResponse},
			{"number", 1, false, ai.ErrMalformedResponse}, {"string", fcErrorCanary, false, ai.ErrMalformedResponse},
			{"bool", false, false, ai.ErrMalformedResponse}, {"array", []any{}, false, ai.ErrMalformedResponse},
			{"object", map[string]any{"message": fcErrorCanary}, false, ai.ErrProvider},
			{"empty object", map[string]any{}, false, ai.ErrProvider},
			{"reason string", map[string]any{"reason": fcErrorCanary}, false, ai.ErrProvider},
			{"reason null", map[string]any{"reason": nil}, false, ai.ErrProvider},
			{"reason number", map[string]any{"reason": 1}, false, ai.ErrProvider},
			{"reason array", map[string]any{"reason": []any{}}, false, ai.ErrProvider},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				obj := fcResponse(fcItem("{}"))
				obj[field] = tc.value
				if tc.missing {
					delete(obj, field)
				}
				want := tc.want
				if field == "incomplete_details" && tc.want == ai.ErrProvider {
					want = ai.ErrMalformedResponse
					if tc.name == "reason string" {
						want = ai.ErrIncompleteResponse
					}
				}
				if want == nil {
					fcAssertDecision(t, fcBytes(obj), fcCatalog(t), tool.Allowed)
				} else {
					fcAssertError(t, fcBytes(obj), fcCatalog(t), want)
				}
			})
		}
	}
	obj := fcResponse(fcItem("{}"))
	obj["error"] = map[string]any{"message": fcErrorCanary}
	delete(obj, "incomplete_details")
	obj["output"] = []any{}
	fcAssertError(t, fcBytes(obj), fcCatalog(t), ai.ErrProvider)
	delete(obj, "error")
	obj["incomplete_details"] = map[string]any{"reason": "max_output_tokens"}
	fcAssertError(t, fcBytes(obj), fcCatalog(t), ai.ErrMalformedResponse)
}

func TestFunctionCallOutputComposition(t *testing.T) {
	for _, tc := range []struct {
		name    string
		output  any
		missing bool
	}{
		{"missing", nil, true}, {"null", nil, false}, {"string", "output", false},
		{"object", fcItem("{}"), false}, {"empty", []any{}, false},
		{"null item", []any{nil}, false}, {"string item", []any{"item"}, false},
		{"array item", []any{[]any{}}, false},
		{"two calls", []any{fcItem("{}"), fcItem("{}")}, false},
		{"message and call", []any{message("dummy"), fcItem("{}")}, false},
		{"call and message", []any{fcItem("{}"), message("dummy")}, false},
		{"reasoning and call", []any{map[string]any{"type": "reasoning"}, fcItem("{}")}, false},
		{"call and unsupported", []any{fcItem("{}"), map[string]any{"type": "mcp_call"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := fcResponse(fcItem("{}"))
			obj["output"] = tc.output
			if tc.missing {
				delete(obj, "output")
			}
			fcAssertError(t, fcBytes(obj), fcCatalog(t), ai.ErrMalformedResponse)
		})
	}
	for _, kind := range []string{
		"message", "reasoning", "function_call_output", "web_search_call", "file_search_call",
		"computer_call", "computer_call_output", "mcp_call", "mcp_list_tools",
		"mcp_approval_request", "mcp_approval_response", "shell_call", "shell_call_output",
		"local_shell_call", "local_shell_call_output", "code_interpreter_call",
		"apply_patch_call", "apply_patch_call_output", "custom_tool_call", "custom_tool_call_output",
		"program", "program_output", "tool_search_call", "tool_search_output",
		"additional_tools", "compaction", "image_generation_call", "future_output",
	} {
		t.Run(kind, func(t *testing.T) {
			item := fcItem("{}")
			item["type"] = kind
			fcAssertError(t, fcBytes(fcResponse(item)), fcCatalog(t), ai.ErrMalformedResponse)
		})
	}
}

func TestFunctionCallRequiredFields(t *testing.T) {
	for _, field := range []string{"type", "status", "id", "call_id", "name", "arguments"} {
		for _, tc := range []struct {
			name    string
			value   any
			missing bool
		}{
			{"missing", nil, true}, {"null", nil, false}, {"number", 1, false},
			{"bool", false, false}, {"object", map[string]any{}, false}, {"array", []any{}, false},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				item := fcItem("{}")
				item[field] = tc.value
				if tc.missing {
					delete(item, field)
				}
				fcAssertError(t, fcBytes(fcResponse(item)), fcCatalog(t), ai.ErrMalformedResponse)
			})
		}
		item := fcItem("{}")
		value := item[field]
		delete(item, field)
		item[strings.ToUpper(field)] = value
		fcAssertError(t, fcBytes(fcResponse(item)), fcCatalog(t), ai.ErrMalformedResponse)
	}
	for _, state := range []string{"in_progress", "incomplete", "queued", "failed", "cancelled", "", "COMPLETED"} {
		item := fcItem("{}")
		item["status"] = state
		want := ai.ErrMalformedResponse
		if state == "in_progress" || state == "incomplete" {
			want = ai.ErrIncompleteResponse
		}
		fcAssertError(t, fcBytes(fcResponse(item)), fcCatalog(t), want)
	}
}

func TestFunctionCallCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		value       any
		allowed     bool
	}{
		{"async false", "async", false, true}, {"async true", "async", true, false},
		{"async null", "async", nil, false}, {"async string", "async", "false", false},
		{"async number", "async", 0, false},
		{"namespace empty", "namespace", "", false}, {"namespace named", "namespace", "ns", false},
		{"namespace object", "namespace", map[string]any{}, false}, {"namespace null", "namespace", nil, false},
		{"direct", "caller", map[string]any{"type": "direct"}, true},
		{"program", "caller", map[string]any{"type": "program", "caller_id": "p"}, false},
		{"unknown caller", "caller", map[string]any{"type": "unknown"}, false},
		{"caller sibling", "caller", map[string]any{"type": "direct", "extra": true}, false},
		{"caller null", "caller", nil, false}, {"caller string", "caller", "direct", false},
		{"caller array", "caller", []any{}, false}, {"caller missing type", "caller", map[string]any{}, false},
		{"caller null type", "caller", map[string]any{"type": nil}, false},
		{"caller numeric type", "caller", map[string]any{"type": 1}, false},
		{"caller case", "caller", map[string]any{"Type": "direct"}, false},
		{"unknown field", "new_capability", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := fcItem("{}")
			item[tc.field] = tc.value
			if tc.allowed {
				fcAssertDecision(t, fcBytes(fcResponse(item)), fcCatalog(t), tool.Allowed)
			} else {
				fcAssertError(t, fcBytes(fcResponse(item)), fcCatalog(t), ai.ErrMalformedResponse)
			}
		})
	}
}

func TestFunctionCallIDsAndAuthority(t *testing.T) {
	for _, field := range []string{"id", "call_id"} {
		for _, id := range []string{"x", strings.Repeat("x", 256), strings.Repeat("é", 128), "opaque/../value", "é", "e\u0301"} {
			item := fcItem("{}")
			item[field] = id
			d := fcAssertDecision(t, fcBytes(fcResponse(item)), fcCatalog(t), tool.Allowed)
			itemID, callID, ok := d.Correlation()
			got := itemID
			if field == "call_id" {
				got = callID
			}
			if !ok || got != id {
				t.Fatal("ID rewritten")
			}
		}
		for i, id := range []string{"", strings.Repeat("x", 257), strings.Repeat("é", 128) + "x",
			" x", "x ", "x y", "x\t", "x\n", "\u00a0x", "x\u2003y", "x\u2028",
			"x\x00", "x\x1b", "x\x1f", "x\x7f", "x\u0080", "x\u009f"} {
			t.Run(fmt.Sprintf("%s/invalid%d", field, i), func(t *testing.T) {
				item := fcItem("{}")
				item[field] = id
				fcAssertError(t, fcBytes(fcResponse(item)), fcCatalog(t), ai.ErrMalformedResponse)
			})
		}
	}
	item := fcItem("{}")
	item["name"] = "unknown"
	item["id"], item["call_id"] = fcNameCanary, fcNameCanary
	fcAssertDecision(t, fcBytes(fcResponse(item)), fcCatalog(t), tool.UnknownTool)
}

func TestFunctionCallAdmissionAndExactArguments(t *testing.T) {
	catalog := fcCatalog(t, tool.Parameter{Name: "text", Type: tool.String, Required: true})
	for _, tc := range []struct {
		name      string
		arguments string
		want      tool.Reason
	}{
		{fcNameCanary, `{"text":"dummy"}`, tool.Allowed},
		{"unknown", "{broken", tool.UnknownTool},
		{"unknown", strings.Repeat("x", 16385), tool.UnknownTool},
		{"", "{}", tool.MalformedCall}, {"invalid name", "{}", tool.MalformedCall},
		{strings.Repeat("a", 65), "{}", tool.MalformedCall},
		{"é", "{}", tool.MalformedCall}, {"*", "{}", tool.MalformedCall},
		{strings.ToUpper(fcNameCanary), "{}", tool.UnknownTool},
		{fcNameCanary, `{"text":"` + fcArgsCanary, tool.InvalidArguments},
		{fcNameCanary, `{"text":1}`, tool.InvalidArguments},
		{fcNameCanary, "{}", tool.InvalidArguments},
		{fcNameCanary, `{"text":"ok","extra":null}`, tool.InvalidArguments},
		{fcNameCanary, "{}" + strings.Repeat(" ", 16383), tool.PolicyRejected},
		{fcNameCanary, `{"text":"\uD800"}`, tool.InvalidArguments},
		{fcNameCanary, `{"text":"\uDC00"}`, tool.InvalidArguments},
		{fcNameCanary, `{"text":"\\uD800"}`, tool.Allowed},
		{fcNameCanary, `{"text":"\uD83D\uDE00"}`, tool.Allowed},
		{fcNameCanary, `{"text":"�"}`, tool.Allowed},
	} {
		item := fcItem(tc.arguments)
		item["name"] = tc.name
		d := fcAssertDecision(t, fcBytes(fcResponse(item)), catalog, tc.want)
		if tc.want == tool.Allowed {
			call, ok := d.Admission().AdmittedCall()
			if !ok || call.Name != tc.name || string(call.Arguments) != tc.arguments {
				t.Fatal("logical proposal changed")
			}
		}
	}
	catalog = fcCatalog(t, tool.Parameter{Name: "b", Type: tool.Number}, tool.Parameter{Name: "text", Type: tool.String})
	raw := " \t" + `{"b":1e+02,"te\u0078t":"\uD83D\uDE00"}` + "\r\n "
	data := fcBytes(fcResponse(fcItem(raw)))
	d := fcAssertDecision(t, data, catalog, tool.Allowed)
	for i := range data {
		data[i] = 'x'
	}
	first, ok := d.Admission().AdmittedCall()
	if !ok || string(first.Arguments) != raw {
		t.Fatal("input aliased or rewritten")
	}
	for i := range first.Arguments {
		first.Arguments[i] = 'y'
	}
	first.Name = "replacement"
	second, _ := d.Admission().AdmittedCall()
	third, _ := d.Admission().AdmittedCall()
	second.Arguments[0] = 'z'
	if string(third.Arguments) != raw || third.Name != fcNameCanary {
		t.Fatal("accessor alias")
	}
	if itemID, callID, ok := d.Correlation(); !ok || itemID != fcItemCanary || callID != fcCallCanary {
		t.Fatal("input mutation changed IDs")
	}
}

func TestFunctionCallStageATypesAndData(t *testing.T) {
	for _, tc := range []struct {
		kind  tool.ValueType
		value string
	}{
		{tool.String, `""`}, {tool.Number, "-1.25e+9"}, {tool.Boolean, "false"},
		{tool.Object, `{"nested":{"arbitrary":true}}`}, {tool.Array, `[1,"two",true,null,{},[]]`}, {tool.Null, "null"},
	} {
		raw := `{"v":` + tc.value + "}"
		d := fcAssertDecision(t, fcBytes(fcResponse(fcItem(raw))),
			fcCatalog(t, tool.Parameter{Name: "v", Type: tc.kind, Required: true}), tool.Allowed)
		call, _ := d.Admission().AdmittedCall()
		if string(call.Arguments) != raw {
			t.Fatal("typed arguments rewritten")
		}
	}
	objectCatalog := fcCatalog(t, tool.Parameter{Name: "v", Type: tool.Object})
	fcAssertDecision(t, fcBytes(fcResponse(fcItem(`{"v":{"text":1,"te\u0078t":2}}`))), objectCatalog, tool.InvalidArguments)
	fcAssertDecision(t, fcBytes(fcResponse(fcItem("{}"+strings.Repeat(" ", 16382)))), fcCatalog(t), tool.Allowed)
	for _, value := range []string{"rm -rf /", "Remove-Item -Recurse C:\\dummy", "C:\\dummy\\file", "https://example.invalid/dummy", "$(git status); cmd /c echo data"} {
		raw := `{"v":` + encoded(value) + "}"
		d := fcAssertDecision(t, fcBytes(fcResponse(fcItem(raw))), fcCatalog(t, tool.Parameter{Name: "v", Type: tool.String}), tool.Allowed)
		call, _ := d.Admission().AdmittedCall()
		if string(call.Arguments) != raw {
			t.Fatal("string data interpreted")
		}
	}
}

func TestFunctionCallPrivacyAndPrecedence(t *testing.T) {
	catalog := fcCatalog(t, tool.Parameter{Name: "text", Type: tool.String})
	for _, arguments := range []string{`{"text":"` + fcArgsCanary + `"}`, `{"text":"` + fcArgsCanary} {
		want := tool.Allowed
		if !json.Valid([]byte(arguments)) {
			want = tool.InvalidArguments
		}
		fcAssertDecision(t, fcBytes(fcResponse(fcItem(arguments))), catalog, want)
	}
	obj := fcResponse(fcItem(`{"text":"` + fcArgsCanary + `"}`))
	obj["error"] = map[string]any{"message": fcErrorCanary}
	fcAssertError(t, fcBytes(obj), catalog, ai.ErrProvider)
	fcAssertError(t, bytes.Repeat([]byte{0xff}, maxBody+1), catalog, ai.ErrResponseTooLarge)
	// Validate the entire envelope before interpreting early status failures.
	fcAssertError(t, []byte(`{"status":"failed","ignored":"\uD800"}`), catalog, ai.ErrMalformedResponse)
	fcAssertError(t, []byte(`{"status":"failed","ignored":`+strings.Repeat("[", 32)+"0"+strings.Repeat("]", 32)+"}"), catalog, ai.ErrResponseTooLarge)
}

func TestFunctionCallConcurrentDeterminism(t *testing.T) {
	catalog := fcCatalog(t, tool.Parameter{Name: "v", Type: tool.Number})
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < 24; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			raw := fmt.Sprintf(`{"v":%d}`, i)
			data := fcBytes(fcResponse(fcItem(raw)))
			for j := 0; j < 25; j++ {
				d := fcAssertDecision(t, data, catalog, tool.Allowed)
				call, _ := d.Admission().AdmittedCall()
				if string(call.Arguments) != raw {
					t.Error("cross-call state")
					return
				}
				call.Arguments[0] = 'x'
				fcAssertDecision(t, fcBytes(fcResponse(fcItem(`{"v":"wrong"}`))), catalog, tool.InvalidArguments)
			}
		}(i)
	}
	close(start)
	workers.Wait()
	data := fcBytes(fcResponse(fcItem(`{"v":1e+02}`)))
	first, err := MapFunctionCallResponse(data, catalog)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		next, err := MapFunctionCallResponse(data, catalog)
		if err != nil || !reflect.DeepEqual(first, next) {
			t.Fatal("nondeterministic mapping")
		}
	}
}

func TestFunctionCallTextPathRegression(t *testing.T) {
	for _, isCall := range []bool{false, true} {
		obj := responseObject("unchanged text")
		if isCall {
			obj["output"] = []any{fcItem("{}")}
		}
		body := &trackedBody{reader: bytes.NewReader(fcBytes(obj))}
		calls := 0
		client := responseClient(t, 200, body, &calls)
		result, err := client.Execute(context.Background(), request())
		if isCall {
			requireFailure(t, result, err, ai.ErrMalformedResponse)
		} else if err != nil || result.Text != "unchanged text" {
			t.Fatal("text success changed")
		}
		if calls != 1 || body.closed != 1 {
			t.Fatal("request count/closure changed")
		}
	}
}

// This audit reads source in tests only. The two B1 production files themselves
// have an explicit import/call allowlist and cannot reach the existing Client.
func TestFunctionCallProductionBoundary(t *testing.T) {
	allowedImports := map[string]bool{
		"bytes": true, "encoding/json": true, "io": true, "strings": true,
		"unicode": true, "unicode/utf8": true,
		"github.com/kaizenforyou91/forge/pkg/ai":      true,
		"github.com/kaizenforyou91/forge/pkg/ai/tool": true,
	}
	allowedCalls := map[string]bool{
		"len": true, "make": true, "string": true, "uint16": true,
		"validateFunctionCallJSON": true, "functionCallSurrogatesValid": true,
		"functionCallHexCode": true, "validFunctionCallID": true,
		"jsonObject": true, "jsonString": true, "isNull": true,
		"bytes.Clone": true, "bytes.Equal": true, "bytes.TrimSpace": true,
		"bytes.HasPrefix": true, "bytes.NewReader": true,
		"json.Unmarshal": true, "json.Valid": true, "json.NewDecoder": true, "json.Delim": true,
		"strings.Clone": true, "unicode.IsSpace": true, "utf8.Valid": true, "utf8.ValidString": true,
		"decoder.UseNumber": true, "decoder.Token": true, "parser.object": true,
		"p.decoder.More": true, "p.decoder.Token": true, "p.value": true, "p.object": true, "p.array": true,
		"catalog.Admit": true, "admission.Status": true, "admission.Reason": true,
		"d.admission.Status": true, "d.admission.Reason": true, "d.admission.String": true, "d.String": true,
	}
	admitCalls := 0
	for _, filename := range []string{"function_call.go", "function_call_json.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil || !allowedImports[path] {
				t.Fatal("unexpected B1 import")
			}
		}
		for _, declaration := range file.Decls {
			if declaration, ok := declaration.(*ast.GenDecl); ok && declaration.Tok == token.VAR {
				t.Fatal("B1 declares mutable global state")
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.GoStmt, *ast.FuncLit:
				t.Error("B1 starts work or introduces callback")
			case *ast.CallExpr:
				if _, conversion := node.Fun.(*ast.ArrayType); conversion {
					break
				} // []byte conversion
				name := fcCallName(node.Fun)
				if !allowedCalls[name] {
					t.Errorf("unexpected B1 call: %s", name)
				}
				if name == "catalog.Admit" {
					admitCalls++
				}
			}
			return true
		})
	}
	if admitCalls != 1 {
		t.Fatal("mapper must have exactly one catalog.Admit call site")
	}
}

func fcCallName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.SelectorExpr:
		return fcCallName(expression.X) + "." + expression.Sel.Name
	}
	return ""
}
