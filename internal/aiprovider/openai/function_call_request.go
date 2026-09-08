package openai

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// FunctionCallRequest owns a serialized proposal schema and its corresponding
// Stage A authority. Its zero value exposes neither a body nor a catalog.
// Provider schemas do not grant authority; every proposal still needs
// MapFunctionCallResponse and Catalog.Admit. ADMITTED does not mean EXECUTED.
type FunctionCallRequest struct {
	body    []byte
	catalog *tool.Catalog
}

// Body returns an independent copy of the complete JSON request, or nil for
// the zero value. Reading it performs no provider request or other I/O.
func (r FunctionCallRequest) Body() []byte { return bytes.Clone(r.body) }

// Catalog returns the immutable Stage A catalog built from the same owned
// definitions as Body. The zero value returns nil, not an admission authority.
func (r FunctionCallRequest) Catalog() *tool.Catalog { return r.catalog }

// String and GoString omit prompts, models, tool names, and schemas.
func (r FunctionCallRequest) String() string {
	return "OpenAI function-tool request (payload redacted)"
}

func (r FunctionCallRequest) GoString() string { return r.String() }

type functionCallPayload struct {
	payload
	Tools             []functionRequestTool `json:"tools"`
	ToolChoice        string                `json:"tool_choice"`
	ParallelToolCalls bool                  `json:"parallel_tool_calls"`
}

type functionRequestTool struct {
	Type       string                `json:"type"`
	Name       string                `json:"name"`
	Parameters functionRequestSchema `json:"parameters"`
	Strict     bool                  `json:"strict"`
}

type functionRequestSchema struct {
	Type                 string                              `json:"type"`
	Properties           map[string]functionRequestValueType `json:"properties"`
	Required             []string                            `json:"required"`
	AdditionalProperties bool                                `json:"additionalProperties"`
}

type functionRequestValueType struct {
	Type string `json:"type"`
}

// BuildFunctionCallRequest is a pure, offline builder; it does not construct a
// Client or send a request. Client.Execute remains the separate text-only path.
// Nonempty caller-authorized definitions are required. Callers must not mutate
// definitions or parameter slices while this function reads/copies them.
// Mutations after return cannot change the serialized body or catalog.
//
// Tools, required names, and property keys have canonical name ordering.
// strict:false preserves optional root parameters and structurally open nested
// objects/arrays. parallel_tool_calls:false requests at most one proposal for
// B1; only Stage A admission establishes whether a supplied proposal is allowed.
func BuildFunctionCallRequest(request ai.Request, definitions []tool.Definition) (FunctionCallRequest, error) {
	if err := request.Validate(); err != nil {
		return FunctionCallRequest{}, err
	}
	if len(definitions) == 0 {
		return FunctionCallRequest{}, ai.ErrInvalidRequest
	}
	// Let Stage A bound and validate caller input before allocating our snapshot.
	// Do not duplicate its private limits or name/type validation rules. This
	// preflight catalog is discarded; the published catalog uses only snapshot.
	if _, err := tool.NewCatalog(definitions); err != nil {
		return FunctionCallRequest{}, err
	}
	snapshot := make([]tool.Definition, len(definitions))
	for i, definition := range definitions {
		parameters := make([]tool.Parameter, len(definition.Parameters))
		for j, parameter := range definition.Parameters {
			parameters[j] = tool.Parameter{
				Name: strings.Clone(parameter.Name), Type: parameter.Type, Required: parameter.Required,
			}
		}
		snapshot[i] = tool.Definition{Name: strings.Clone(definition.Name), Parameters: parameters}
	}
	catalog, err := tool.NewCatalog(snapshot)
	if err != nil {
		return FunctionCallRequest{}, err
	}

	names := make([]string, 0, len(snapshot))
	declarations := make(map[string]functionRequestTool, len(snapshot))
	for _, definition := range snapshot {
		schema := functionRequestSchema{
			Type:       "object",
			Properties: make(map[string]functionRequestValueType, len(definition.Parameters)),
			Required:   make([]string, 0, len(definition.Parameters)),
		}
		for _, parameter := range definition.Parameters {
			kind, err := functionRequestType(parameter.Type)
			if err != nil {
				return FunctionCallRequest{}, err
			}
			// Nested containers intentionally have only a type, not field/item schemas.
			schema.Properties[parameter.Name] = functionRequestValueType{Type: kind}
			if parameter.Required {
				schema.Required = append(schema.Required, parameter.Name)
			}
		}
		sort.Strings(schema.Required)
		names = append(names, definition.Name)
		declarations[definition.Name] = functionRequestTool{
			Type: "function", Name: definition.Name, Parameters: schema, Strict: false,
		}
	}
	sort.Strings(names)
	tools := make([]functionRequestTool, 0, len(names))
	for _, name := range names {
		tools = append(tools, declarations[name])
	}
	// encoding/json sorts string map keys, including schema.Properties.
	body, err := json.Marshal(functionCallPayload{
		payload: payload{Model: request.Model, Input: request.Text, MaxOutputTokens: request.MaxOutputTokens},
		Tools:   tools, ToolChoice: "auto", ParallelToolCalls: false,
	})
	if err != nil || len(body) > maxBody {
		return FunctionCallRequest{}, ai.ErrInvalidRequest
	}
	return FunctionCallRequest{body: body, catalog: catalog}, nil
}

func functionRequestType(kind tool.ValueType) (string, error) {
	switch kind {
	case tool.String:
		return "string", nil
	case tool.Number:
		return "number", nil
	case tool.Boolean:
		return "boolean", nil
	case tool.Object:
		return "object", nil
	case tool.Array:
		return "array", nil
	case tool.Null:
		return "null", nil
	default:
		return "", tool.ErrInvalidCatalog
	}
}
