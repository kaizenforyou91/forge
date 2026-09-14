package agent

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func sequenceTextStep(t *testing.T, p ai.Provider) SequenceStep {
	t.Helper()
	s, err := NewTextSequenceStep(p, request(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sequenceToolStep(t *testing.T, p AuthorizedToolRoundTripper, a *tool.Authority) SequenceStep {
	t.Helper()
	s, err := NewAuthorizedToolSequenceStep(p, request(), a, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func newSequence(t *testing.T, steps ...SequenceStep) *Sequence {
	t.Helper()
	s, err := NewSequence(steps, ai.MaxTimeout)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sequenceOpen(t *testing.T, s *Sequence, done <-chan struct{}) {
	t.Helper()
	if done == nil || s.Done() != done {
		t.Fatal("unstable Done")
	}
	select {
	case <-done:
		t.Fatal("premature Done")
	default:
	}
}

func sequenceTerminal(t *testing.T, s *Sequence, done <-chan struct{}, state State) {
	t.Helper()
	if s.State() != state || done == nil || s.Done() != done {
		t.Fatalf("terminal state/Done changed: %s, want %s", s.State(), state)
	}
	select {
	case <-done:
	default:
		t.Fatal("orphaned Done")
	}
	s.core.mu.Lock()
	released := s.core.steps == nil && s.core.active == nil && s.core.cancel == nil
	s.core.mu.Unlock()
	if !released {
		t.Fatal("terminal references retained")
	}
	s.Cancel()
	got, err := s.Execute(context.Background())
	if got != (SequenceResult{}) || err != ErrSequenceConsumed || s.State() != state || s.Done() != done {
		t.Fatal("terminal state rewritten or execution repeated")
	}
}

func TestSequenceConstruction(t *testing.T) {
	var calls atomic.Int32
	p := providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		calls.Add(1)
		return result(), nil
	})
	rt := roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
		calls.Add(1)
		return result(), nil
	})
	a := testAuthority(t) // Its cleanup independently requires zero handler calls.
	var typedProvider *nilProvider
	var nilProviderFunc providerFunc
	var typedTrip *nilRoundTrip
	var nilTripFunc roundTripperFunc
	empty, err := tool.NewAuthority(nil)
	if err != nil {
		t.Fatal(err)
	}
	badText := request()
	badText.Text = strings.Repeat("x", ai.MaxInputBytes+1)
	for _, provider := range []ai.Provider{nil, typedProvider, nilProviderFunc} {
		if step, err := NewTextSequenceStep(provider, request(), time.Second); err != ai.ErrInvalidRequest || !reflect.DeepEqual(step, SequenceStep{}) {
			t.Fatal("nil provider accepted")
		}
	}
	for _, trip := range []AuthorizedToolRoundTripper{nil, typedTrip, nilTripFunc} {
		if step, err := NewAuthorizedToolSequenceStep(trip, request(), a, time.Second); err != ai.ErrInvalidRequest || !reflect.DeepEqual(step, SequenceStep{}) {
			t.Fatal("nil round trip accepted")
		}
	}
	for _, authority := range []*tool.Authority{nil, {}, empty} {
		if _, err := NewAuthorizedToolSequenceStep(rt, request(), authority, time.Second); err != ai.ErrInvalidRequest {
			t.Fatal("invalid authority accepted")
		}
	}
	for _, req := range []ai.Request{{}, badText} {
		if _, err := NewTextSequenceStep(p, req, time.Second); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid text request accepted")
		}
		if _, err := NewAuthorizedToolSequenceStep(rt, req, a, time.Second); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid tool request accepted")
		}
	}
	text := sequenceTextStep(t, p)
	for _, timeout := range []time.Duration{0, -1, ai.MaxTimeout + 1} {
		if _, err := NewTextSequenceStep(p, request(), timeout); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid text timeout accepted")
		}
		if _, err := NewAuthorizedToolSequenceStep(rt, request(), a, timeout); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid tool timeout accepted")
		}
		if s, err := NewSequence([]SequenceStep{text}, timeout); s != nil || !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid overall timeout accepted")
		}
	}
	for _, n := range []int{0, 1, 8, 9} {
		steps := make([]SequenceStep, n)
		for i := range steps {
			steps[i] = text
		}
		s, err := NewSequence(steps, ai.MaxTimeout)
		if n == 0 || n == 9 {
			if s != nil || err != ErrInvalidSequence {
				t.Fatal("step count bound escaped")
			}
		} else {
			if err != nil || s.State() != StateReady || s.core.active != nil {
				t.Fatal("valid sequence failed or preconstructed a Run")
			}
			sequenceOpen(t, s, s.Done())
			s.Cancel()
		}
	}
	toolStep := sequenceToolStep(t, rt, a)
	newSequence(t, text, toolStep).Cancel()
	// Even package-local malformed declarations must be revalidated.
	for _, change := range []func(*SequenceStep){
		func(s *SequenceStep) { *s = SequenceStep{} },
		func(s *SequenceStep) { s.kind = 255 },
		func(s *SequenceStep) { s.request = ai.Request{} },
		func(s *SequenceStep) { s.timeout = 0 },
		func(s *SequenceStep) { s.provider = nil },
		func(s *SequenceStep) { s.authority = a },
		func(s *SequenceStep) { *s = toolStep; s.roundTripper = typedTrip },
		func(s *SequenceStep) { *s = toolStep; s.authority = nil },
		func(s *SequenceStep) { *s = toolStep; s.provider = p },
	} {
		invalid := text
		change(&invalid)
		if s, err := NewSequence([]SequenceStep{invalid}, time.Second); s != nil || err == nil {
			t.Fatal("malformed declaration accepted")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("constructor executed work")
	}
}

func TestSequenceLiteralOrderSnapshotAndFinalResult(t *testing.T) {
	for _, kinds := range []string{"T", "TTT", "U", "TUT", "TTTTTTTT"} {
		t.Run(kinds, func(t *testing.T) {
			var s *Sequence
			var seen []string
			var children []*Run
			a := testAuthority(t)
			steps := make([]SequenceStep, len(kinds))
			for i, kind := range kinds {
				req := request()
				req.Text = fmt.Sprintf("literal-%d", i)
				original := req
				output := result()
				output.Text = fmt.Sprintf("output-%d-not-next-input", i)
				if kind == 'U' {
					output.Usage = &ai.Usage{InputTokens: 7, OutputTokens: 96}
				} else if i == len(kinds)-1 && i > 0 {
					output.Usage = nil // Never infer/sum earlier known usage.
				}
				execute := func(ctx context.Context, got ai.Request) (ai.Result, error) {
					if got != original || ctx.Err() != nil || len(seen) != i || s.State() != StateRunning {
						t.Fatal("literal/order/snapshot/state changed")
					}
					sequenceOpen(t, s, s.Done())
					s.core.mu.Lock()
					child := s.core.active
					s.core.mu.Unlock()
					if child == nil || child.State() != StateRunning {
						t.Fatal("no owned fresh Run")
					}
					for _, old := range children {
						if old == child || old.State() != StateSucceeded {
							t.Fatal("Run reused or children overlapped")
						}
					}
					children = append(children, child)
					seen = append(seen, got.Text)
					return output, nil
				}
				var err error
				if kind == 'T' {
					steps[i], err = NewTextSequenceStep(providerFunc(execute), req, time.Second)
				} else {
					steps[i], err = NewAuthorizedToolSequenceStep(roundTripperFunc(func(ctx context.Context, got ai.Request, authority *tool.Authority) (ai.Result, error) {
						if authority != a || got.MaxOutputTokens != 64 {
							t.Fatal("authority or per-turn cap changed")
						}
						return execute(ctx, got)
					}), req, a, time.Second)
				}
				if err != nil {
					t.Fatal(err)
				}
				req.Text = "caller mutation"
			}
			s = newSequence(t, steps...)
			if s.core.active != nil || len(seen) != 0 {
				t.Fatal("construction performed work")
			}
			for i := range steps {
				steps[i] = SequenceStep{} // Cannot mutate retained specifications.
			}
			done := s.Done()
			got, err := s.Execute(context.Background())
			if err != nil || got.CompletedSteps != len(kinds) || len(seen) != len(kinds) || got.Final.Text != fmt.Sprintf("output-%d-not-next-input", len(kinds)-1) {
				t.Fatal("wrong final result/count")
			}
			if kinds == "U" && got.Final.Usage.OutputTokens != 96 {
				t.Fatal("64-cap/96 aggregate tool result rejected")
			}
			if len(kinds) > 1 && got.Final.Usage != nil {
				t.Fatal("earlier usage leaked into final usage")
			}
			sequenceTerminal(t, s, done, StateSucceeded)
		})
	}
}

func TestSequenceInvalidAndNilContext(t *testing.T) {
	for _, s := range []*Sequence{nil, {}, {core: &sequenceState{}}, {core: &sequenceState{initialized: true, state: StateReady, done: make(chan struct{})}}} {
		s.Cancel()
		got, err := s.Execute(context.Background())
		if err != ErrInvalidSequence || got != (SequenceResult{}) || s.Done() != nil || s.State() != StateUnknown {
			t.Fatal("unconstructed sequence did not fail closed")
		}
	}
	s := newSequence(t, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return result(), nil })))
	done := s.Done()
	if got, err := s.Execute(nil); got != (SequenceResult{}) || err != ai.ErrInvalidRequest || s.State() != StateReady {
		t.Fatal("nil context consumed")
	}
	sequenceOpen(t, s, done)
	if _, err := s.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	sequenceTerminal(t, s, done, StateSucceeded)
}

func TestSequenceFailFastClassification(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		want  []error
		state State
	}{
		{"context", context.Canceled, []error{context.Canceled}, StateCanceled},
		{"deadline", context.DeadlineExceeded, []error{context.DeadlineExceeded}, StateCanceled},
		{"transport", ai.ErrTransport, []error{ai.ErrTransport}, StateFailed},
		{"raw", errors.New("private_error_canary"), []error{ai.ErrProvider}, StateFailed},
		{"mixed", errors.Join(context.Canceled, errors.New("private_error_canary")), []error{context.Canceled, ai.ErrProvider}, StateFailed},
		{"context join", errors.Join(context.Canceled, context.DeadlineExceeded), []error{context.Canceled, context.DeadlineExceeded}, StateCanceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, toolMode := range []bool{false, true} {
				calls := 0
				first := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { calls++; return result(), nil }))
				fail := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { calls++; return result(), tc.err }))
				if toolMode {
					fail = sequenceToolStep(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
						calls++
						return result(), tc.err
					}), testAuthority(t))
				}
				s := newSequence(t, first, fail, first)
				done := s.Done()
				got, err := s.Execute(context.Background())
				if got != (SequenceResult{}) || calls != 2 || err == nil || strings.Contains(err.Error(), "canary") {
					t.Fatal("partial result, retry, later work, or unsafe error")
				}
				for _, want := range tc.want {
					if !errors.Is(err, want) {
						t.Fatal("safe error category lost")
					}
				}
				sequenceTerminal(t, s, done, tc.state)
			}
		})
	}
}

func TestSequencePreCancellation(t *testing.T) {
	for _, mode := range []string{"ready", "caller", "expired"} {
		t.Run(mode, func(t *testing.T) {
			p := providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
				t.Fatal("canceled sequence performed I/O")
				return ai.Result{}, nil
			})
			rt := roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
				t.Fatal("canceled tool called")
				return ai.Result{}, nil
			})
			for _, step := range []SequenceStep{sequenceTextStep(t, p), sequenceToolStep(t, rt, testAuthority(t))} {
				s := newSequence(t, step)
				done := s.Done()
				ctx, cancel := context.WithCancel(context.Background())
				want := context.Canceled
				switch mode {
				case "ready":
					s.Cancel()
					want = ErrSequenceConsumed
				case "caller":
					cancel()
				case "expired":
					cancel()
					ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
					want = context.DeadlineExceeded
				}
				got, err := s.Execute(ctx)
				cancel()
				if got != (SequenceResult{}) || err != want {
					t.Fatal("pre-cancel contract changed")
				}
				sequenceTerminal(t, s, done, StateCanceled)
			}
		})
	}
}

func TestSequenceRunningCancelWaitsAndPreventsLaterWork(t *testing.T) {
	for _, failure := range []bool{false, true} {
		entered, release := make(chan context.Context, 1), make(chan struct{})
		var calls atomic.Int32
		p := providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			calls.Add(1)
			entered <- ctx
			<-release
			if failure {
				return result(), errors.Join(context.Canceled, ai.ErrTransport)
			}
			return result(), nil // Cancellation still suppresses returned success.
		})
		s := newSequence(t, sequenceTextStep(t, p), sequenceTextStep(t, p))
		done := s.Done()
		finished := make(chan error, 1)
		go func() {
			got, err := s.Execute(context.Background())
			if got != (SequenceResult{}) {
				t.Error("canceled result retained data")
			}
			finished <- err
		}()
		ctx := <-entered
		s.Cancel()
		if s.State() != StateRunning || ctx.Err() != context.Canceled {
			t.Fatal("cancellation lost or early terminal")
		}
		sequenceOpen(t, s, done)
		close(release)
		err := <-finished
		state := StateCanceled
		if failure {
			state = StateFailed
			if !errors.Is(err, ai.ErrTransport) {
				t.Fatal("failure suppressed")
			}
		}
		if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
			t.Fatal("later step or lost cancellation")
		}
		sequenceTerminal(t, s, done, state)
	}
}

func TestSequenceTransitionAndPublicationCancellation(t *testing.T) {
	// Isolate the inter-step and terminal decision points without timing sleeps
	// or production hooks. The first real Run has completed before cancellation.
	for _, beforeNext := range []bool{false, true} {
		calls := 0
		step := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { calls++; return result(), nil }))
		s := newSequence(t, step, step)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		s.core.state, s.core.cancel = StateRunning, cancel
		done := s.Done()
		first, err := s.core.nextRun(ctx, 0, "")
		if err != nil {
			t.Fatal(err)
		}
		got, err := first.Execute(ctx)
		if err != nil {
			t.Fatal(err)
		}
		s.core.active = nil
		s.Cancel()
		if beforeNext {
			next, err := s.core.nextRun(ctx, 1, got.Text)
			if next != nil || err != context.Canceled || s.core.active != nil {
				t.Fatal("post-cancel child constructed")
			}
		}
		final, err := s.core.complete(ctx, SequenceResult{Final: got, CompletedSteps: 1}, nil)
		if final != (SequenceResult{}) || err != context.Canceled || calls != 1 {
			t.Fatal("cancellation lost at publication")
		}
		sequenceTerminal(t, s, done, StateCanceled)
	}
}

func TestSequenceConcurrentClaimAndCopies(t *testing.T) {
	const n = 24
	var calls atomic.Int32
	entered, release, start := make(chan struct{}), make(chan struct{}), make(chan struct{})
	s := newSequence(t, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return result(), nil
	})))
	copyOfSequence := *s
	done := s.Done()
	returned := make(chan error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			target := s
			if i%2 == 0 {
				target = &copyOfSequence
			}
			got, err := target.Execute(context.Background())
			if err != nil && got != (SequenceResult{}) {
				t.Error("consumed returned data")
			}
			returned <- err
		}()
	}
	close(start)
	<-entered
	for range n - 1 {
		if err := <-returned; err != ErrSequenceConsumed {
			t.Fatal("multiple claims")
		}
	}
	if copyOfSequence.State() != StateRunning || copyOfSequence.Done() != done || calls.Load() != 1 {
		t.Fatal("copies do not share core")
	}
	sequenceOpen(t, s, done)
	close(release)
	if err := <-returned; err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	sequenceTerminal(t, s, done, StateSucceeded)
	sequenceTerminal(t, &copyOfSequence, done, StateSucceeded)
}

func TestSequenceDeadlines(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		caller, overall, child, want time.Duration
	}{
		{"overall", 10 * time.Second, 2 * time.Second, 5 * time.Second, 2 * time.Second},
		{"child", 10 * time.Second, 5 * time.Second, time.Second, time.Second},
		{"caller", time.Second, 5 * time.Second, 3 * time.Second, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, toolMode := range []bool{false, true} {
				synctest.Test(t, func(t *testing.T) {
					start := time.Now()
					calls := 0
					parent := context.WithValue(context.Background(), struct{}{}, "caller")
					ctx, cancel := context.WithTimeout(parent, tc.caller)
					defer cancel()
					execute := func(ctx context.Context, _ ai.Request) (ai.Result, error) {
						calls++
						deadline, ok := ctx.Deadline()
						if !ok || !deadline.Equal(start.Add(tc.want)) || ctx.Value(struct{}{}) != "caller" {
							t.Fatal("deadline/parent detached")
						}
						<-ctx.Done()
						return ai.Result{}, ctx.Err()
					}
					step := sequenceTextStep(t, providerFunc(execute))
					if toolMode {
						step = sequenceToolStep(t, roundTripperFunc(func(c context.Context, r ai.Request, _ *tool.Authority) (ai.Result, error) { return execute(c, r) }), testAuthority(t))
					}
					step.timeout = tc.child
					s, err := NewSequence([]SequenceStep{step, step}, tc.overall)
					if err != nil {
						t.Fatal(err)
					}
					done := s.Done()
					got, err := s.Execute(ctx)
					if got != (SequenceResult{}) || err != context.DeadlineExceeded || calls != 1 || time.Since(start) != tc.want {
						t.Fatal("timeout outcome changed")
					}
					sequenceTerminal(t, s, done, StateCanceled)
				})
			}
		})
	}
}

func TestSequenceOverallDeadlineNotReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		calls := 0
		p := providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			calls++
			deadline, _ := ctx.Deadline()
			if !deadline.Equal(start.Add(3 * time.Second)) {
				t.Fatal("overall deadline reset")
			}
			if calls == 1 {
				time.Sleep(2 * time.Second)
				return result(), nil
			} // Fake synctest time.
			if time.Until(deadline) != time.Second {
				t.Fatal("remaining time extended")
			}
			<-ctx.Done()
			return ai.Result{}, ctx.Err()
		})
		step := sequenceTextStep(t, p)
		step.timeout = 10 * time.Second
		s, err := NewSequence([]SequenceStep{step, step, step}, 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		done := s.Done()
		got, err := s.Execute(context.Background())
		if got != (SequenceResult{}) || err != context.DeadlineExceeded || calls != 2 || time.Since(start) != 3*time.Second {
			t.Fatal("deadline reset or fail-fast lost")
		}
		sequenceTerminal(t, s, done, StateCanceled)
	})
}

func TestSequencePanicUnwinds(t *testing.T) {
	for _, toolMode := range []bool{false, true} {
		var s *Sequence
		var owned context.Context
		marker := &struct{ private string }{"private_panic_canary"}
		p := providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) { owned = ctx; panic(marker) })
		step := sequenceTextStep(t, p)
		if toolMode {
			step = sequenceToolStep(t, roundTripperFunc(func(ctx context.Context, _ ai.Request, _ *tool.Authority) (ai.Result, error) {
				owned = ctx
				panic(marker)
			}), testAuthority(t))
		}
		s = newSequence(t, step, step)
		done := s.Done()
		func() {
			defer func() {
				if recover() != marker {
					t.Error("original panic replaced or swallowed")
				}
			}()
			_, _ = s.Execute(context.Background())
			t.Error("provider panic returned normally")
		}()
		if owned == nil || owned.Err() != context.Canceled {
			t.Fatal("panic context retained")
		}
		sequenceTerminal(t, s, done, StateFailed)
	}
}

func TestSequenceRedaction(t *testing.T) {
	step := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return result(), nil }))
	s := newSequence(t, step)
	for _, value := range []any{s, *s, step, &step} {
		for _, format := range []string{"%v", "%+v", "%#v"} {
			got := fmt.Sprintf(format, value)
			if strings.Contains(got, "canary") || !strings.Contains(got, "data redacted") {
				t.Fatal("format exposes retained data")
			}
		}
	}
	if ErrInvalidSequence.Error() != "agent: invalid sequence" || ErrSequenceConsumed.Error() != "agent: sequence already consumed" {
		t.Fatal("unsafe lifecycle vocabulary")
	}
	s.Cancel()
	sequenceTerminal(t, s, s.Done(), StateCanceled)
}

func TestSequencePreviousConstruction(t *testing.T) {
	var calls atomic.Int32
	p := providerFunc(func(context.Context, ai.Request) (ai.Result, error) { calls.Add(1); return result(), nil })
	rt := roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
		calls.Add(1)
		return result(), nil
	})
	a := testAuthority(t)
	literal := sequenceTextStep(t, p)
	if literal.inputMode != sequenceLiteralText || sequenceToolStep(t, rt, a).inputMode != sequenceLiteralText {
		t.Fatal("B1 constructor mode changed")
	}
	text, err := NewTextSequenceStepFromPrevious(p, request().Model, 64, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	toolStep, err := NewAuthorizedToolSequenceStepFromPrevious(rt, request().Model, 64, a, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []SequenceStep{text, toolStep} {
		if step.inputMode != sequencePreviousStepText || step.request.Text != "" {
			t.Fatal("probe retained or wrong mode")
		}
		if s, err := NewSequence([]SequenceStep{step}, time.Second); s != nil || err != ErrInvalidSequence {
			t.Fatal("first previous accepted")
		}
		newSequence(t, literal, step).Cancel()
		for _, format := range []string{"%v", "%+v", "%#v"} {
			if strings.Contains(fmt.Sprintf(format, step), "sequence validation") {
				t.Fatal("probe exposed")
			}
		}
	}
	var typedProvider *nilProvider
	var typedTrip *nilRoundTrip
	var nilProviderFn providerFunc
	var nilTripFn roundTripperFunc
	for _, invalid := range []ai.Provider{nil, typedProvider, nilProviderFn} {
		if _, err := NewTextSequenceStepFromPrevious(invalid, request().Model, 64, time.Second); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("nil previous provider accepted")
		}
	}
	for _, invalid := range []AuthorizedToolRoundTripper{nil, typedTrip, nilTripFn} {
		if _, err := NewAuthorizedToolSequenceStepFromPrevious(invalid, request().Model, 64, a, time.Second); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("nil previous round trip accepted")
		}
	}
	empty, err := tool.NewAuthority(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []*tool.Authority{nil, {}, empty} {
		if _, err := NewAuthorizedToolSequenceStepFromPrevious(rt, request().Model, 64, invalid, time.Second); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid previous authority accepted")
		}
	}
	for _, tc := range []struct {
		model   string
		tokens  int
		timeout time.Duration
	}{
		{"", 64, time.Second}, {"bad model", 64, time.Second}, {strings.Repeat("m", ai.MaxModelBytes+1), 64, time.Second},
		{"m", ai.MinOutputTokens - 1, time.Second}, {"m", ai.MaxOutputTokens + 1, time.Second},
		{"m", 64, 0}, {"m", 64, -1}, {"m", 64, ai.MaxTimeout + 1},
	} {
		if _, err := NewTextSequenceStepFromPrevious(p, tc.model, tc.tokens, tc.timeout); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid previous text spec accepted")
		}
		if _, err := NewAuthorizedToolSequenceStepFromPrevious(rt, tc.model, tc.tokens, a, tc.timeout); !errors.Is(err, ai.ErrInvalidRequest) {
			t.Fatal("invalid previous tool spec accepted")
		}
	}
	for _, change := range []func(*SequenceStep){
		func(s *SequenceStep) { s.inputMode = 0 }, func(s *SequenceStep) { s.inputMode = 255 },
		func(s *SequenceStep) { s.request.Text = "must not be retained" },
		func(s *SequenceStep) { s.request.Model = "" }, func(s *SequenceStep) { s.request.MaxOutputTokens = 0 },
		func(s *SequenceStep) { s.timeout = 0 }, func(s *SequenceStep) { s.kind = 255 },
		func(s *SequenceStep) { s.provider = nil },
		func(s *SequenceStep) { *s = toolStep; s.authority = nil },
	} {
		bad := text
		change(&bad)
		if s, err := NewSequence([]SequenceStep{literal, bad}, time.Second); s != nil || err == nil {
			t.Fatal("malformed previous spec accepted")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("constructor performed work")
	}
}

func TestSequencePreviousHandoffAndAuthority(t *testing.T) {
	for _, kinds := range []string{"TT", "TU", "UT", "UU", "TUTUT"} {
		t.Run(kinds, func(t *testing.T) {
			a := testAuthority(t)
			definitions := a.Definitions()
			steps := make([]SequenceStep, len(kinds))
			calls := 0
			var expectedFinal ai.Result
			for i, kind := range kinds {
				// The third node is literal even after output; subsequent nodes
				// must receive only their immediate predecessor's output.
				literal := i == 0 || i == 2
				wantText := fmt.Sprintf(" \tprevious-%d\nline\n\u00e9 ", i-1)
				if literal {
					wantText = fmt.Sprintf("literal-%d", i)
				}
				model := fmt.Sprintf("caller-model-%d", i)
				tokens := 64 + i
				out := result()
				out.Text = fmt.Sprintf(" \tprevious-%d\nline\n\u00e9 ", i)
				if i == len(kinds)-1 {
					expectedFinal = out
				}
				execute := func(ctx context.Context, req ai.Request) (ai.Result, error) {
					if calls != i || req.Text != wantText || req.Model != model || req.MaxOutputTokens != tokens || ctx.Err() != nil {
						t.Fatal("handoff/order/caller authority changed")
					}
					calls++
					return out, nil
				}
				var err error
				if kind == 'T' {
					p := providerFunc(execute)
					if literal {
						steps[i], err = NewTextSequenceStep(p, ai.Request{Text: wantText, Model: model, MaxOutputTokens: tokens}, time.Second)
					} else {
						steps[i], err = NewTextSequenceStepFromPrevious(p, model, tokens, time.Second)
					}
				} else {
					rt := roundTripperFunc(func(ctx context.Context, req ai.Request, authority *tool.Authority) (ai.Result, error) {
						if authority != a || !reflect.DeepEqual(authority.Definitions(), definitions) {
							t.Fatal("tool authority changed")
						}
						return execute(ctx, req)
					})
					if literal {
						steps[i], err = NewAuthorizedToolSequenceStep(rt, ai.Request{Text: wantText, Model: model, MaxOutputTokens: tokens}, a, time.Second)
					} else {
						steps[i], err = NewAuthorizedToolSequenceStepFromPrevious(rt, model, tokens, a, time.Second)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			s := newSequence(t, steps...)
			done := s.Done()
			got, err := s.Execute(context.Background())
			if err != nil || calls != len(kinds) || got.CompletedSteps != calls || !reflect.DeepEqual(got.Final, expectedFinal) {
				t.Fatal("handoff result or call count changed")
			}
			if got.AggregateUsage == nil || got.AggregateUsage.InputTokens != int64(calls) || got.AggregateUsage.OutputTokens != int64(2*calls) {
				t.Fatal("known mixed usage sum changed")
			}
			sequenceTerminal(t, s, done, StateSucceeded)
		})
	}
}

func TestSequenceHandoffBoundsAndInvalidResults(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       error
	}{
		{"exact input bound", strings.Repeat("x", ai.MaxInputBytes), nil},
		{"over input bound", strings.Repeat("x", ai.MaxInputBytes+1), ai.ErrInvalidRequest},
		{"blank child", " \t\n", ai.ErrMalformedResponse},
		{"invalid utf8 child", "\xff", ai.ErrMalformedResponse},
		{"malformed child", "\x1bunsafe", ai.ErrMalformedResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, toolMode := range []bool{false, true} {
				firstCalls, nextCalls := 0, 0
				first := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
					firstCalls++
					r := result()
					r.Text = tc.text
					return r, nil
				}))
				nextExecute := func(_ context.Context, r ai.Request) (ai.Result, error) {
					nextCalls++
					if r.Text != tc.text {
						t.Fatal("handoff truncated or rewritten")
					}
					return result(), nil
				}
				next, err := NewTextSequenceStepFromPrevious(providerFunc(nextExecute), request().Model, 64, time.Second)
				if toolMode {
					next, err = NewAuthorizedToolSequenceStepFromPrevious(roundTripperFunc(func(ctx context.Context, r ai.Request, _ *tool.Authority) (ai.Result, error) {
						return nextExecute(ctx, r)
					}), request().Model, 64, testAuthority(t), time.Second)
				}
				if err != nil {
					t.Fatal(err)
				}
				s := newSequence(t, first, next)
				done := s.Done()
				got, err := s.Execute(context.Background())
				if firstCalls != 1 || !errors.Is(err, tc.want) {
					t.Fatal("result validation or retry behavior changed")
				}
				if tc.want != nil {
					if got != (SequenceResult{}) || nextCalls != 0 || strings.Contains(err.Error(), tc.text) {
						t.Fatal("invalid handoff exposed data or executed next work")
					}
					sequenceTerminal(t, s, done, StateFailed)
				} else {
					if got.CompletedSteps != 2 || nextCalls != 1 {
						t.Fatal("boundary rejected or extra call")
					}
					sequenceTerminal(t, s, done, StateSucceeded)
				}
			}
		})
	}
	// Isolate resolution before child construction using valid >16 KiB Result
	// text. This does not bypass child Result validation in the end-to-end test.
	first := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return result(), nil }))
	next, err := NewTextSequenceStepFromPrevious(first.provider, request().Model, 64, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	s := newSequence(t, first, next)
	child, err := s.core.nextRun(context.Background(), 1, strings.Repeat("x", ai.MaxInputBytes+1))
	if child != nil || s.core.active != nil || !errors.Is(err, ai.ErrInvalidRequest) {
		t.Fatal("bad handoff constructed a child")
	}
	s.Cancel()
}

func TestSequenceAggregateUsage(t *testing.T) {
	for _, tc := range []struct {
		name, kinds string
		usage       []*ai.Usage
		want        *ai.Usage
		failAt      int
	}{
		{"known text", "TT", []*ai.Usage{{InputTokens: 3, OutputTokens: 5}, {InputTokens: 7, OutputTokens: 11}}, &ai.Usage{InputTokens: 10, OutputTokens: 16}, 0},
		{"known tool", "UU", []*ai.Usage{{InputTokens: 3, OutputTokens: 96}, {InputTokens: 7, OutputTokens: 96}}, &ai.Usage{InputTokens: 10, OutputTokens: 192}, 0},
		{"mixed 64-96", "UT", []*ai.Usage{{InputTokens: 3, OutputTokens: 96}, {InputTokens: 7, OutputTokens: 11}}, &ai.Usage{InputTokens: 10, OutputTokens: 107}, 0},
		{"first unknown", "TTT", []*ai.Usage{nil, {InputTokens: 7}, {OutputTokens: 9}}, nil, 0},
		{"middle unknown", "TTT", []*ai.Usage{{InputTokens: 7}, nil, {OutputTokens: 9}}, nil, 0},
		{"final unknown", "TTT", []*ai.Usage{{InputTokens: 7}, {OutputTokens: 9}, nil}, nil, 0},
		{"unknown before large", "UUU", []*ai.Usage{nil, {InputTokens: math.MaxInt64, OutputTokens: math.MaxInt64}, {InputTokens: math.MaxInt64, OutputTokens: math.MaxInt64}}, nil, 0},
		{"unknown after large", "UUUU", []*ai.Usage{{InputTokens: math.MaxInt64, OutputTokens: math.MaxInt64}, nil, {InputTokens: math.MaxInt64, OutputTokens: math.MaxInt64}, {InputTokens: 1, OutputTokens: 1}}, nil, 0},
		{"exact max", "UU", []*ai.Usage{{InputTokens: math.MaxInt64 - 1, OutputTokens: math.MaxInt64 - 1}, {InputTokens: 1, OutputTokens: 1}}, &ai.Usage{InputTokens: math.MaxInt64, OutputTokens: math.MaxInt64}, 0},
		{"input overflow before nil", "TTT", []*ai.Usage{{InputTokens: math.MaxInt64}, {InputTokens: 1}, nil}, nil, 2},
		{"output overflow before nil", "UUU", []*ai.Usage{{OutputTokens: math.MaxInt64}, {OutputTokens: 1}, nil}, nil, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var steps []SequenceStep
			for i, kind := range tc.kinds {
				execute := func(context.Context, ai.Request) (ai.Result, error) {
					if calls != i {
						t.Fatal("usage step order changed")
					}
					calls++
					out := result()
					out.Usage = tc.usage[i]
					return out, nil
				}
				step := sequenceTextStep(t, providerFunc(execute))
				if kind == 'U' {
					step = sequenceToolStep(t, roundTripperFunc(func(ctx context.Context, r ai.Request, _ *tool.Authority) (ai.Result, error) {
						if r.MaxOutputTokens != 64 {
							t.Fatal("tool cap changed")
						}
						return execute(ctx, r)
					}), testAuthority(t))
				}
				steps = append(steps, step)
			}
			s := newSequence(t, steps...)
			done := s.Done()
			got, err := s.Execute(context.Background())
			if tc.failAt != 0 {
				if got != (SequenceResult{}) || err != ai.ErrMalformedResponse || calls != tc.failAt {
					t.Fatal("overflow classification, partial result, or later work")
				}
				sequenceTerminal(t, s, done, StateFailed)
				return
			}
			if err != nil || calls != len(steps) || got.CompletedSteps != calls || !reflect.DeepEqual(got.AggregateUsage, tc.want) || !reflect.DeepEqual(got.Final.Usage, tc.usage[len(steps)-1]) {
				t.Fatal("aggregate/final usage semantics changed")
			}
			if got.AggregateUsage != nil {
				if got.AggregateUsage == got.Final.Usage {
					t.Fatal("aggregate aliases final")
				}
				for _, usage := range tc.usage {
					if usage == got.AggregateUsage {
						t.Fatal("aggregate aliases child")
					}
				}
				finalBefore := *got.Final.Usage
				got.AggregateUsage.OutputTokens = 0
				if *got.Final.Usage != finalBefore {
					t.Fatal("aggregate mutation changed final usage")
				}
			}
			sequenceTerminal(t, s, done, StateSucceeded)
		})
	}
}

func TestSequencePreviousPanicAndCancellationAfterAccounting(t *testing.T) {
	for _, panicChild := range []bool{false, true} {
		first := sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return result(), nil }))
		marker := &struct{}{}
		calls := 0
		var s *Sequence
		next, err := NewTextSequenceStepFromPrevious(providerFunc(func(ctx context.Context, r ai.Request) (ai.Result, error) {
			calls++
			if r.Text != result().Text {
				t.Fatal("wrong predecessor")
			}
			if panicChild {
				panic(marker)
			}
			s.Cancel()
			if ctx.Err() != context.Canceled {
				t.Fatal("previous child cancellation lost")
			}
			return result(), nil
		}), request().Model, 64, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		s = newSequence(t, first, next, first)
		done := s.Done()
		if panicChild {
			func() {
				defer func() {
					if recover() != marker {
						t.Error("panic replaced")
					}
				}()
				_, _ = s.Execute(context.Background())
				t.Error("panic returned")
			}()
			sequenceTerminal(t, s, done, StateFailed)
		} else {
			got, err := s.Execute(context.Background())
			if got != (SequenceResult{}) || err != context.Canceled {
				t.Fatal("cancellation returned partial accounting")
			}
			sequenceTerminal(t, s, done, StateCanceled)
		}
		if calls != 1 {
			t.Fatal("previous child repeated")
		}
	}
}
