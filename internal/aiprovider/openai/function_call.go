package openai

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// FunctionCallDecision contains admission only, never execution, safety, success,
// or persistent authorization. Its zero value is Rejected / NotEvaluated with
// no correlation. Rejected proposals retain neither IDs nor proposal payload.
type FunctionCallDecision struct {
	itemID    string
	callID    string
	admission tool.Admission
}

// Admission returns the immutable Stage A decision. AdmittedCall on that value
// returns its own argument copy; the proposal is never an execution permit.
func (d FunctionCallDecision) Admission() tool.Admission { return d.admission }

// Correlation exposes opaque provider IDs only for admitted proposals. IDs are
// not authority, execution targets, paths, uniqueness, or replay protection.
func (d FunctionCallDecision) Correlation() (itemID string, callID string, ok bool) {
	if d.admission.Status() != tool.Admitted || d.admission.Reason() != tool.Allowed {
		return "", "", false
	}
	return d.itemID, d.callID, true
}

// String emits fixed classifications only, including for admitted proposals.
func (d FunctionCallDecision) String() string {
	return "OpenAI function-call mapping: " + d.admission.String()
}

// GoString also omits IDs, names, arguments, and provider metadata.
func (d FunctionCallDecision) GoString() string { return d.String() }

// MapFunctionCallResponse maps one completed, direct, non-namespaced function
// call from an already supplied Responses body through the caller's catalog.
// It performs no I/O, client construction, callbacks, or background work.
// The body is bounded to maxBody (1 MiB including whitespace) and copied before
// parsing. Callers must not mutate data while this call reads/copies it.
// Independent calls may share an immutable catalog concurrently.
//
// Outer JSON must pass the limits documented in function_call_json.go before
// any selected objects or logical strings are decoded. Arguments are decoded
// exactly once from the outer string and passed unchanged to Stage A: no
// argument parsing, rewriting, repair, or independent schema validation here.
// Every error returns a zero decision. A valid mapping may return a rejected
// admission with nil error; nil error does not imply admission or execution.
// Client.Execute remains a separate, unchanged text-only path.
func MapFunctionCallResponse(data []byte, catalog *tool.Catalog) (FunctionCallDecision, error) {
	if catalog == nil {
		return FunctionCallDecision{}, tool.ErrInvalidCatalog
	}
	if len(data) > maxBody {
		return FunctionCallDecision{}, ai.ErrResponseTooLarge
	}
	snapshot := bytes.Clone(data)
	if err := validateFunctionCallJSON(snapshot); err != nil {
		return FunctionCallDecision{}, err
	}
	// Existing helpers are used only after complete strict outer prevalidation.
	// RawMessage maps keep field lookup exact and case-sensitive.
	response, err := jsonObject(snapshot)
	if err != nil {
		return FunctionCallDecision{}, err
	}
	status, err := jsonString(response["status"])
	if err != nil {
		return FunctionCallDecision{}, err
	}
	switch status {
	case "incomplete", "queued", "in_progress":
		return FunctionCallDecision{}, ai.ErrIncompleteResponse
	case "failed", "cancelled":
		return FunctionCallDecision{}, ai.ErrProvider
	case "completed":
	default:
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	for _, field := range []string{"error", "incomplete_details"} {
		raw, present := response[field]
		if !present {
			return FunctionCallDecision{}, ai.ErrMalformedResponse
		}
		if isNull(raw) {
			continue
		}
		details, err := jsonObject(raw)
		if err != nil {
			return FunctionCallDecision{}, err
		}
		if field == "error" {
			return FunctionCallDecision{}, ai.ErrProvider
		}
		if _, err := jsonString(details["reason"]); err != nil {
			return FunctionCallDecision{}, err
		}
		return FunctionCallDecision{}, ai.ErrIncompleteResponse
	}
	var output []json.RawMessage
	if json.Unmarshal(response["output"], &output) != nil || len(output) != 1 {
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	item, err := jsonObject(output[0])
	if err != nil {
		return FunctionCallDecision{}, err
	}
	kind, err := jsonString(item["type"])
	if err != nil || kind != "function_call" {
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	state, err := jsonString(item["status"])
	if err != nil {
		return FunctionCallDecision{}, err
	}
	switch state {
	case "in_progress", "incomplete":
		return FunctionCallDecision{}, ai.ErrIncompleteResponse
	case "completed":
	default:
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	// Unknown fields cannot quietly add new provider capabilities. Namespace
	// is intentionally not allowed, even with a null or empty value.
	for field := range item {
		switch field {
		case "type", "status", "id", "call_id", "name", "arguments", "async", "caller":
		default:
			return FunctionCallDecision{}, ai.ErrMalformedResponse
		}
	}
	if raw, present := item["async"]; present && !bytes.Equal(bytes.TrimSpace(raw), []byte("false")) {
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	if raw, present := item["caller"]; present {
		caller, err := jsonObject(raw)
		if err != nil || len(caller) != 1 {
			return FunctionCallDecision{}, ai.ErrMalformedResponse
		}
		kind, err := jsonString(caller["type"])
		if err != nil || kind != "direct" {
			return FunctionCallDecision{}, ai.ErrMalformedResponse
		}
	}
	itemID, err := jsonString(item["id"])
	if err != nil || !validFunctionCallID(itemID) {
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	callID, err := jsonString(item["call_id"])
	if err != nil || !validFunctionCallID(callID) {
		return FunctionCallDecision{}, ai.ErrMalformedResponse
	}
	name, err := jsonString(item["name"])
	if err != nil {
		return FunctionCallDecision{}, err
	}
	arguments, err := jsonString(item["arguments"])
	if err != nil {
		return FunctionCallDecision{}, err
	}
	admission, err := catalog.Admit(tool.Call{Name: name, Arguments: []byte(arguments)})
	if err != nil {
		return FunctionCallDecision{}, err
	}
	decision := FunctionCallDecision{admission: admission}
	if admission.Status() == tool.Admitted && admission.Reason() == tool.Allowed {
		decision.itemID = strings.Clone(itemID)
		decision.callID = strings.Clone(callID)
	}
	return decision, nil
}

func validFunctionCallID(id string) bool {
	if len(id) == 0 || len(id) > 256 || !utf8.ValidString(id) {
		return false
	}
	for _, r := range id {
		if unicode.IsSpace(r) || r <= 0x1f || r >= 0x7f && r <= 0x9f {
			return false
		}
	}
	return true
}
