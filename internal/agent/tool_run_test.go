package agent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

type roundTripperFunc func(context.Context, ai.Request, *tool.Authority) (ai.Result, error)

func (f roundTripperFunc) ExecuteAuthorizedFunctionRoundTrip(ctx context.Context, req ai.Request, authority *tool.Authority) (ai.Result, error) {
	return f(ctx, req, authority)
}

type nilRoundTrip struct{}

func (*nilRoundTrip) ExecuteAuthorizedFunctionRoundTrip(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
	panic("nil round trip must never execute")
}

func testAuthority(t *testing.T) *tool.Authority {
	t.Helper()
	var handlers atomic.Int32
	a, err := tool.NewAuthority([]tool.Binding{{
		Definition: tool.Definition{Name: "private_authority_canary"},
		Handler: func(context.Context, tool.Call) (string, error) {
			handlers.Add(1)
			return "in-memory", nil
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if handlers.Load() != 0 {
			t.Error("Run construction/composition invoked a handler")
		}
	})
	return a
}

func newToolRun(t *testing.T, p AuthorizedToolRoundTripper) *Run {
	t.Helper()
	r, err := NewAuthorizedToolRun(p, request(), testAuthority(t), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestToolConstructionValidation(t *testing.T) {
	var calls atomic.Int32
	p := roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
		calls.Add(1)
		return result(), nil
	})
	a := testAuthority(t)
	empty, err := tool.NewAuthority(nil)
	if err != nil {
		t.Fatal(err)
	}
	var typedNil *nilRoundTrip
	var nilFunc roundTripperFunc
	for _, tc := range []struct {
		name    string
		p       AuthorizedToolRoundTripper
		req     ai.Request
		a       *tool.Authority
		timeout time.Duration
	}{
		{"nil", nil, request(), a, time.Second},
		{"typed nil", typedNil, request(), a, time.Second},
		{"nil function", nilFunc, request(), a, time.Second},
		{"invalid request", p, ai.Request{}, a, time.Second},
		{"zero timeout", p, request(), a, 0},
		{"negative timeout", p, request(), a, -time.Second},
		{"large timeout", p, request(), a, ai.MaxTimeout + 1},
		{"nil authority", p, request(), nil, time.Second},
		{"zero authority", p, request(), &tool.Authority{}, time.Second},
		{"deny all", p, request(), empty, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := NewAuthorizedToolRun(tc.p, tc.req, tc.a, tc.timeout)
			if r != nil || !errors.Is(err, ai.ErrInvalidRequest) || calls.Load() != 0 {
				t.Fatal("invalid construction escaped validation")
			}
		})
	}
	r := newToolRun(t, p)
	if r.State() != StateReady || calls.Load() != 0 {
		t.Fatal("construction executed work")
	}
	done := r.Done()
	openDone(t, r, done)
	r.Cancel()
	terminal(t, r, done, StateCanceled)
	if calls.Load() != 0 {
		t.Fatal("pre-start Cancel delegated work")
	}
}

func TestToolSuccessAggregateUsageAndOwnership(t *testing.T) {
	a := testAuthority(t)
	original := request()
	input := original
	aggregate := result()
	// Two individually valid 48-token turns aggregate above the request's 64.
	aggregate.Usage.OutputTokens = 96
	var calls int
	var r *Run
	r, err := NewAuthorizedToolRun(roundTripperFunc(func(ctx context.Context, req ai.Request, authority *tool.Authority) (ai.Result, error) {
		calls++
		if authority != a || req != original || ctx.Err() != nil || r.State() != StateRunning {
			t.Fatal("authority/request/context/state pass-through changed")
		}
		openDone(t, r, r.Done())
		return aggregate, nil
	}), input, a, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	input.Text = "changed"
	done := r.Done()
	if got, err := r.Execute(nil); got != (ai.Result{}) || err != ai.ErrInvalidRequest || r.State() != StateReady || calls != 0 {
		t.Fatal("nil context consumed tool Run")
	}
	got, err := r.Execute(context.Background())
	if err != nil || !reflect.DeepEqual(got, aggregate) || got.Usage == aggregate.Usage || calls != 1 {
		t.Fatal("aggregate result rejected, aliased, or repeated")
	}
	aggregate.Usage.OutputTokens = 1
	if got.Usage.OutputTokens != 96 {
		t.Fatal("provider mutation affected returned usage")
	}
	terminal(t, r, done, StateSucceeded)
}

func TestToolResultValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value ai.Result
		want  error
	}{
		{"empty", ai.Result{}, ai.ErrMalformedResponse},
		{"invalid model", ai.Result{Text: "text", Model: "bad model"}, ai.ErrMalformedResponse},
		{"terminal controls", ai.Result{Text: "\x1bunsafe", Model: "m"}, ai.ErrMalformedResponse},
		{"negative input", ai.Result{Text: "text", Model: "m", Usage: &ai.Usage{InputTokens: -1}}, ai.ErrMalformedResponse},
		{"negative output", ai.Result{Text: "text", Model: "m", Usage: &ai.Usage{OutputTokens: -1}}, ai.ErrMalformedResponse},
		{"oversize", ai.Result{Text: strings.Repeat("x", ai.MaxTextBytes+1), Model: "m"}, ai.ErrResponseTooLarge},
		{"nil usage", ai.Result{Text: "text", Model: "m"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newToolRun(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) { return tc.value, nil }))
			done := r.Done()
			got, err := r.Execute(context.Background())
			if tc.want == nil {
				if err != nil || !reflect.DeepEqual(got, tc.value) {
					t.Fatal("valid nil usage rejected")
				}
				terminal(t, r, done, StateSucceeded)
			} else {
				if got != (ai.Result{}) || !errors.Is(err, tc.want) {
					t.Fatal("result validation bypassed")
				}
				terminal(t, r, done, StateFailed)
			}
		})
	}
}

func TestToolPreCanceledContexts(t *testing.T) {
	for _, expired := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		want := context.Canceled
		cancel()
		if expired {
			ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			want = context.DeadlineExceeded
		}
		r := newToolRun(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
			t.Fatal("already completed context reached round trip")
			return ai.Result{}, nil
		}))
		done := r.Done()
		got, err := r.Execute(ctx)
		cancel()
		if got != (ai.Result{}) || err != want {
			t.Fatal("context identity lost")
		}
		terminal(t, r, done, StateCanceled)
	}
}

func TestToolInFlightCancellationAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
		want    []error
		state   State
	}{
		{"success suppressed", nil, []error{context.Canceled}, StateCanceled},
		{"context only", context.Canceled, []error{context.Canceled}, StateCanceled},
		{"context join", errors.Join(context.Canceled, context.DeadlineExceeded), []error{context.Canceled, context.DeadlineExceeded}, StateCanceled},
		{"independent", ai.ErrTransport, []error{ai.ErrTransport}, StateFailed},
		{"mixed", errors.Join(context.Canceled, ai.ErrProvider), []error{context.Canceled, ai.ErrProvider}, StateFailed},
		{"unknown", errors.New("private_error_canary private_authority_canary"), []error{ai.ErrProvider}, StateFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := make(chan context.Context, 1)
			release := make(chan struct{})
			var calls atomic.Int32
			r := newToolRun(t, roundTripperFunc(func(ctx context.Context, _ ai.Request, _ *tool.Authority) (ai.Result, error) {
				calls.Add(1)
				entered <- ctx
				<-release
				return result(), tc.failure
			}))
			done := r.Done()
			type outcome struct {
				result ai.Result
				err    error
			}
			finished := make(chan outcome, 1)
			go func() { got, err := r.Execute(context.Background()); finished <- outcome{got, err} }()
			ctx := <-entered
			for range 3 {
				r.Cancel()
			}
			if ctx.Err() != context.Canceled || r.State() != StateRunning {
				t.Fatal("cancellation lost or completion premature")
			}
			openDone(t, r, done)
			close(release)
			got := <-finished
			if got.result != (ai.Result{}) || calls.Load() != 1 {
				t.Fatal("partial result or duplicate work")
			}
			for _, want := range tc.want {
				if !errors.Is(got.err, want) {
					t.Fatal("safe category lost")
				}
			}
			if strings.Contains(got.err.Error(), "canary") {
				t.Fatal("unsafe error")
			}
			terminal(t, r, done, tc.state)
		})
	}
}

func TestToolTimeoutAndCallerDeadline(t *testing.T) {
	for _, callerFirst := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			parent := context.WithValue(context.Background(), struct{}{}, "caller")
			limit := 2 * time.Second
			if callerFirst {
				limit = time.Second / 2
			}
			ctx, cancel := context.WithTimeout(parent, limit)
			defer cancel()
			r := newToolRun(t, roundTripperFunc(func(operation context.Context, _ ai.Request, _ *tool.Authority) (ai.Result, error) {
				deadline, ok := operation.Deadline()
				want := time.Second
				if callerFirst {
					want = limit
				}
				if !ok || time.Until(deadline) != want || operation.Value(struct{}{}) != "caller" {
					t.Fatal("timeout/context not inherited")
				}
				<-operation.Done()
				// Even apparent success after the adapter deadline must be suppressed.
				return result(), nil
			}))
			done := r.Done()
			got, err := r.Execute(ctx)
			if got != (ai.Result{}) || err != context.DeadlineExceeded {
				t.Fatal("deadline success escaped")
			}
			terminal(t, r, done, StateCanceled)
		})
	}
}

func TestToolConcurrentExecuteAndCopies(t *testing.T) {
	const n = 32
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	r := newToolRun(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return result(), nil
	}))
	copyOfRun := *r
	done := r.Done()
	start, returned := make(chan struct{}), make(chan error, n)
	for i := range n {
		go func() {
			<-start
			target := r
			if i%2 == 0 {
				target = &copyOfRun
			}
			got, err := target.Execute(context.Background())
			if err != nil && got != (ai.Result{}) {
				t.Error("consumed run returned data")
			}
			returned <- err
		}()
	}
	close(start)
	<-entered
	for range n - 1 {
		if err := <-returned; err != ErrRunConsumed {
			t.Fatal("multiple claims")
		}
	}
	if calls.Load() != 1 || r.State() != StateRunning || copyOfRun.State() != StateRunning {
		t.Fatal("copy claim not shared")
	}
	openDone(t, r, done)
	close(release)
	if err := <-returned; err != nil {
		t.Fatal(err)
	}
	terminal(t, r, done, StateSucceeded)
	terminal(t, &copyOfRun, done, StateSucceeded)
	if calls.Load() != 1 {
		t.Fatal("round trip repeated")
	}
}

func TestToolRedaction(t *testing.T) {
	check := func(r *Run) {
		t.Helper()
		for _, output := range []string{r.String(), r.GoString(), fmt.Sprintf("%v", r), fmt.Sprintf("%+v", *r), fmt.Sprintf("%#v", r), fmt.Sprintf("%#v", *r), r.State().String(), ErrInvalidRun.Error(), ErrRunConsumed.Error()} {
			for _, canary := range []string{request().Text, request().Model, result().Text, "private_authority_canary", "private_error_canary", "private_credential_canary"} {
				if strings.Contains(output, canary) {
					t.Fatal("tool Run leaked retained data")
				}
			}
		}
	}
	for _, fail := range []bool{false, true} {
		var r *Run
		r = newToolRun(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
			check(r)
			if fail {
				return result(), errors.New("private_error_canary private_credential_canary private_authority_canary")
			}
			return result(), nil
		}))
		check(r)
		_, err := r.Execute(context.Background())
		if fail && err != ai.ErrProvider {
			t.Fatal("raw error exposed")
		}
		check(r)
	}
}
