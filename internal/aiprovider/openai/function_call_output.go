package openai

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// FunctionCallOutput owns one serialized Responses input item, not a request
// or permission to send one. Its zero value exposes no body.
type FunctionCallOutput struct {
	body []byte
}

// Body returns an independent copy of the serialized item, or nil for zero.
func (o FunctionCallOutput) Body() []byte { return bytes.Clone(o.body) }

// String and GoString expose neither correlation IDs nor execution output.
func (o FunctionCallOutput) String() string {
	return "OpenAI function-call output (payload redacted)"
}

func (o FunctionCallOutput) GoString() string { return o.String() }

type functionCallOutputItem struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// BuildFunctionCallOutput is pure, offline serialization of B1 correlation and
// an already-produced C1 result. B1 correlation != execution authority;
// C1 success != provider continuation authority; C2 serialization != permission
// to send a request. It neither executes handlers nor sends provider requests.
//
// The caller must pair the decision with the corresponding successful C1
// Execute result (error == nil). ExecutionResult provides type coupling, not
// unforgeable provenance, proof of success, or correlation to this decision.
// Its zero value is indistinguishable from successful empty output. This builder
// cannot enforce that pairing and does not serialize execution failures.
func BuildFunctionCallOutput(decision FunctionCallDecision, result tool.ExecutionResult) (FunctionCallOutput, error) {
	_, callID, ok := decision.Correlation()
	if !ok || !validFunctionCallID(callID) {
		return FunctionCallOutput{}, ai.ErrInvalidRequest
	}
	output := result.Output()
	// Defend C1's public 64 KiB UTF-8 contract before encoding; do not repair
	// invalid strings through encoding/json's replacement-character behavior.
	if len(output) > 64*1024 || !utf8.ValidString(output) {
		return FunctionCallOutput{}, ai.ErrInvalidRequest
	}
	body, err := json.Marshal(functionCallOutputItem{
		Type: "function_call_output", CallID: callID, Output: output,
	})
	if err != nil || len(body) > maxBody {
		return FunctionCallOutput{}, ai.ErrInvalidRequest
	}
	return FunctionCallOutput{body: body}, nil
}
