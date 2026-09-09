package openai

import (
	"context"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// ExecuteAuthorizedFunctionRoundTrip obtains declarations and execution
// configuration from the same caller-owned Authority. Empty authorities fail
// before HTTP. All round-trip behavior is delegated to the existing C4 API:
// Stage A admission is not execution, C1 supplies execution authority, C3 owns
// replay coordination, and C4 owns HTTP. Independent calls may share Authority;
// each C4 invocation still creates its own coordinator, not shared replay state.
func (c *Client) ExecuteAuthorizedFunctionRoundTrip(ctx context.Context, request ai.Request, authority *tool.Authority) (ai.Result, error) {
	definitions, executor := authority.Definitions(), authority.Executor()
	if len(definitions) == 0 || executor == nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	return c.ExecuteFunctionRoundTrip(ctx, request, definitions, executor)
}
