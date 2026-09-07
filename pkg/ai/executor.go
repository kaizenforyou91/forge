package ai

import (
	"context"
	"reflect"
	"time"
)

// Executor is immutable; each Execute owns its own deadline and synchronous call.
type Executor struct {
	provider Provider
	timeout  time.Duration
}

// NewExecutor performs no I/O. timeout must be explicit; no core defaults apply.
func NewExecutor(provider Provider, timeout time.Duration) (*Executor, error) {
	if nilProvider(provider) {
		return nil, ErrInvalidRequest
	}
	if err := ValidateTimeout(timeout); err != nil {
		return nil, err
	}
	return &Executor{provider: provider, timeout: timeout}, nil
}

// A typed nil implementing Provider is also invalid. Reflection is confined to
// this interface-nil check; requests and results remain statically typed.
func nilProvider(p Provider) bool {
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

func (e *Executor) Execute(ctx context.Context, req Request) (Result, error) {
	if e == nil || nilProvider(e.provider) || ctx == nil {
		return Result{}, ErrInvalidRequest
	}
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		return Result{}, err
	}
	result, err := e.provider.Execute(requestCtx, req)
	if err != nil {
		return Result{}, SafeError(err)
	}
	if err := requestCtx.Err(); err != nil {
		return Result{}, err
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	if result.Usage != nil {
		if result.Usage.OutputTokens > int64(req.MaxOutputTokens) {
			return Result{}, ErrMalformedResponse
		}
		usage := *result.Usage
		result.Usage = &usage
	}
	return result, nil
}
