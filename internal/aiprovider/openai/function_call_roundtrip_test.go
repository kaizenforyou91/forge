package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// Every fixture uses synthetic secrets; no environment or credential file is read.
var frDiagnosticCanaries = []string{
	canary, "private_prompt_C10", "private_argument", "call_fixture", "item_private",
	"private_reasoning_C10", "private_encrypted_C10", "private_tool_output_C10",
	"private_provider_error_C10", "private_response_id_C10", "private_model_output_C10",
}

func frDiagnosticSafe(t *testing.T, result ai.Result, stage FunctionRoundTripStage, err error) {
	t.Helper()
	observable := fmt.Sprintf("%+v %#v %s %+v %#v", result, result, stage.String(), err, err)
	for _, secret := range frDiagnosticCanaries {
		if strings.Contains(observable, secret) {
			t.Fatal("diagnostic exposed a canary")
		}
	}
}

func TestFunctionRoundTripDiagnosticFailures(t *testing.T) {
	for _, name := range []string{
		"build", "authority", "post1 transport", "post1 provider", "post1 malformed",
		"map", "B1 reasoning mixed", "admission", "reflection", "usage",
		"handler", "handler panic", "post2 transport", "post2 provider", "post2 decode",
		"post2 validate", "post2 reflection", "post1 validate", "aggregate", "second tool",
	} {
		t.Run(name, func(t *testing.T) {
			var ordinaryResult ai.Result
			var ordinaryError error
			var ordinaryWires [][]byte
			for _, diagnostic := range []bool{false, true} {
				first, second := frFunction(), responseObject("private_model_output_C10")
				first["id"], second["id"] = "private_response_id_C10", "private_response_id_C10"
				item := first["output"].([]any)[0].(map[string]any)
				reasoning := map[string]any{"type": "reasoning", "summary": []any{},
					"id": "private_reasoning_C10", "encrypted_content": "private_encrypted_C10"}
				second["output"] = append([]any{reasoning}, second["output"].([]any)...)
				first["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1}
				second["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1}
				req := request()
				req.Text = "private_prompt_C10"
				wantStage, wantErr := StagePost1Map, ai.ErrMalformedResponse
				posts, handlers := 1, 0
				var handler tool.Handler
				switch name {
				case "build", "authority":
					wantStage, wantErr, posts = StageRequestBuild, ai.ErrInvalidRequest, 0
					if name == "build" {
						req.MaxOutputTokens = 0
					}
				case "post1 transport":
					wantStage, wantErr = StagePost1Request, ai.ErrTransport
				case "post1 provider":
					wantStage, wantErr = StagePost1Request, ai.ErrProvider
				case "post1 malformed":
					first["output"] = []any{reasoning}
					wantStage = StagePost1TextDecode
				case "map":
					delete(item, "call_id")
				case "B1 reasoning mixed":
					first["output"] = []any{reasoning, item}
				case "admission":
					item["name"] = "unknown"
					wantStage, wantErr = StagePost1Admission, ai.ErrInvalidRequest
				case "reflection":
					item["arguments"] = encoded(map[string]any{"value": canary})
					wantStage = StagePost1Reflection
				case "usage":
					first["usage"] = map[string]any{"input_tokens": -1, "output_tokens": 1}
					wantStage = StagePost1Usage
				case "handler", "handler panic":
					wantStage, wantErr, handlers = StageHandlerExecute, ai.ErrProvider, 1
					handler = func(context.Context, tool.Call) (string, error) {
						if name == "handler panic" {
							panic("private_provider_error_C10")
						}
						return "private_tool_output_C10", errors.New("private_provider_error_C10")
					}
				case "post1 validate":
					first = responseObject(canary)
					wantStage = StagePost1Validate
				default:
					posts, handlers = 2, 1
					switch name {
					case "post2 transport":
						wantStage, wantErr = StagePost2Request, ai.ErrTransport
					case "post2 provider":
						wantStage, wantErr = StagePost2Request, ai.ErrProvider
					case "post2 decode":
						second["output"] = []any{reasoning}
						wantStage = StagePost2Decode
					case "second tool":
						second = frFunction()
						wantStage = StagePost2Decode
					case "post2 validate":
						second["usage"] = map[string]any{"input_tokens": 1, "output_tokens": req.MaxOutputTokens + 1}
						wantStage = StagePost2Validate
					case "post2 reflection":
						second["model"] = canary
						wantStage = StagePost2Validate
					case "aggregate":
						first["usage"] = map[string]any{"input_tokens": int64(math.MaxInt64), "output_tokens": 1}
						wantStage = StageUsageAggregate
					}
				}
				steps := []frStep{{body: encoded(first)}, {body: encoded(second)}}
				for i, prefix := range []string{"post1", "post2"} {
					if name == prefix+" transport" {
						steps[i].err = errors.New(strings.Join(frDiagnosticCanaries, " "))
					}
					if name == prefix+" provider" {
						steps[i].status, steps[i].body = 500, strings.Join(frDiagnosticCanaries, " ")
					}
				}
				s := frClient(t, steps...)
				calls := 0
				authority, err := tool.NewAuthority([]tool.Binding{{Definition: fcoDefinition(), Handler: func(ctx context.Context, call tool.Call) (string, error) {
					calls++
					if calls > 1 {
						t.Fatal("handler retried")
					}
					if handler != nil {
						return handler(ctx, call)
					}
					return "private_tool_output_C10", nil
				}}})
				if err != nil {
					t.Fatal(err)
				}
				if name == "authority" {
					authority = nil
				}
				var result ai.Result
				stage := StageNone
				if diagnostic {
					result, stage, err = s.client.ExecuteAuthorizedFunctionRoundTripDiagnostic(context.Background(), req, authority)
					if stage != wantStage {
						t.Fatalf("stage = %s; want %s", stage, wantStage)
					}
					if !reflect.DeepEqual(result, ordinaryResult) || err.Error() != ai.SafeError(ordinaryError).Error() || !reflect.DeepEqual(s.wires, ordinaryWires) {
						t.Fatal("diagnostic changed result, classification or wire requests")
					}
				} else {
					result, err = s.client.ExecuteAuthorizedFunctionRoundTrip(context.Background(), req, authority)
					ordinaryResult, ordinaryError, ordinaryWires = result, err, s.wires
				}
				if !errors.Is(ai.SafeError(err), wantErr) || !reflect.DeepEqual(result, ai.Result{}) {
					t.Fatal("failure classification/result changed")
				}
				if len(s.wires) != posts || calls != handlers {
					t.Fatal("failure crossed effect bound")
				}
				frDiagnosticSafe(t, result, stage, err)
			}
		})
	}
}

func TestFunctionRoundTripDiagnosticSuccess(t *testing.T) {
	for _, direct := range []bool{false, true} {
		var baseline ai.Result
		var wires [][]byte
		for _, diagnostic := range []bool{false, true} {
			first, second := frFunction(), responseObject("final text")
			second["output"] = append([]any{frReasoning()}, second["output"].([]any)...)
			if direct {
				first = responseObject("direct text")
			}
			s := frClient(t, frStep{body: encoded(first)}, frStep{body: encoded(second)})
			calls := 0
			a, err := tool.NewAuthority([]tool.Binding{{Definition: fcoDefinition(), Handler: func(context.Context, tool.Call) (string, error) { calls++; return "local result", nil }}})
			if err != nil {
				t.Fatal(err)
			}
			var result ai.Result
			if diagnostic {
				var stage FunctionRoundTripStage
				result, stage, err = s.client.ExecuteAuthorizedFunctionRoundTripDiagnostic(context.Background(), request(), a)
				if stage != StageComplete || !reflect.DeepEqual(result, baseline) || !reflect.DeepEqual(wires, s.wires) {
					t.Fatal("diagnostic changed success")
				}
			} else {
				result, err = s.client.ExecuteAuthorizedFunctionRoundTrip(context.Background(), request(), a)
				baseline, wires = result, s.wires
			}
			if err != nil {
				t.Fatal(err)
			}
			if direct {
				if len(s.wires) != 1 || calls != 0 {
					t.Fatal("direct effects")
				}
			} else {
				if len(s.wires) != 2 || calls != 1 {
					t.Fatal("round-trip effects")
				}
				var payload map[string]any
				json.Unmarshal(s.wires[1], &payload)
				if payload["tool_choice"] != "none" {
					t.Fatal("continuation tool choice changed")
				}
			}
		}
	}
}

func TestFunctionRoundTripDiagnosticContext(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		for _, where := range []string{"initial", "first", "handler", "second"} {
			t.Run(fmt.Sprintf("%t/%s", deadline, where), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					stop := func() {
						if deadline {
							time.Sleep(time.Second)
							<-ctx.Done()
						} else {
							cancel()
						}
					}
					stage, posts, handlers := StageInitialContext, 0, 0
					steps := []frStep{{body: encoded(frFunction())}, {body: encoded(responseObject("final"))}}
					switch where {
					case "initial":
						stop()
					case "first":
						stage, posts = StagePost1Request, 1
						steps[0].before = func(*http.Request) { stop() }
					case "handler":
						stage, posts, handlers = StageHandlerExecute, 1, 1
					case "second":
						stage, posts, handlers = StagePost2Request, 2, 1
						steps[1].before = func(*http.Request) { stop() }
					}
					calls := 0
					a, err := tool.NewAuthority([]tool.Binding{{Definition: fcoDefinition(), Handler: func(context.Context, tool.Call) (string, error) {
						calls++
						if where == "handler" {
							stop()
						}
						return "private_tool_output_C10", nil
					}}})
					if err != nil {
						t.Fatal(err)
					}
					s := frClient(t, steps...)
					result, gotStage, err := s.client.ExecuteAuthorizedFunctionRoundTripDiagnostic(ctx, request(), a)
					wantErr := context.Canceled
					if deadline {
						wantErr = context.DeadlineExceeded
					}
					if gotStage != stage || !errors.Is(err, wantErr) || len(s.wires) != posts || calls != handlers {
						t.Fatal("context stage/category/bounds changed")
					}
					frDiagnosticSafe(t, result, gotStage, err)
				})
			})
		}
	}
}

type frStep struct {
	body                   string
	status                 int
	err, readErr, closeErr error
	before                 func(*http.Request)
}

type frScript struct {
	client   *Client
	wires    [][]byte
	contexts []context.Context
	bodies   []*trackedBody
}

func frClient(t *testing.T, steps ...frStep) *frScript {
	t.Helper()
	s := &frScript{}
	s.client = testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		i := len(s.wires)
		if i >= len(steps) || i >= 2 {
			t.Fatal("unexpected provider POST/retry")
		}
		data, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		s.wires = append(s.wires, data)
		s.contexts = append(s.contexts, r.Context())
		if r.Method != http.MethodPost || r.URL.String() != endpoint || r.GetBody != nil || r.ContentLength != int64(len(data)) || len(data) > maxBody {
			t.Fatal("bounded non-replayable POST contract changed")
		}
		if len(r.Header) != 3 || r.Header.Get("Authorization") != "Bearer "+canary || r.Header.Get("Content-Type") != "application/json" || !reflect.DeepEqual(r.Header["User-Agent"], []string{""}) {
			t.Fatal("request security headers changed")
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > ai.MaxTimeout {
			t.Fatal("missing operation deadline")
		}
		var payload map[string]any
		if json.Unmarshal(data, &payload) != nil {
			t.Fatal("invalid request JSON")
		}
		for _, field := range []string{"store", "stream", "background", "parallel_tool_calls"} {
			if payload[field] != false {
				t.Fatal("request privacy/control changed")
			}
		}
		for _, field := range []string{"previous_response_id", "conversation", "response_id", "include"} {
			if _, exists := payload[field]; exists {
				t.Fatal("stateful continuation field present")
			}
		}
		step := steps[i]
		if step.before != nil {
			step.before(r)
		}
		if step.err != nil {
			return nil, step.err
		}
		var reader io.Reader = strings.NewReader(step.body)
		if step.readErr != nil {
			reader = readerFunc(func([]byte) (int, error) { return 0, step.readErr })
		}
		body := &trackedBody{reader: reader, closeErr: step.closeErr}
		s.bodies = append(s.bodies, body)
		status := step.status
		if status == 0 {
			status = 200
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{endpoint}}, Body: body}, nil
	}))
	t.Cleanup(func() {
		for _, body := range s.bodies {
			if body.closed != 1 {
				t.Error("response body not closed exactly once")
			}
		}
	})
	return s
}

func frFunction() map[string]any {
	obj := responseObject("unused")
	obj["output"] = []any{map[string]any{
		"type": "function_call", "status": "completed", "id": "item_private", "call_id": "call_fixture",
		"name": "private_tool", "arguments": " {\"value\":\"private_argument\"} \n",
	}}
	return obj
}

func frExecutor(t *testing.T, definition tool.Definition, calls *int, handler tool.Handler) *tool.Executor {
	t.Helper()
	e, err := tool.NewExecutor([]tool.Binding{{Definition: definition, Handler: func(ctx context.Context, call tool.Call) (string, error) {
		*calls++
		if *calls > 1 {
			t.Fatal("handler retried")
		}
		if handler != nil {
			return handler(ctx, call)
		}
		return "local result", nil
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestFunctionRoundTripInputGuards(t *testing.T) {
	for _, name := range []string{"nil client", "nil http", "missing key", "nil ctx", "invalid request", "nil definitions", "empty definitions", "invalid definitions", "nil executor", "canceled"} {
		t.Run(name, func(t *testing.T) {
			s := frClient(t)
			c, ctx, req, defs := s.client, context.Background(), request(), []tool.Definition{fcoDefinition()}
			calls := 0
			e := frExecutor(t, fcoDefinition(), &calls, nil)
			want := ai.ErrInvalidRequest
			switch name {
			case "nil client":
				c = nil
			case "nil http":
				c.http = nil
			case "missing key":
				c.key = ""
			case "nil ctx":
				ctx = nil
			case "invalid request":
				req.Text = ""
			case "nil definitions":
				defs = nil
			case "empty definitions":
				defs = []tool.Definition{}
			case "invalid definitions":
				defs[0].Name = "bad name"
				want = tool.ErrInvalidCatalog
			case "nil executor":
				e = nil
			case "canceled":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
				want = context.Canceled
			}
			r, err := c.ExecuteFunctionRoundTrip(ctx, req, defs, e)
			requireFailure(t, r, err, want)
			if len(s.wires) != 0 || calls != 0 {
				t.Fatal("invalid input performed effects")
			}
		})
	}
}

func TestFunctionRoundTripDirectAndContinuation(t *testing.T) {
	for _, tc := range []struct {
		direct    bool
		reasoning int
	}{{true, 0}, {false, 0}, {false, 1}, {false, 2}} {
		direct := tc.direct
		first := responseObject("direct answer")
		first["usage"] = map[string]any{"input_tokens": 10, "output_tokens": 2}
		if !direct {
			first = frFunction()
			first["usage"] = map[string]any{"input_tokens": 10, "output_tokens": 2}
		}
		final := responseObject("provider final answer")
		for i := 0; i < tc.reasoning; i++ {
			final["output"] = append([]any{frReasoning()}, final["output"].([]any)...)
		}
		finalMessage := final["output"].([]any)[tc.reasoning].(map[string]any)
		finalMessage["id"], finalMessage["phase"] = "message_fixture", "final_answer"
		finalMessage["content"] = []any{
			map[string]any{"type": "output_text", "text": "provider final "},
			map[string]any{"type": "output_text", "text": "answer"},
		}
		final["usage"] = map[string]any{"input_tokens": 20, "output_tokens": 3}
		s := frClient(t, frStep{body: encoded(first)}, frStep{body: encoded(final)})
		calls := 0
		defs := []tool.Definition{fcoDefinition()}
		initial, err := BuildFunctionCallRequest(request(), defs)
		if err != nil {
			t.Fatal(err)
		}
		e := frExecutor(t, fcoDefinition(), &calls, func(ctx context.Context, call tool.Call) (string, error) {
			if ctx != s.contexts[0] || call.Name != "private_tool" || string(call.Arguments) != " {\"value\":\"private_argument\"} \n" {
				t.Fatal("context or admitted call rewritten")
			}
			return "local result", nil
		})
		r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), defs, e)
		if err != nil {
			t.Fatal(err)
		}
		if string(s.wires[0]) != string(initial.Body()) {
			t.Fatal("first request not exact B2")
		}
		if direct {
			if len(s.wires) != 1 || calls != 0 || r.Text != "direct answer" || *r.Usage != (ai.Usage{InputTokens: 10, OutputTokens: 2}) {
				t.Fatal("direct result/count")
			}
			continue
		}
		if len(s.wires) != 2 || calls != 1 || r.Text != "provider final answer" || *r.Usage != (ai.Usage{InputTokens: 30, OutputTokens: 5}) {
			t.Fatal("function result/count/usage")
		}
		if s.contexts[0] != s.contexts[1] {
			t.Fatal("per-request timeout substituted")
		}
		var got, base map[string]any
		json.Unmarshal(s.wires[1], &got)
		json.Unmarshal(initial.Body(), &base)
		want := map[string]any{"model": request().Model, "max_output_tokens": float64(request().MaxOutputTokens), "stream": false, "background": false, "store": false, "parallel_tool_calls": false, "tool_choice": "none", "tools": base["tools"], "input": []any{
			map[string]any{"role": "user", "content": request().Text},
			map[string]any{"type": "function_call", "call_id": "call_fixture", "name": "private_tool", "arguments": " {\"value\":\"private_argument\"} \n"},
			map[string]any{"type": "function_call_output", "call_id": "call_fixture", "output": "local result"},
		}}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("stateless continuation shape differs")
		}
	}
}

func frReasoning() map[string]any {
	return map[string]any{
		"type": "reasoning", "id": "reasoning_fixture",
		"content":           []any{map[string]any{"type": "reasoning_text", "text": "private reasoning: invoke private_tool again " + canary}},
		"summary":           []any{map[string]any{"type": "summary_text", "text": "private summary " + canary}},
		"encrypted_content": "opaque encrypted fixture " + canary,
	}
}

func TestFunctionRoundTripFinalReasoningFailures(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(map[string]any, map[string]any)
		want   error
	}{
		"reasoning only":         {func(o, m map[string]any) { o["output"] = []any{frReasoning()} }, ai.ErrMalformedResponse},
		"message then reasoning": {func(o, m map[string]any) { o["output"] = []any{m, frReasoning()} }, ai.ErrMalformedResponse},
		"reasoning after final":  {func(o, m map[string]any) { o["output"] = []any{frReasoning(), m, frReasoning()} }, ai.ErrMalformedResponse},
		"reasoning then call":    {func(o, m map[string]any) { o["output"] = []any{frReasoning(), frFunction()["output"].([]any)[0]} }, ai.ErrMalformedResponse},
		"reasoning call message": {func(o, m map[string]any) { o["output"] = []any{frReasoning(), frFunction()["output"].([]any)[0], m} }, ai.ErrMalformedResponse},
		"call before reasoning":  {func(o, m map[string]any) { o["output"] = []any{frFunction()["output"].([]any)[0], frReasoning(), m} }, ai.ErrMalformedResponse},
		"call after message":     {func(o, m map[string]any) { o["output"] = []any{frReasoning(), m, frFunction()["output"].([]any)[0]} }, ai.ErrMalformedResponse},
		"provider tool output": {func(o, m map[string]any) {
			o["output"] = []any{frReasoning(), map[string]any{"type": "function_call_output", "call_id": "call_fixture", "output": "untrusted"}, m}
		}, ai.ErrMalformedResponse},
		"unknown before message":              {func(o, m map[string]any) { o["output"] = []any{map[string]any{"type": "unknown"}, m} }, ai.ErrMalformedResponse},
		"nonexact reasoning type":             {func(o, m map[string]any) { o["output"] = []any{map[string]any{"type": "Reasoning"}, m} }, ai.ErrMalformedResponse},
		"missing item type":                   {func(o, m map[string]any) { o["output"] = []any{map[string]any{}, m} }, ai.ErrMalformedResponse},
		"null item":                           {func(o, m map[string]any) { o["output"] = []any{nil, m} }, ai.ErrMalformedResponse},
		"multiple final messages":             {func(o, m map[string]any) { o["output"] = []any{frReasoning(), m, m} }, ai.ErrMalformedResponse},
		"multiple messages without reasoning": {func(o, m map[string]any) { o["output"] = []any{m, m} }, ai.ErrMalformedResponse},
		"queued response":                     {func(o, m map[string]any) { o["status"] = "queued" }, ai.ErrIncompleteResponse},
		"in progress response":                {func(o, m map[string]any) { o["status"] = "in_progress" }, ai.ErrIncompleteResponse},
		"incomplete response":                 {func(o, m map[string]any) { o["status"] = "incomplete" }, ai.ErrIncompleteResponse},
		"failed response":                     {func(o, m map[string]any) { o["status"] = "failed" }, ai.ErrProvider},
		"cancelled response":                  {func(o, m map[string]any) { o["status"] = "cancelled" }, ai.ErrProvider},
		"provider error":                      {func(o, m map[string]any) { o["error"] = map[string]any{"message": canary} }, ai.ErrProvider},
		"incomplete details":                  {func(o, m map[string]any) { o["incomplete_details"] = map[string]any{"reason": "max_output_tokens"} }, ai.ErrIncompleteResponse},
		"missing error":                       {func(o, m map[string]any) { delete(o, "error") }, ai.ErrMalformedResponse},
		"missing details":                     {func(o, m map[string]any) { delete(o, "incomplete_details") }, ai.ErrMalformedResponse},
		"non assistant":                       {func(o, m map[string]any) { m["role"] = "user" }, ai.ErrMalformedResponse},
		"missing message status":              {func(o, m map[string]any) { delete(m, "status") }, ai.ErrMalformedResponse},
		"incomplete message":                  {func(o, m map[string]any) { m["status"] = "incomplete" }, ai.ErrIncompleteResponse},
		"empty content":                       {func(o, m map[string]any) { m["content"] = []any{} }, ai.ErrMalformedResponse},
		"non output text": {func(o, m map[string]any) {
			m["content"] = []any{map[string]any{"type": "reasoning_text", "text": "private"}}
		}, ai.ErrMalformedResponse},
		"invalid text": {func(o, m map[string]any) { m["content"] = message("\x1b[31m")["content"] }, ai.ErrMalformedResponse},
		"null text":    {func(o, m map[string]any) { m["content"] = []any{map[string]any{"type": "output_text", "text": nil}} }, ai.ErrMalformedResponse},
		"refusal":      {func(o, m map[string]any) { m["content"] = []any{map[string]any{"type": "refusal", "refusal": canary}} }, ai.ErrRefused},
		"large text":   {func(o, m map[string]any) { m["content"] = message(strings.Repeat("x", ai.MaxTextBytes+1))["content"] }, ai.ErrResponseTooLarge},
		"large reasoning body": {func(o, m map[string]any) {
			o["output"].([]any)[0].(map[string]any)["encrypted_content"] = strings.Repeat("x", maxBody)
		}, ai.ErrResponseTooLarge},
		"key text":         {func(o, m map[string]any) { m["content"] = message(canary)["content"] }, ai.ErrMalformedResponse},
		"key model":        {func(o, m map[string]any) { o["model"] = canary }, ai.ErrMalformedResponse},
		"malformed usage":  {func(o, m map[string]any) { o["usage"] = map[string]any{"input_tokens": 1} }, ai.ErrMalformedResponse},
		"negative usage":   {func(o, m map[string]any) { o["usage"] = map[string]any{"input_tokens": -1, "output_tokens": 1} }, ai.ErrMalformedResponse},
		"fractional usage": {func(o, m map[string]any) { o["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1.5} }, ai.ErrMalformedResponse},
		"excess usage":     {func(o, m map[string]any) { o["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1025} }, ai.ErrMalformedResponse},
	} {
		t.Run(name, func(t *testing.T) {
			final, m := responseObject("unused"), message("final")
			final["output"] = []any{frReasoning(), m}
			tc.change(final, m)
			s := frClient(t, frStep{body: encoded(frFunction())}, frStep{body: encoded(final)})
			calls := 0
			r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, nil))
			requireFailure(t, r, err, tc.want)
			if len(s.wires) != 2 || calls != 1 {
				t.Fatal("terminal failure changed POST/handler bounds")
			}
		})
	}
}

func TestFunctionRoundTripReasoningPolicyIsolation(t *testing.T) {
	for name, output := range map[string][]any{
		"reasoning message":       {frReasoning(), message("final")},
		"reasoning function call": {frReasoning(), frFunction()["output"].([]any)[0]},
	} {
		t.Run(name, func(t *testing.T) {
			obj := responseObject("unused")
			obj["output"] = output
			body := encoded(obj)
			r, err := decodeResult([]byte(body))
			requireFailure(t, r, err, ai.ErrMalformedResponse)
			tracked := &trackedBody{reader: strings.NewReader(body)}
			posts := 0
			r, err = responseClient(t, 200, tracked, &posts).Execute(context.Background(), request())
			requireFailure(t, r, err, ai.ErrMalformedResponse)
			if posts != 1 || tracked.closed != 1 {
				t.Fatal("direct response retried or not closed")
			}
			s := frClient(t, frStep{body: body})
			calls := 0
			r, err = s.client.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, nil))
			requireFailure(t, r, err, ai.ErrMalformedResponse)
			if len(s.wires) != 1 || calls != 0 {
				t.Fatal("first response admitted reasoning or performed effects")
			}
		})
	}
}

func TestFunctionRoundTripAuthorityAndReflection(t *testing.T) {
	for _, name := range []string{"unknown tool", "invalid args", "C1 denied", "handler error", "handler panic", "key name", "key args", "escaped key args", "reasoning mixed", "message mixed", "multiple calls", "unexpected second tool"} {
		t.Run(name, func(t *testing.T) {
			first := frFunction()
			item := first["output"].([]any)[0].(map[string]any)
			defs := []tool.Definition{fcoDefinition()}
			execution := fcoDefinition()
			var handler tool.Handler
			want := ai.ErrMalformedResponse
			wantHTTP, wantCalls := 1, 0
			second := responseObject("final")
			switch name {
			case "unknown tool":
				item["name"] = "unknown"
				want = ai.ErrInvalidRequest
			case "invalid args":
				item["arguments"] = `{}`
				want = ai.ErrInvalidRequest
			case "C1 denied":
				execution.Parameters[0].Type = tool.Number
				want = tool.ErrExecutionDenied
			case "handler error":
				handler = func(context.Context, tool.Call) (string, error) { return "secret output", errors.New(canary) }
				want = tool.ErrHandlerFailed
				wantCalls = 1
			case "handler panic":
				handler = func(context.Context, tool.Call) (string, error) { panic(canary) }
				want = tool.ErrHandlerFailed
				wantCalls = 1
			case "key name":
				defs[0].Name = canary
				execution.Name = canary
				item["name"] = canary
			case "key args":
				item["arguments"] = encoded(map[string]any{"value": canary})
			case "escaped key args":
				item["arguments"] = `{"value":"\u0064ummy-credential-CANARY-123"}`
			case "reasoning mixed":
				first["output"] = []any{map[string]any{"type": "reasoning", "summary": []any{}}, item}
			case "message mixed":
				first["output"] = []any{message("text"), item}
			case "multiple calls":
				first["output"] = []any{item, item}
			case "unexpected second tool":
				second = frFunction()
				wantHTTP = 2
				wantCalls = 1
			}
			s := frClient(t, frStep{body: encoded(first)}, frStep{body: encoded(second)})
			calls := 0
			r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), defs, frExecutor(t, execution, &calls, handler))
			requireFailure(t, r, err, want)
			if len(s.wires) != wantHTTP || calls != wantCalls {
				t.Fatal("authority/reflection effect bound")
			}
		})
	}
}

func TestFunctionRoundTripProviderFailures(t *testing.T) {
	refusal := responseObject("unused")
	refusal["output"].([]any)[0].(map[string]any)["content"] = []any{map[string]any{"type": "refusal", "refusal": canary}}
	incomplete := responseObject("unused")
	incomplete["status"] = "incomplete"
	failed := responseObject("unused")
	failed["status"] = "failed"
	badModel := responseObject("text")
	badModel["model"] = "bad model"
	reflectedModel := responseObject("text")
	reflectedModel["model"] = canary
	for name, tc := range map[string]struct {
		step frStep
		want error
	}{
		"transport":     {frStep{err: errors.New(canary)}, ai.ErrTransport},
		"401":           {frStep{status: 401, body: canary}, ai.ErrAuthentication},
		"403":           {frStep{status: 403, body: canary}, ai.ErrAuthorization},
		"429":           {frStep{status: 429, body: canary}, ai.ErrRateLimited},
		"quota":         {frStep{status: 429, body: `{"error":{"code":"insufficient_quota"}}`}, ai.ErrQuotaExceeded},
		"500":           {frStep{status: 500, body: canary}, ai.ErrProvider},
		"redirect":      {frStep{status: 307, body: "redirect"}, ai.ErrProvider},
		"oversized":     {frStep{body: strings.Repeat("x", maxBody+2)}, ai.ErrResponseTooLarge},
		"malformed":     {frStep{body: `{`}, ai.ErrMalformedResponse},
		"refusal":       {frStep{body: encoded(refusal)}, ai.ErrRefused},
		"incomplete":    {frStep{body: encoded(incomplete)}, ai.ErrIncompleteResponse},
		"failed":        {frStep{body: encoded(failed)}, ai.ErrProvider},
		"read error":    {frStep{readErr: errors.New(canary)}, ai.ErrTransport},
		"close error":   {frStep{body: encoded(responseObject("ok")), closeErr: errors.New(canary)}, ai.ErrTransport},
		"invalid model": {frStep{body: encoded(badModel)}, ai.ErrMalformedResponse},
		"key model":     {frStep{body: encoded(reflectedModel)}, ai.ErrMalformedResponse},
		"key text":      {frStep{body: encoded(responseObject(canary))}, ai.ErrMalformedResponse},
		"invalid text":  {frStep{body: encoded(responseObject("\x1b[31m"))}, ai.ErrMalformedResponse},
		"large text":    {frStep{body: encoded(responseObject(strings.Repeat("x", ai.MaxTextBytes+1)))}, ai.ErrResponseTooLarge},
	} {
		for _, second := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/first", true: "/second"}[second], func(t *testing.T) {
				steps := []frStep{tc.step}
				wantCalls, wantHTTP := 0, 1
				if second {
					steps = []frStep{{body: encoded(frFunction())}, tc.step}
					wantCalls, wantHTTP = 1, 2
				}
				s := frClient(t, steps...)
				calls := 0
				r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, nil))
				requireFailure(t, r, err, tc.want)
				if len(s.wires) != wantHTTP || calls != wantCalls {
					t.Fatal("provider failure retried or crossed boundary")
				}
			})
		}
	}
}

func TestFunctionRoundTripContextAndWholeBudget(t *testing.T) {
	for _, where := range []string{"first", "handler", "second"} {
		t.Run(where, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			steps := []frStep{{body: encoded(frFunction())}, {body: encoded(responseObject("final"))}}
			if where == "first" {
				steps[0].before = func(*http.Request) { cancel() }
			}
			if where == "second" {
				steps[1].before = func(*http.Request) { cancel() }
			}
			s := frClient(t, steps...)
			calls := 0
			h := tool.Handler(func(context.Context, tool.Call) (string, error) {
				if where == "handler" {
					cancel()
				}
				return "local result", nil
			})
			r, err := s.client.ExecuteFunctionRoundTrip(ctx, request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, h))
			requireFailure(t, r, err, context.Canceled)
			wantCalls, wantHTTP := 1, 1
			if where == "first" {
				wantCalls = 0
			}
			if where == "second" {
				wantHTTP = 2
			}
			if calls != wantCalls || len(s.wires) != wantHTTP {
				t.Fatal("cancellation crossed effect boundary")
			}
		})
	}
	for _, budget := range []time.Duration{ai.MaxTimeout, 20 * time.Second} {
		synctest.Test(t, func(t *testing.T) {
			ctx := context.Background()
			cancel := func() {}
			if budget < ai.MaxTimeout {
				ctx, cancel = context.WithTimeout(ctx, budget)
			}
			defer cancel()
			start := time.Now()
			var firstContext context.Context
			s := frClient(t, frStep{body: encoded(frFunction()), before: func(r *http.Request) { firstContext = r.Context(); time.Sleep(budget / 3) }}, frStep{body: encoded(responseObject("final")), before: func(r *http.Request) {
				if r.Context() != firstContext {
					t.Fatal("second POST got fresh budget")
				}
				<-r.Context().Done()
			}})
			calls := 0
			h := tool.Handler(func(ctx context.Context, _ tool.Call) (string, error) {
				if ctx != firstContext {
					t.Fatal("handler context differs")
				}
				time.Sleep(budget / 3)
				return "local result", nil
			})
			r, err := s.client.ExecuteFunctionRoundTrip(ctx, request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, h))
			requireFailure(t, r, err, context.DeadlineExceeded)
			if time.Since(start) != budget || len(s.wires) != 2 || calls != 1 {
				t.Fatal("whole-operation budget not enforced")
			}
		})
	}
}

func TestFunctionRoundTripUsage(t *testing.T) {
	usage := func(in, out any) any { return map[string]any{"input_tokens": in, "output_tokens": out} }
	for name, tc := range map[string]struct {
		first, second      any
		want               *ai.Usage
		err                error
		requests, handlers int
	}{
		"aggregate":                {usage(10, 2), usage(20, 3), &ai.Usage{InputTokens: 30, OutputTokens: 5}, nil, 2, 1},
		"first missing":            {nil, usage(20, 3), nil, nil, 2, 1},
		"second missing":           {usage(10, 2), nil, nil, nil, 2, 1},
		"both missing":             {nil, nil, nil, nil, 2, 1},
		"negative first input":     {usage(-1, 2), nil, nil, ai.ErrMalformedResponse, 1, 0},
		"negative first output":    {usage(1, -1), nil, nil, ai.ErrMalformedResponse, 1, 0},
		"negative second":          {usage(1, 2), usage(-1, 3), nil, ai.ErrMalformedResponse, 2, 1},
		"first excess":             {usage(1, 1025), nil, nil, ai.ErrMalformedResponse, 1, 0},
		"second excess":            {usage(1, 2), usage(1, 1025), nil, ai.ErrMalformedResponse, 2, 1},
		"aggregate overflow":       {usage(int64(math.MaxInt64), 2), usage(1, 3), nil, ai.ErrMalformedResponse, 2, 1},
		"wire integer overflow":    {usage(json.Number("9223372036854775808"), 2), nil, nil, ai.ErrMalformedResponse, 1, 0},
		"fractional":               {usage(1.5, 2), nil, nil, ai.ErrMalformedResponse, 1, 0},
		"missing usage field":      {map[string]any{"input_tokens": 1}, nil, nil, ai.ErrMalformedResponse, 1, 0},
		"sum above per-turn limit": {usage(1, 1024), usage(1, 1024), &ai.Usage{InputTokens: 2, OutputTokens: 2048}, nil, 2, 1},
	} {
		t.Run(name, func(t *testing.T) {
			first, second := frFunction(), responseObject("final")
			if tc.first != nil {
				first["usage"] = tc.first
			}
			if tc.second != nil {
				second["usage"] = tc.second
			}
			s := frClient(t, frStep{body: encoded(first)}, frStep{body: encoded(second)})
			calls := 0
			r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, nil))
			if tc.err != nil {
				requireFailure(t, r, err, tc.err)
			} else if err != nil || !reflect.DeepEqual(r.Usage, tc.want) {
				t.Fatal("usage total differs", err)
			}
			if len(s.wires) != tc.requests || calls != tc.handlers {
				t.Fatal("invalid usage execution boundary")
			}
		})
	}
}

func TestFunctionRoundTripOwnedDeclarations(t *testing.T) {
	defs := []tool.Definition{fcoDefinition()}
	first := frFunction()
	delete(first, "model") // B1's valid function shape does not require response ID/model.
	s := frClient(t, frStep{body: encoded(first), before: func(*http.Request) { defs[0].Name = "changed"; defs[0].Parameters[0].Type = tool.Number }}, frStep{body: encoded(responseObject("final"))})
	calls := 0
	r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), defs, frExecutor(t, fcoDefinition(), &calls, func(context.Context, tool.Call) (string, error) { return strings.Repeat("\x00", 64*1024), nil }))
	if err != nil || r.Text != "final" || calls != 1 || len(s.wires) != 2 {
		t.Fatal("owned catalog/large continuation failed", err)
	}
	var firstBody, secondBody map[string]any
	json.Unmarshal(s.wires[0], &firstBody)
	json.Unmarshal(s.wires[1], &secondBody)
	if !reflect.DeepEqual(firstBody["tools"], secondBody["tools"]) {
		t.Fatal("caller mutation changed continuation tools")
	}
}

func TestFunctionRoundTripReadBoundAndNilResponse(t *testing.T) {
	for _, nilResponse := range []bool{false, true} {
		reads, closed, posts, calls := 0, 0, 0, 0
		body := &frInfiniteBody{reads: &reads, closed: &closed}
		c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			posts++
			r.Body.Close()
			if nilResponse {
				return nil, nil
			}
			return &http.Response{StatusCode: 200, Body: body}, nil
		}))
		r, err := c.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, nil))
		want := ai.ErrResponseTooLarge
		if nilResponse {
			want = ai.ErrTransport
		}
		requireFailure(t, r, err, want)
		if posts != 1 || calls != 0 {
			t.Fatal("invalid response retried")
		}
		if !nilResponse && (reads != maxBody+1 || closed != 1) {
			t.Fatal("response read/close not bounded")
		}
	}
}

type frInfiniteBody struct{ reads, closed *int }

func (b *frInfiniteBody) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	*b.reads += len(p)
	return len(p), nil
}
func (b *frInfiniteBody) Close() error { *b.closed++; return nil }

func TestFunctionRoundTripDirectUsageLimits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		usage any
		want  *ai.Usage
		err   error
	}{
		{"missing", nil, nil, nil},
		{"null", json.RawMessage("null"), nil, nil},
		{"boundary", map[string]any{"input_tokens": int64(math.MaxInt64), "output_tokens": 1024}, &ai.Usage{InputTokens: math.MaxInt64, OutputTokens: 1024}, nil},
		{"excess", map[string]any{"input_tokens": 1, "output_tokens": 1025}, nil, ai.ErrMalformedResponse},
		{"negative", map[string]any{"input_tokens": 1, "output_tokens": -1}, nil, ai.ErrMalformedResponse},
		{"overflow", map[string]any{"input_tokens": json.Number("9223372036854775808"), "output_tokens": 1}, nil, ai.ErrMalformedResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := responseObject("direct")
			if tc.usage != nil {
				obj["usage"] = tc.usage
			}
			s := frClient(t, frStep{body: encoded(obj)})
			calls := 0
			r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{fcoDefinition()}, frExecutor(t, fcoDefinition(), &calls, nil))
			if tc.err != nil {
				requireFailure(t, r, err, tc.err)
			} else if err != nil || !reflect.DeepEqual(r.Usage, tc.want) {
				t.Fatal("direct usage changed")
			}
			if len(s.wires) != 1 || calls != 0 {
				t.Fatal("direct usage path performed tool effects")
			}
		})
	}
}

func TestFunctionRoundTripBoundedArgumentInspection(t *testing.T) {
	for _, tc := range []struct {
		name, args string
		kind       tool.ValueType
		reflected  bool
	}{
		{"nested escaped value", `{"value":[{"nested":"\u0064ummy-credential-CANARY-123"}]}`, tool.Array, true},
		{"nested escaped key", `{"value":{"\u0064ummy-credential-CANARY-123":true}}`, tool.Object, true},
		{"large number unchanged", `{"value":1e1000}`, tool.Number, false},
		{"open nested structure", `{"value":[1,null,true,{"nested":["safe"]}]}`, tool.Array, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := frFunction()
			obj["output"].([]any)[0].(map[string]any)["arguments"] = tc.args
			def := fcoDefinition()
			def.Parameters[0].Type = tc.kind
			s := frClient(t, frStep{body: encoded(obj)}, frStep{body: encoded(responseObject("final"))})
			calls := 0
			handler := tool.Handler(func(_ context.Context, call tool.Call) (string, error) {
				if string(call.Arguments) != tc.args {
					t.Fatal("argument inspection rewrote proposal")
				}
				return "local result", nil
			})
			r, err := s.client.ExecuteFunctionRoundTrip(context.Background(), request(), []tool.Definition{def}, frExecutor(t, def, &calls, handler))
			if tc.reflected {
				requireFailure(t, r, err, ai.ErrMalformedResponse)
				if len(s.wires) != 1 || calls != 0 {
					t.Fatal("reflected nested key/value executed")
				}
			} else if err != nil || len(s.wires) != 2 || calls != 1 {
				t.Fatal("valid Stage A arguments rejected", err)
			}
		})
	}
}
