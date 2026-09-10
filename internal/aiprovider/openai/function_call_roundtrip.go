package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// ExecuteFunctionRoundTrip permits a direct text answer or one function round:
// at most two provider POSTs and one handler attempt, under one caller-bounded
// timeout. Client.Execute remains the separate text-only API.
//
// B2 declares tools; B1 admits exactly one completed direct function proposal;
// C3 owns replay-safe pairing through C1 execution authority and C2 output.
// Continuation replays owned user/call/output data with store:false, never remote
// conversation state. Reasoning plus function calls and other mixed output are
// outside B1's contract. The second response must be final text, not another call.
// Only that terminal response may contain leading opaque reasoning metadata.
// Replay protection lasts only for this invocation's coordinator, not across
// independent invocations or process restarts. Handlers must honor context.
func (c *Client) ExecuteFunctionRoundTrip(ctx context.Context, request ai.Request, definitions []tool.Definition, executor *tool.Executor) (ai.Result, error) {
	if c == nil || c.http == nil || c.key == "" || ctx == nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	ctx, cancel := context.WithTimeout(ctx, ai.MaxTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return ai.Result{}, err
	}
	functionRequest, err := BuildFunctionCallRequest(request, definitions)
	if err != nil {
		return ai.Result{}, err
	}
	coordinator, err := NewFunctionCallCoordinator(executor)
	if err != nil {
		return ai.Result{}, err
	}
	initialBody := functionRequest.Body()
	var initial functionCallPayload
	// Only Forge-owned B2 JSON is decoded here, to reuse its exact declaration.
	if json.Unmarshal(initialBody, &initial) != nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	first, err := c.functionRoundTripPost(ctx, initialBody)
	if err != nil {
		return ai.Result{}, err
	}
	result, textErr := decodeResult(first)
	if textErr == nil {
		return c.functionRoundTripResult(ctx, result, request.MaxOutputTokens)
	}
	if textErr != ai.ErrMalformedResponse || !functionRoundTripCandidate(first) {
		return ai.Result{}, textErr
	}
	decision, err := MapFunctionCallResponse(first, functionRequest.Catalog())
	if err != nil {
		return ai.Result{}, err
	}
	call, admitted := decision.Admission().AdmittedCall()
	if !admitted {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	if functionRoundTripReflectsKey(call, c.key) {
		return ai.Result{}, ai.ErrMalformedResponse
	}
	firstUsage, err := functionRoundTripUsage(first, request.MaxOutputTokens)
	if err != nil {
		return ai.Result{}, err
	}
	output, err := coordinator.Execute(ctx, decision)
	if err != nil {
		return ai.Result{}, err
	}
	_, callID, _ := decision.Correlation()
	continuation, err := json.Marshal(functionRoundTripPayload{
		Model: initial.Model, MaxOutputTokens: initial.MaxOutputTokens,
		Tools: initial.Tools, ToolChoice: "none",
		Input: [3]any{
			functionRoundTripUser{Role: "user", Content: initial.Input},
			functionRoundTripCall{Type: "function_call", CallID: callID, Name: call.Name, Arguments: string(call.Arguments)},
			json.RawMessage(output.Body()),
		},
	})
	if err != nil || len(continuation) > maxBody {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	second, err := c.functionRoundTripPost(ctx, continuation)
	if err != nil {
		return ai.Result{}, err
	}
	result, err = decodeFunctionRoundTripFinalResult(second)
	if err != nil {
		return ai.Result{}, err
	}
	result, err = c.functionRoundTripResult(ctx, result, request.MaxOutputTokens)
	if err != nil {
		return ai.Result{}, err
	}
	result.Usage, err = functionRoundTripTotalUsage(firstUsage, result.Usage)
	if err != nil {
		return ai.Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return ai.Result{}, err
	}
	return result, nil
}

type functionRoundTripPayload struct {
	Model           string                `json:"model"`
	Input           [3]any                `json:"input"`
	MaxOutputTokens int                   `json:"max_output_tokens"`
	Stream          bool                  `json:"stream"`
	Background      bool                  `json:"background"`
	Store           bool                  `json:"store"`
	Tools           []functionRequestTool `json:"tools"`
	ToolChoice      string                `json:"tool_choice"`
	ParallelCalls   bool                  `json:"parallel_tool_calls"`
}

type functionRoundTripUser struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type functionRoundTripCall struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// This is only a dispatch hint after text decoding fails. B1 still performs all
// strict validation and rejects mixed/multiple outputs before any execution.
func functionRoundTripCandidate(body []byte) bool {
	var hint struct {
		Output []struct {
			Type string `json:"type"`
		} `json:"output"`
	}
	if json.Unmarshal(body, &hint) != nil {
		return false
	}
	for _, item := range hint.Output {
		if item.Type == "function_call" {
			return true
		}
	}
	return false
}

func functionRoundTripReflectsKey(call tool.Call, key string) bool {
	if strings.Contains(call.Name, key) || bytes.Contains(call.Arguments, []byte(key)) {
		return true
	}
	// Stage A already bounded/validated these arguments. Inspect decoded strings
	// and member names too, so JSON escapes cannot hide a reflected credential.
	var value any
	decoder := json.NewDecoder(bytes.NewReader(call.Arguments))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return true
	}
	return functionRoundTripContainsKey(value, key)
}

func functionRoundTripContainsKey(value any, key string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(v, key)
	case []any:
		for _, item := range v {
			if functionRoundTripContainsKey(item, key) {
				return true
			}
		}
	case map[string]any:
		for name, item := range v {
			if strings.Contains(name, key) || functionRoundTripContainsKey(item, key) {
				return true
			}
		}
	}
	return false
}

func functionRoundTripUsage(body []byte, limit int) (*ai.Usage, error) {
	obj, err := jsonObject(body)
	if err != nil {
		return nil, err
	}
	raw, exists := obj["usage"]
	if !exists || isNull(raw) {
		return nil, nil
	}
	usage, err := jsonObject(raw)
	if err != nil {
		return nil, err
	}
	var in, out *int64
	if json.Unmarshal(usage["input_tokens"], &in) != nil || in == nil || json.Unmarshal(usage["output_tokens"], &out) != nil || out == nil || *in < 0 || *out < 0 || *out > int64(limit) {
		return nil, ai.ErrMalformedResponse
	}
	return &ai.Usage{InputTokens: *in, OutputTokens: *out}, nil
}

func functionRoundTripTotalUsage(first, second *ai.Usage) (*ai.Usage, error) {
	if first == nil || second == nil {
		return nil, nil
	}
	if first.InputTokens > math.MaxInt64-second.InputTokens || first.OutputTokens > math.MaxInt64-second.OutputTokens {
		return nil, ai.ErrMalformedResponse
	}
	return &ai.Usage{InputTokens: first.InputTokens + second.InputTokens, OutputTokens: first.OutputTokens + second.OutputTokens}, nil
}

func (c *Client) functionRoundTripResult(ctx context.Context, result ai.Result, limit int) (ai.Result, error) {
	if err := result.Validate(); err != nil {
		return ai.Result{}, err
	}
	if result.Usage != nil && result.Usage.OutputTokens > int64(limit) || strings.Contains(result.Text, c.key) || strings.Contains(result.Model, c.key) {
		return ai.Result{}, ai.ErrMalformedResponse
	}
	if err := ctx.Err(); err != nil {
		return ai.Result{}, err
	}
	return result, nil
}

// One non-replayable POST with the existing owned client's redirect policy.
func (c *Client) functionRoundTripPost(ctx context.Context, data []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(data) > maxBody {
		return nil, ai.ErrInvalidRequest
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return nil, ai.ErrInvalidRequest
	}
	req.ContentLength = int64(len(data))
	req.GetBody = nil
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header["User-Agent"] = []string{""}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, transportError(ctx, err)
	}
	if resp == nil || resp.Body == nil {
		return nil, ai.ErrTransport
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	closeErr := resp.Body.Close()
	if readErr != nil || closeErr != nil {
		var safe []error
		if readErr != nil {
			safe = append(safe, transportError(ctx, readErr))
		}
		if closeErr != nil {
			safe = append(safe, ai.ErrTransport)
		}
		return nil, ai.SafeError(errors.Join(safe...))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, ai.ErrResponseTooLarge
	}
	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp.StatusCode, body)
	}
	return body, nil
}
