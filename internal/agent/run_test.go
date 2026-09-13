package agent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
)

type providerFunc func(context.Context, ai.Request) (ai.Result, error)

func (f providerFunc) Execute(ctx context.Context, req ai.Request) (ai.Result, error) {
	return f(ctx, req)
}

type nilProvider struct{}

func (*nilProvider) Execute(context.Context, ai.Request) (ai.Result, error) {
	panic("nil provider must never execute")
}

func request() ai.Request {
	return ai.Request{Text: "private_prompt_canary", Model: "private-model-canary", MaxOutputTokens: 64}
}

func result() ai.Result {
	return ai.Result{Text: "private_result_canary", Model: "private-model-canary", Usage: &ai.Usage{InputTokens: 1, OutputTokens: 2}}
}

func newRun(t *testing.T, p ai.Provider) *Run {
	t.Helper()
	r, err := NewRun(p, request(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func openDone(t *testing.T, r *Run, done <-chan struct{}) {
	t.Helper()
	if done == nil || r.Done() != done {
		t.Fatal("Done must be stable and non-nil")
	}
	select {
	case <-done:
		t.Fatal("Done closed before completion")
	default:
	}
}

func terminal(t *testing.T, r *Run, done <-chan struct{}, state State) {
	t.Helper()
	if r.Done() != done || done == nil {
		t.Fatal("Done identity changed")
	}
	select {
	case <-done:
	default:
		t.Fatal("terminal Done is open")
	}
	if r.State() != state {
		t.Fatalf("state = %s, want %s", r.State(), state)
	}
	r.core.mu.Lock()
	released := r.core.operation == nil && r.core.cancel == nil
	r.core.mu.Unlock()
	if !released {
		t.Fatal("terminal references retained")
	}
	for range 3 {
		r.Cancel()
	}
	if r.State() != state || r.Done() != done {
		t.Fatal("late Cancel rewrote terminal state")
	}
	got, err := r.Execute(context.Background())
	if got != (ai.Result{}) || err != ErrRunConsumed {
		t.Fatal("terminal Run executed again")
	}
}

func TestConstructionValidation(t *testing.T) {
	var calls atomic.Int32
	p := providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		calls.Add(1)
		return result(), nil
	})
	var typedNil *nilProvider
	var nilFunc providerFunc
	bad := request()
	bad.Text = strings.Repeat("x", ai.MaxInputBytes+1)
	cases := []struct {
		name    string
		p       ai.Provider
		req     ai.Request
		timeout time.Duration
	}{
		{"empty request", p, ai.Request{}, time.Second},
		{"nil provider", nil, request(), time.Second},
		{"typed nil", typedNil, request(), time.Second},
		{"nil function", nilFunc, request(), time.Second},
		{"zero timeout", p, request(), 0},
		{"negative timeout", p, request(), -time.Second},
		{"large timeout", p, request(), ai.MaxTimeout + 1},
		{"oversize request", p, bad, time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := NewRun(tc.p, tc.req, tc.timeout)
			if r != nil || !errors.Is(err, ai.ErrInvalidRequest) || calls.Load() != 0 {
				t.Fatal("invalid construction escaped AI validation")
			}
		})
	}
	r := newRun(t, p)
	if r.State() != StateReady || calls.Load() != 0 {
		t.Fatal("construction executed work")
	}
	openDone(t, r, r.Done())
	r.Cancel()
}

func TestInvalidRunsAndNilContext(t *testing.T) {
	for _, r := range []*Run{
		nil, {}, {core: &runState{}},
		{core: &runState{initialized: true, state: StateReady, done: make(chan struct{})}},
		{core: &runState{initialized: true, state: StateRunning, done: make(chan struct{})}},
		{core: &runState{initialized: true, state: State(255), done: make(chan struct{})}},
		{core: &runState{state: StateSucceeded, done: make(chan struct{})}},
	} {
		if r.State() != StateUnknown || r.Done() != nil {
			t.Fatal("unconstructed Run looks usable")
		}
		r.Cancel()
		got, err := r.Execute(context.Background())
		if got != (ai.Result{}) || err != ErrInvalidRun {
			t.Fatal("unconstructed Run did not fail closed")
		}
	}
	var calls int
	r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		calls++
		return result(), nil
	}))
	done := r.Done()
	got, err := r.Execute(nil)
	if got != (ai.Result{}) || err != ai.ErrInvalidRequest || r.State() != StateReady || calls != 0 {
		t.Fatal("nil context consumed Run")
	}
	openDone(t, r, done)
	got, err = r.Execute(context.Background())
	if err != nil || !reflect.DeepEqual(got, result()) || calls != 1 {
		t.Fatal("valid execution after nil context failed")
	}
	terminal(t, r, done, StateSucceeded)
}

func TestSuccessSnapshotAndSynchronousObservation(t *testing.T) {
	original := request()
	input := original
	var r *Run
	var calls int
	r, err := NewRun(providerFunc(func(ctx context.Context, req ai.Request) (ai.Result, error) {
		calls++
		if req != original || r.State() != StateRunning || ctx.Err() != nil {
			t.Fatal("snapshot/state/context changed")
		}
		openDone(t, r, r.Done())
		return result(), nil
	}), input, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	input.Text = "mutated"
	done := r.Done()
	got, err := r.Execute(context.Background())
	if err != nil || !reflect.DeepEqual(got, result()) || calls != 1 {
		t.Fatal("success/result count changed")
	}
	terminal(t, r, done, StateSucceeded)
}

func TestFailureClassification(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		want  []error
		state State
	}{
		{"provider", ai.ErrTransport, []error{ai.ErrTransport}, StateFailed},
		{"unknown", errors.New("private_error_canary"), []error{ai.ErrProvider}, StateFailed},
		{"mixed", errors.Join(context.Canceled, ai.ErrProvider), []error{context.Canceled, ai.ErrProvider}, StateFailed},
		{"unknown mixed", errors.Join(context.DeadlineExceeded, errors.New("private_error_canary")), []error{context.DeadlineExceeded, ai.ErrProvider}, StateFailed},
		{"canceled", context.Canceled, []error{context.Canceled}, StateCanceled},
		{"deadline", context.DeadlineExceeded, []error{context.DeadlineExceeded}, StateCanceled},
		{"context join", errors.Join(context.Canceled, context.DeadlineExceeded), []error{context.Canceled, context.DeadlineExceeded}, StateCanceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
				calls++
				return result(), tc.err
			}))
			done := r.Done()
			got, err := r.Execute(context.Background())
			if got != (ai.Result{}) || calls != 1 || err == nil || strings.Contains(err.Error(), "canary") {
				t.Fatal("unsafe failure or retained partial result")
			}
			for _, want := range tc.want {
				if !errors.Is(err, want) {
					t.Fatal("safe category lost")
				}
			}
			terminal(t, r, done, tc.state)
		})
	}
}

func TestExecutorResultValidationPreserved(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result ai.Result
		want   error
	}{
		{"empty", ai.Result{}, ai.ErrMalformedResponse},
		{"token limit", ai.Result{Text: "text", Model: "m", Usage: &ai.Usage{OutputTokens: 65}}, ai.ErrMalformedResponse},
		{"terminal controls", ai.Result{Text: "\x1bunsafe", Model: "m"}, ai.ErrMalformedResponse},
		{"text bound", ai.Result{Text: strings.Repeat("x", ai.MaxTextBytes+1), Model: "m"}, ai.ErrResponseTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return tc.result, nil }))
			done := r.Done()
			got, err := r.Execute(context.Background())
			if got != (ai.Result{}) || !errors.Is(err, tc.want) {
				t.Fatal("AI result validation bypassed")
			}
			terminal(t, r, done, StateFailed)
		})
	}
}

func TestPreStartCancel(t *testing.T) {
	r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		t.Fatal("pre-start canceled Run called provider")
		return ai.Result{}, nil
	}))
	done := r.Done()
	r.Cancel()
	terminal(t, r, done, StateCanceled)
}

func TestPreCanceledContextsConsumeRun(t *testing.T) {
	for _, expired := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		want := context.Canceled
		if expired {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			want = context.DeadlineExceeded
		} else {
			cancel()
		}
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("pre-canceled context reached provider")
			return ai.Result{}, nil
		}))
		done := r.Done()
		got, err := r.Execute(ctx)
		cancel()
		if got != (ai.Result{}) || err != want {
			t.Fatal("context identity changed")
		}
		terminal(t, r, done, StateCanceled)
	}
}

func TestInFlightCancellationAndFailurePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name        string
		providerErr error
		want        error
		state       State
	}{
		{"success suppressed", nil, context.Canceled, StateCanceled},
		{"context", context.Canceled, context.Canceled, StateCanceled},
		{"independent failure", ai.ErrTransport, ai.ErrTransport, StateFailed},
		{"mixed failure", errors.Join(context.Canceled, ai.ErrProvider), ai.ErrProvider, StateFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entered := make(chan context.Context, 1)
			release := make(chan struct{})
			var calls atomic.Int32
			r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
				calls.Add(1)
				entered <- ctx
				<-release
				return result(), tc.providerErr
			}))
			done := r.Done()
			type outcome struct {
				result ai.Result
				err    error
			}
			finished := make(chan outcome, 1)
			go func() { got, err := r.Execute(context.Background()); finished <- outcome{got, err} }()
			ctx := <-entered
			if r.State() != StateRunning {
				t.Fatal("running state not observable")
			}
			for range 4 {
				r.Cancel()
			}
			if ctx.Err() != context.Canceled || r.State() != StateRunning {
				t.Fatal("cancellation lost or prematurely terminal")
			}
			openDone(t, r, done)
			close(release)
			got := <-finished
			if got.result != (ai.Result{}) || !errors.Is(got.err, tc.want) || calls.Load() != 1 {
				t.Fatal("cancellation outcome/count changed")
			}
			if tc.name == "mixed failure" && !errors.Is(got.err, context.Canceled) {
				t.Fatal("mixed context category lost")
			}
			terminal(t, r, done, tc.state)
		})
	}
}

func TestCallerDeadlineAndExecutorTimeout(t *testing.T) {
	for _, callerFirst := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			parent := context.WithValue(context.Background(), struct{}{}, "caller")
			callerTimeout := 2 * time.Second
			if callerFirst {
				callerTimeout = time.Second / 2
			}
			ctx, cancel := context.WithTimeout(parent, callerTimeout)
			defer cancel()
			calls := 0
			r := newRun(t, providerFunc(func(operation context.Context, _ ai.Request) (ai.Result, error) {
				calls++
				if operation.Value(struct{}{}) != "caller" {
					t.Fatal("caller context detached")
				}
				deadline, ok := operation.Deadline()
				want := time.Second
				if callerFirst {
					want = callerTimeout
				}
				if !ok || time.Until(deadline) != want {
					t.Fatal("deadline replaced")
				}
				<-operation.Done()
				return ai.Result{}, operation.Err()
			}))
			done := r.Done()
			got, err := r.Execute(ctx)
			if got != (ai.Result{}) || err != context.DeadlineExceeded || calls != 1 {
				t.Fatal("deadline identity changed")
			}
			terminal(t, r, done, StateCanceled)
		})
	}
}

func TestConcurrentExecuteSingleClaimAndSharedCopies(t *testing.T) {
	const n = 32
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return result(), nil
	}))
	copyOfRun := *r
	done := r.Done()
	start := make(chan struct{})
	returned := make(chan error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			target := r
			if i%2 == 0 {
				target = &copyOfRun
			}
			got, err := target.Execute(context.Background())
			if err != nil && got != (ai.Result{}) {
				t.Error("consumed Run returned data")
			}
			returned <- err
		}()
	}
	close(start)
	<-entered
	for range n - 1 {
		if err := <-returned; err != ErrRunConsumed {
			t.Fatal("multiple execution claims")
		}
	}
	if calls.Load() != 1 || r.State() != StateRunning || copyOfRun.State() != StateRunning {
		t.Fatal("claim is not shared")
	}
	openDone(t, r, done)
	close(release)
	if err := <-returned; err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	terminal(t, r, done, StateSucceeded)
	terminal(t, &copyOfRun, done, StateSucceeded)
	if calls.Load() != 1 {
		t.Fatal("provider repeated")
	}
}

func TestConcurrentCancelExecuteAndObservation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			calls.Add(1)
			<-ctx.Done()
			return ai.Result{}, ctx.Err()
		}))
		done := r.Done()
		start := make(chan struct{})
		var wg sync.WaitGroup
		for range 16 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				if r.Done() != done {
					t.Error("Done identity raced")
				}
				_ = r.State()
				r.Cancel()
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := r.Execute(context.Background())
			if got != (ai.Result{}) || (err != ErrRunConsumed && err != context.Canceled) {
				t.Error("cancel/claim race escaped contract")
			}
		}()
		close(start)
		wg.Wait()
		if calls.Load() > 1 {
			t.Fatal("duplicate provider attempt")
		}
		terminal(t, r, done, StateCanceled)
	})
}

func TestTerminalSuccessPublicationBoundary(t *testing.T) {
	// Exercise the small completion boundary with an already-validated executor
	// success. This isolates cancellation after executor return without adding
	// production hooks or weakening the concrete ai.Executor dependency.
	for _, before := range []bool{false, true} {
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("unexpected provider call")
			return ai.Result{}, nil
		}))
		ctx, cancel := context.WithCancel(context.Background())
		r.core.mu.Lock()
		r.core.state, r.core.cancel = StateRunning, cancel
		r.core.mu.Unlock()
		done := r.Done()
		if before {
			r.Cancel()
			openDone(t, r, done)
		}
		got, err := r.core.complete(ctx, result(), nil)
		if before {
			if got != (ai.Result{}) || err != context.Canceled {
				t.Fatal("pre-publication cancellation lost")
			}
			terminal(t, r, done, StateCanceled)
		} else {
			if err != nil || !reflect.DeepEqual(got, result()) {
				t.Fatal("success changed")
			}
			terminal(t, r, done, StateSucceeded)
		}
	}
}

func TestStateAndRunRedaction(t *testing.T) {
	labels := []string{"unknown", "ready", "running", "succeeded", "failed", "canceled"}
	for i := 0; i <= 255; i++ {
		want := "unknown"
		if i < len(labels) {
			want = labels[i]
		}
		if State(i).String() != want {
			t.Fatal("State vocabulary escaped")
		}
	}
	if ErrInvalidRun.Error() != "agent: invalid run" || ErrRunConsumed.Error() != "agent: run already consumed" {
		t.Fatal("lifecycle error vocabulary changed")
	}
	check := func(r *Run) {
		t.Helper()
		outputs := []string{r.String(), r.GoString(), fmt.Sprintf("%v", r), fmt.Sprintf("%+v", *r), fmt.Sprintf("%#v", r), fmt.Sprintf("%#v", *r), r.State().String(), ErrInvalidRun.Error(), ErrRunConsumed.Error()}
		for _, output := range outputs {
			for _, canary := range []string{request().Text, request().Model, result().Text, "private_error_canary", "private_credential_canary"} {
				if strings.Contains(output, canary) {
					t.Fatal("Run formatting leaked data")
				}
			}
		}
	}
	for _, fail := range []bool{false, true} {
		var r *Run
		r = newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			check(r)
			if fail {
				return result(), errors.New("private_error_canary private_credential_canary")
			}
			return result(), nil
		}))
		check(r)
		_, err := r.Execute(context.Background())
		if fail && (err != ai.ErrProvider || strings.Contains(err.Error(), "canary")) {
			t.Fatal("raw provider error leaked")
		}
		check(r)
	}
}
