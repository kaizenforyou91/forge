package agent

import (
	"context"
	"reflect"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// AuthorizedToolRoundTripper is the existing bounded, authorized round-trip
// contract. Implementations must honor ctx and finish owned work before return.
// The caller supplies both the implementation and its immutable C5 authority.
type AuthorizedToolRoundTripper interface {
	ExecuteAuthorizedFunctionRoundTrip(context.Context, ai.Request, *tool.Authority) (ai.Result, error)
}

// NewAuthorizedToolRun owns one explicit authorized round trip. Construction
// validates only; it does not create authority or execute providers or handlers.
// This path does not compose stage diagnostics or change the existing CLI path.
func NewAuthorizedToolRun(roundTripper AuthorizedToolRoundTripper, request ai.Request, authority *tool.Authority, timeout time.Duration) (*Run, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if err := ai.ValidateTimeout(timeout); err != nil {
		return nil, err
	}
	if nilRoundTripper(roundTripper) || authority.Executor() == nil || len(authority.Definitions()) == 0 {
		return nil, ai.ErrInvalidRequest
	}
	return newRunState(func(ctx context.Context) (ai.Result, error) {
		operationCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := operationCtx.Err(); err != nil {
			return ai.Result{}, err
		}
		result, err := roundTripper.ExecuteAuthorizedFunctionRoundTrip(operationCtx, request, authority)
		if err != nil {
			return ai.Result{}, err
		}
		if err := operationCtx.Err(); err != nil {
			return ai.Result{}, err
		}
		if err := result.Validate(); err != nil {
			return ai.Result{}, err
		}
		// Usage aggregates independently bounded turns. ai.Executor's single-turn
		// OutputTokens comparison must not be applied to this combined result.
		if result.Usage != nil {
			usage := *result.Usage
			result.Usage = &usage
		}
		return result, nil
	}), nil
}

// Reflection is restricted to rejecting typed nil implementations, as in the
// existing text executor. It does not inspect provider or handler internals.
func nilRoundTripper(p AuthorizedToolRoundTripper) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
