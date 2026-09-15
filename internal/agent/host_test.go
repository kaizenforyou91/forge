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
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
	"github.com/kaizenforyou91/forge/pkg/app"
)

type hostTestModule struct {
	start func(*app.App) error
	stop  func(*app.App) error
}

func (*hostTestModule) Name() string            { return "host-test-barrier" }
func (*hostTestModule) Register(*app.App) error { return nil }
func (m *hostTestModule) Start(a *app.App) error {
	if m.start != nil {
		return m.start(a)
	}
	return nil
}
func (m *hostTestModule) Stop(a *app.App) error {
	if m.stop != nil {
		return m.stop(a)
	}
	return nil
}

func registeredHost(t *testing.T) (*RunHost, *app.App) {
	t.Helper()
	h, a := NewRunHost(), app.New()
	if err := a.Add(h); err != nil {
		t.Fatal(err)
	}
	return h, a
}

func runningHost(t *testing.T) (*RunHost, *app.App) {
	t.Helper()
	h, a := registeredHost(t)
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := a.Stop(); err != nil {
			t.Error(err)
		}
	})
	return h, a
}

func hostEmpty(t *testing.T, h *RunHost, stopped bool) {
	t.Helper()
	h.core.mu.Lock()
	defer h.core.mu.Unlock()
	if len(h.core.active) != 0 {
		t.Fatal("active execution retained")
	}
	if stopped && (h.core.phase != hostStopped || h.core.appCtx != nil || h.core.stopDone != nil) {
		t.Fatal("stopped host retained lifecycle context or drain")
	}
}

func hostReject(t *testing.T, h *RunHost, r *Run, want error) {
	t.Helper()
	got, err := h.Execute(context.Background(), r)
	if got != (ai.Result{}) || err != want || r.State() != StateReady {
		t.Fatal("host rejection consumed Run or returned wrong classification")
	}
}

func TestHostConstructionAndBinding(t *testing.T) {
	a := app.New()
	for _, h := range []*RunHost{nil, {}, {core: &hostState{}}} {
		if h.Name() != "agent-run-host" {
			t.Fatal("unsafe module name")
		}
		if err := h.Register(a); err != ErrInvalidHost {
			t.Fatal("invalid host registered")
		}
		if err := h.Start(a); err != ErrInvalidHost {
			t.Fatal("invalid host started")
		}
		if err := h.Stop(a); err != ErrInvalidHost {
			t.Fatal("invalid host stopped")
		}
		got, err := h.Execute(context.Background(), nil)
		if got != (ai.Result{}) || err != ErrInvalidHost {
			t.Fatal("invalid host executed")
		}
	}
	var typedNil *RunHost
	if err := a.Add(typedNil); err != ErrInvalidHost || len(a.Modules()) != 0 {
		t.Fatal("typed-nil module escaped registration")
	}
	h := NewRunHost()
	if h.core.phase != hostReady {
		t.Fatal("constructor not ready")
	}
	if h.Register(nil) != ErrInvalidHost || h.Register(&app.App{}) != ErrInvalidHost {
		t.Fatal("invalid App accepted")
	}
	if h.Start(a) != ErrInvalidHost || h.Stop(a) != ErrInvalidHost {
		t.Fatal("unbound lifecycle accepted")
	}
	if _, err := h.Execute(context.Background(), nil); err != ErrInvalidHost {
		t.Fatal("unbound execution accepted")
	}
	if err := a.Add(h); err != nil {
		t.Fatal(err)
	}
	if len(a.Modules()) != 1 || a.Modules()[0] != h || h.core.app != a {
		t.Fatal("registration ownership changed")
	}
	if a.Add(h) != ErrInvalidHost || h.Register(app.New()) != ErrInvalidHost {
		t.Fatal("ambiguous registration accepted")
	}
	for _, wrong := range []*app.App{nil, app.New()} {
		if h.Start(wrong) != ErrInvalidHost || h.Stop(wrong) != ErrInvalidHost {
			t.Fatal("cross-App lifecycle accepted")
		}
	}
	var calls int
	r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { calls++; return result(), nil }))
	hostReject(t, h, r, ErrHostNotRunning)
	if calls != 0 {
		t.Fatal("registration executed provider")
	}
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("Start executed provider")
	}
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	hostReject(t, h, r, ErrHostNotRunning)
	if calls != 0 {
		t.Fatal("Stop executed provider")
	}
	hostEmpty(t, h, true)
	r.Cancel()
}

func TestHostPartialStartupAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		h, a := registeredHost(t)
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return result(), nil }))
		if err := a.Add(&hostTestModule{start: func(a *app.App) error {
			if a.Started() {
				t.Fatal("App running before all module starts")
			}
			hostReject(t, h, r, ErrHostNotRunning)
			if fail {
				return errors.New("startup failure")
			}
			return nil
		}}); err != nil {
			t.Fatal(err)
		}
		err := a.Start()
		if fail {
			if err == nil {
				t.Fatal("startup failure lost")
			}
			hostEmpty(t, h, true)
			hostReject(t, h, r, ErrHostNotRunning)
		} else {
			if err != nil {
				t.Fatal(err)
			}
			got, err := h.Execute(context.Background(), r)
			if err != nil || !reflect.DeepEqual(got, result()) {
				t.Fatal("fully Running host rejected work")
			}
			if err := a.Stop(); err != nil {
				t.Fatal(err)
			}
		}
		r.Cancel()
	}
}

func TestHostExplicitTextAndToolExecution(t *testing.T) {
	h, a := runningHost(t)
	var calls atomic.Int32
	textRun := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { calls.Add(1); return result(), nil }))
	toolRun := newToolRun(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
		calls.Add(1)
		got := result()
		got.Usage.OutputTokens = 96
		return got, nil
	}))
	if calls.Load() != 0 {
		t.Fatal("construction executed work")
	}
	if got, err := h.Execute(nil, textRun); got != (ai.Result{}) || err != ai.ErrInvalidRequest || textRun.State() != StateReady {
		t.Fatal("nil context consumed Run")
	}
	for i, r := range []*Run{textRun, toolRun} {
		done := r.Done()
		got, err := h.Execute(context.Background(), r)
		if err != nil || got.Text != result().Text {
			t.Fatal("explicit execution failed")
		}
		if i == 1 && got.Usage.OutputTokens != 96 {
			t.Fatal("aggregate 96 > 64 reinterpreted")
		}
		terminal(t, r, done, StateSucceeded)
		hostEmpty(t, h, false)
	}
	if calls.Load() != 2 {
		t.Fatal("unexpected delegation count")
	}
	for _, invalid := range []*Run{nil, {}} {
		if got, err := h.Execute(context.Background(), invalid); got != (ai.Result{}) || err != ErrInvalidRun {
			t.Fatal("invalid Run escaped")
		}
		hostEmpty(t, h, false)
	}
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	hostEmpty(t, h, true)
}

func TestHostPreCanceledCallerConsumesRun(t *testing.T) {
	h, _ := runningHost(t)
	for _, expired := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		want := context.Canceled
		if expired {
			ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			want = context.DeadlineExceeded
		}
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("pre-canceled caller delegated work")
			return ai.Result{}, nil
		}))
		done := r.Done()
		got, err := h.Execute(ctx, r)
		cancel()
		if got != (ai.Result{}) || err != want {
			t.Fatal("caller cancellation identity lost")
		}
		terminal(t, r, done, StateCanceled)
		hostEmpty(t, h, false)
	}
}

func TestHostCallerValuesDeadlineAndAppCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), struct{}{}, "caller-value"), time.Second/2)
		defer cancel()
		r := newRun(t, providerFunc(func(operation context.Context, _ ai.Request) (ai.Result, error) {
			deadline, ok := operation.Deadline()
			if operation.Value(struct{}{}) != "caller-value" || !ok || time.Until(deadline) != time.Second/2 {
				t.Fatal("caller context replaced")
			}
			<-operation.Done()
			a.Cancel() // Later App cancellation must not replace the earlier deadline.
			return ai.Result{}, operation.Err()
		}))
		done := r.Done()
		got, err := h.Execute(ctx, r)
		if got != (ai.Result{}) || err != context.DeadlineExceeded {
			t.Fatal("deadline identity replaced")
		}
		terminal(t, r, done, StateCanceled)
		if !a.Started() {
			t.Fatal("App.Cancel automatically stopped App")
		}
		ready := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("canceled App delegated")
			return ai.Result{}, nil
		}))
		hostReject(t, h, ready, ErrHostNotRunning)
		ready.Cancel()
	})
}

func TestHostAppCancelWithoutStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		entered := make(chan context.Context, 1)
		release := make(chan struct{})
		r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			entered <- ctx
			<-release
			return result(), nil
		}))
		done := r.Done()
		returned := make(chan error, 1)
		go func() { _, err := h.Execute(context.Background(), r); returned <- err }()
		ctx := <-entered
		a.Cancel()
		synctest.Wait()
		if ctx.Err() != context.Canceled || r.State() != StateRunning || !a.Started() {
			t.Fatal("App.Cancel bridge failed")
		}
		openDone(t, r, done)
		close(release)
		if err := <-returned; err != context.Canceled {
			t.Fatal("success after app cancellation escaped")
		}
		terminal(t, r, done, StateCanceled)
		hostEmpty(t, h, false)
	})
}

func TestHostShutdownCancelsBeforeReverseModuleStop(t *testing.T) {
	for _, toolPath := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			h, a := registeredHost(t)
			moduleEntered, moduleRelease := make(chan struct{}), make(chan struct{})
			if err := a.Add(&hostTestModule{stop: func(*app.App) error { close(moduleEntered); <-moduleRelease; return nil }}); err != nil {
				t.Fatal(err)
			}
			if err := a.Start(); err != nil {
				t.Fatal(err)
			}
			entered := make(chan context.Context, 1)
			release := make(chan struct{})
			work := func(ctx context.Context) (ai.Result, error) { entered <- ctx; <-release; return ai.Result{}, ctx.Err() }
			var r *Run
			if toolPath {
				r = newToolRun(t, roundTripperFunc(func(ctx context.Context, _ ai.Request, _ *tool.Authority) (ai.Result, error) { return work(ctx) }))
			} else {
				r = newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) { return work(ctx) }))
			}
			done := r.Done()
			returned, stopped := make(chan error, 1), make(chan error, 1)
			go func() { _, err := h.Execute(context.Background(), r); returned <- err }()
			ctx := <-entered
			go func() { stopped <- a.Stop() }()
			<-moduleEntered
			synctest.Wait()
			if ctx.Err() != context.Canceled || r.State() != StateRunning {
				t.Fatal("early App cancellation missing")
			}
			openDone(t, r, done)
			ready := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
				t.Fatal("stopping App delegated")
				return ai.Result{}, nil
			}))
			hostReject(t, h, ready, ErrHostNotRunning)
			ready.Cancel()
			close(moduleRelease)
			synctest.Wait()
			select {
			case <-stopped:
				t.Fatal("Stop returned with active Execute")
			default:
			}
			if r.State() != StateRunning {
				t.Fatal("Run terminal before delegated return")
			}
			openDone(t, r, done)
			close(release)
			if err := <-returned; err != context.Canceled {
				t.Fatal("shutdown cancellation lost")
			}
			if err := <-stopped; err != nil {
				t.Fatal(err)
			}
			terminal(t, r, done, StateCanceled)
			hostEmpty(t, h, true)
		})
	}
}

// A test-only parent pauses context linkage after Host admission but before the
// Run claim. It allows Stop to cancel a still-Ready registered Run deterministically.
type hostGatedContext struct {
	context.Context
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *hostGatedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.entered); <-c.release })
	return c.Context.Done()
}

func TestHostStopBeforeRegisteredRunClaim(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("pre-claim Stop delegated work")
			return ai.Result{}, nil
		}))
		done := r.Done()
		ctx := &hostGatedContext{Context: context.Background(), entered: make(chan struct{}), release: make(chan struct{})}
		returned, stopped := make(chan error, 1), make(chan error, 1)
		go func() { _, err := h.Execute(ctx, r); returned <- err }()
		<-ctx.entered
		h.core.mu.Lock()
		active := len(h.core.active)
		h.core.mu.Unlock()
		if active != 1 || r.State() != StateReady {
			t.Fatal("test did not pause registered Ready Run")
		}
		go func() { stopped <- a.Stop() }()
		synctest.Wait()
		if r.State() != StateCanceled {
			t.Fatal("Stop did not cancel Ready registration")
		}
		select {
		case <-stopped:
			t.Fatal("Stop failed to wait for registered call")
		default:
		}
		close(ctx.release)
		if err := <-returned; err != ErrRunConsumed {
			t.Fatal("pre-start canceled Run was not consumed")
		}
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		terminal(t, r, done, StateCanceled)
		hostEmpty(t, h, true)
	})
}

func TestHostAppCancelBeforeContextLink(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("App cancellation before context link escaped to provider")
			return ai.Result{}, nil
		}))
		done := r.Done()
		ctx := &hostGatedContext{Context: context.Background(), entered: make(chan struct{}), release: make(chan struct{})}
		returned := make(chan error, 1)
		go func() { _, err := h.Execute(ctx, r); returned <- err }()
		<-ctx.entered
		a.Cancel()
		close(ctx.release)
		if err := <-returned; err != context.Canceled {
			t.Fatal("already-canceled App context was not checked before Run delegation")
		}
		terminal(t, r, done, StateCanceled)
		hostEmpty(t, h, false)
	})
}

func TestHostDuplicateRunEntriesDrainWinner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		var calls atomic.Int32
		entered := make(chan context.Context, 1)
		release := make(chan struct{})
		r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			calls.Add(1)
			entered <- ctx
			<-release
			return ai.Result{}, ctx.Err()
		}))
		copyOfRun := *r
		done := r.Done()
		start := make(chan struct{})
		returned := make(chan error, 32)
		for i := range 32 {
			go func() {
				<-start
				target := r
				if i%2 == 0 {
					target = &copyOfRun
				}
				_, err := h.Execute(context.Background(), target)
				returned <- err
			}()
		}
		close(start)
		ctx := <-entered
		for range 31 {
			if err := <-returned; err != ErrRunConsumed {
				t.Fatal("duplicate claim escaped")
			}
		}
		h.core.mu.Lock()
		active := len(h.core.active)
		h.core.mu.Unlock()
		if active != 1 || calls.Load() != 1 {
			t.Fatal("losing duplicate untracked winner")
		}
		stopped := make(chan error, 1)
		go func() { stopped <- a.Stop() }()
		synctest.Wait()
		if ctx.Err() != context.Canceled {
			t.Fatal("winning entry was not canceled")
		}
		openDone(t, r, done)
		select {
		case <-stopped:
			t.Fatal("winner not drained")
		default:
		}
		close(release)
		if err := <-returned; err != context.Canceled {
			t.Fatal("winner cancellation lost")
		}
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		terminal(t, r, done, StateCanceled)
		hostEmpty(t, h, true)
		if calls.Load() != 1 {
			t.Fatal("duplicate underlying operation")
		}
	})
}

func TestHostConcurrentStopAdmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		start := make(chan struct{})
		returned := make(chan struct{}, 32)
		for range 32 {
			go func() {
				r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
					<-ctx.Done()
					return ai.Result{}, ctx.Err()
				}))
				<-start
				got, err := h.Execute(context.Background(), r)
				if got != (ai.Result{}) {
					t.Error("Stop race returned data")
				}
				switch err {
				case ErrHostNotRunning:
					if r.State() != StateReady {
						t.Error("rejected Run consumed")
					}
					r.Cancel()
				case ErrRunConsumed, context.Canceled:
					if r.State() != StateCanceled {
						t.Error("admitted Run not canceled")
					}
				default:
					t.Error("unexpected Stop race outcome", err)
				}
				returned <- struct{}{}
			}()
		}
		stopped := make(chan error, 1)
		go func() { <-start; stopped <- a.Stop() }()
		close(start)
		for range 32 {
			<-returned
		}
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		hostEmpty(t, h, true)
		ready := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Fatal("post-stop work")
			return ai.Result{}, nil
		}))
		hostReject(t, h, ready, ErrHostNotRunning)
		ready.Cancel()
	})
}

func TestHostRestartFreshContext(t *testing.T) {
	h, a := runningHost(t)
	oldCtx := a.Context()
	oldRun := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) { return result(), nil }))
	if _, err := h.Execute(context.Background(), oldRun); err != nil {
		t.Fatal(err)
	}
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	hostEmpty(t, h, true)
	ready := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
		if ctx.Err() != nil {
			t.Fatal("restart retained old cancellation")
		}
		return result(), nil
	}))
	hostReject(t, h, ready, ErrHostNotRunning)
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	if a.Context() == oldCtx || oldCtx.Err() != context.Canceled || h.core.appCtx != a.Context() {
		t.Fatal("restart context not fresh")
	}
	if _, err := h.Execute(context.Background(), oldRun); err != ErrRunConsumed {
		t.Fatal("restart resumed old Run")
	}
	if got, err := h.Execute(context.Background(), ready); err != nil || !reflect.DeepEqual(got, result()) {
		t.Fatal("new generation failed")
	}
	hostEmpty(t, h, false)
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	hostEmpty(t, h, true)
}

func TestHostFailureAndPanicCleanup(t *testing.T) {
	for _, panicWork := range []bool{false, true} {
		h, a := runningHost(t)
		r := newRun(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			if panicWork {
				panic("test unwind")
			}
			return result(), errors.Join(context.Canceled, ai.ErrProvider)
		}))
		if panicWork {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("panic not propagated")
					}
				}()
				_, _ = h.Execute(context.Background(), r)
			}()
		} else {
			got, err := h.Execute(context.Background(), r)
			if got != (ai.Result{}) || !errors.Is(err, context.Canceled) || !errors.Is(err, ai.ErrProvider) || r.State() != StateFailed {
				t.Fatal("Host reclassified Run failure")
			}
		}
		hostEmpty(t, h, false)
		if err := a.Stop(); err != nil {
			t.Fatal("operation failure became Stop error", err)
		}
		hostEmpty(t, h, true)
	}
}

func TestHostConcurrentStopsAndCopies(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		copyOfHost := *h
		entered := make(chan struct{})
		release := make(chan struct{})
		r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			close(entered)
			<-release
			return ai.Result{}, ctx.Err()
		}))
		returned := make(chan error, 1)
		go func() { _, err := copyOfHost.Execute(context.Background(), r); returned <- err }()
		<-entered
		stopped := make(chan error, 2)
		go func() { stopped <- h.Stop(a) }()
		go func() { stopped <- copyOfHost.Stop(a) }()
		synctest.Wait()
		select {
		case <-stopped:
			t.Fatal("concurrent Stop returned early")
		default:
		}
		close(release)
		if err := <-returned; err != context.Canceled {
			t.Fatal(err)
		}
		for range 2 {
			if err := <-stopped; err != nil {
				t.Fatal(err)
			}
		}
		hostEmpty(t, h, true)
		if err := a.Stop(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestHostRedaction(t *testing.T) {
	h, _ := runningHost(t)
	if h.Name() != "agent-run-host" || h.String() != "agent run host (data redacted)" || h.GoString() != h.String() || ErrInvalidHost.Error() != "agent: invalid host" || ErrHostNotRunning.Error() != "agent: host not running" {
		t.Fatal("host vocabulary changed")
	}
	check := func() {
		t.Helper()
		for _, output := range []string{h.String(), h.GoString(), fmt.Sprintf("%v", h), fmt.Sprintf("%+v", *h), fmt.Sprintf("%#v", h), fmt.Sprintf("%#v", *h), h.Name(), ErrInvalidHost.Error(), ErrHostNotRunning.Error()} {
			for _, canary := range []string{request().Text, request().Model, result().Text, "private_authority_canary", "private_error_canary", "private_credential_canary"} {
				if strings.Contains(output, canary) {
					t.Fatal("host leaked payload")
				}
			}
		}
	}
	for _, fail := range []bool{false, true} {
		r := newToolRun(t, roundTripperFunc(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
			check()
			if fail {
				return result(), errors.New("private_error_canary private_credential_canary private_authority_canary")
			}
			return result(), nil
		}))
		check()
		_, err := h.Execute(context.Background(), r)
		if fail && err != ai.ErrProvider {
			t.Fatal("host exposed raw error")
		}
		check()
		hostEmpty(t, h, false)
	}
}

func hostActiveCount(t *testing.T, h *RunHost, want int) *hostExecution {
	t.Helper()
	h.core.mu.Lock()
	defer h.core.mu.Unlock()
	if len(h.core.active) != want {
		t.Fatalf("active calls = %d, want %d", len(h.core.active), want)
	}
	for entry := range h.core.active {
		return entry
	}
	return nil
}

func TestHostSequenceAdmissionAndZeroWork(t *testing.T) {
	var calls int
	s := newSequence(t, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		calls++
		return result(), nil
	})))
	check := func(h *RunHost, ctx context.Context, want error) {
		t.Helper()
		got, err := h.ExecuteSequence(ctx, s)
		if got != (SequenceResult{}) || err != want || s.State() != StateReady || calls != 0 {
			t.Fatal("rejection consumed Sequence or performed work")
		}
	}
	for _, h := range []*RunHost{nil, {}, {core: &hostState{}}, NewRunHost()} {
		check(h, context.Background(), ErrInvalidHost)
		check(h, nil, ai.ErrInvalidRequest)
	}
	h, a := registeredHost(t)
	check(h, context.Background(), ErrHostNotRunning)
	if err := a.Add(&hostTestModule{start: func(*app.App) error {
		check(h, context.Background(), ErrHostNotRunning)
		return nil
	}}); err != nil {
		t.Fatal(err)
	}
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || s.State() != StateReady {
		t.Fatal("Register/Start executed Sequence")
	}
	check(h, nil, ai.ErrInvalidRequest)
	for _, invalid := range []*Sequence{nil, {}, {core: &sequenceState{}}} {
		got, err := h.ExecuteSequence(context.Background(), invalid)
		if got != (SequenceResult{}) || err != ErrInvalidSequence {
			t.Fatal("Sequence validation ownership changed")
		}
		hostEmpty(t, h, false)
	}
	// A live but different context is not the captured application generation.
	h.core.mu.Lock()
	actual := h.core.appCtx
	h.core.appCtx = context.Background()
	h.core.mu.Unlock()
	check(h, context.Background(), ErrHostNotRunning)
	h.core.mu.Lock()
	h.core.appCtx = actual
	h.core.mu.Unlock()
	got, err := h.ExecuteSequence(context.Background(), s)
	if err != nil || got.CompletedSteps != 1 || !reflect.DeepEqual(got.Final, result()) || *got.AggregateUsage != *result().Usage {
		t.Fatal("one-step result changed")
	}
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	hostEmpty(t, h, true)
}

func TestHostSequenceSingleAdmissionAndResults(t *testing.T) {
	for _, previous := range []bool{false, true} {
		h, _ := runningHost(t)
		var identity *hostExecution
		var calls int
		work := func(req ai.Request) (ai.Result, error) {
			entry := hostActiveCount(t, h, 1)
			if identity == nil {
				identity = entry
			} else if identity != entry {
				t.Fatal("per-child host admission")
			}
			want := request().Text
			if previous && calls > 0 {
				want = result().Text
			}
			if req.Text != want {
				t.Fatal("host changed handoff/literal")
			}
			calls++
			got := result()
			if calls%2 == 0 {
				got.Usage.OutputTokens = 96
			}
			return got, nil
		}
		p := providerFunc(func(_ context.Context, r ai.Request) (ai.Result, error) { return work(r) })
		rt := roundTripperFunc(func(_ context.Context, r ai.Request, _ *tool.Authority) (ai.Result, error) { return work(r) })
		authority := testAuthority(t)
		steps := make([]SequenceStep, 8)
		for i := range steps {
			var err error
			if previous && i > 0 {
				if i%2 == 0 {
					steps[i], err = NewTextSequenceStepFromPrevious(p, request().Model, 64, time.Second)
				} else {
					steps[i], err = NewAuthorizedToolSequenceStepFromPrevious(rt, request().Model, 64, authority, time.Second)
				}
			} else if i%2 == 0 {
				steps[i] = sequenceTextStep(t, p)
			} else {
				steps[i] = sequenceToolStep(t, rt, authority)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		s := newSequence(t, steps...)
		got, err := h.ExecuteSequence(context.Background(), s)
		if err != nil || calls != 8 || got.CompletedSteps != 8 || got.Final.Text != result().Text ||
			*got.Final.Usage != (ai.Usage{InputTokens: 1, OutputTokens: 96}) ||
			*got.AggregateUsage != (ai.Usage{InputTokens: 8, OutputTokens: 392}) || got.Final.Usage == got.AggregateUsage {
			t.Fatal("host rewrote result or aggregate usage")
		}
		hostEmpty(t, h, false)
		if identity.cancel != nil {
			t.Fatal("released entry retained cancellation owner")
		}
	}
}

func TestHostSequenceCancellationAndDrain(t *testing.T) {
	for _, source := range []string{"caller", "app", "stop", "concurrent-host-stop"} {
		for _, toolPath := range []bool{false, true} {
			t.Run(fmt.Sprint(source, "/tool=", toolPath), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					h, a := runningHost(t)
					entered := make(chan context.Context, 1)
					release := make(chan struct{})
					work := func(ctx context.Context) (ai.Result, error) { entered <- ctx; <-release; return result(), nil }
					step := sequenceTextStep(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) { return work(ctx) }))
					if toolPath {
						step = sequenceToolStep(t, roundTripperFunc(func(ctx context.Context, _ ai.Request, _ *tool.Authority) (ai.Result, error) { return work(ctx) }), testAuthority(t))
					}
					s := newSequence(t, step, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
						t.Error("later child executed after cancellation")
						return result(), nil
					})))
					done := s.Done()
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					returned, stopped := make(chan error, 1), make(chan error, 2)
					go func() {
						got, err := h.ExecuteSequence(ctx, s)
						if got != (SequenceResult{}) {
							t.Error("partial result escaped")
						}
						returned <- err
					}()
					childCtx := <-entered
					entry := hostActiveCount(t, h, 1)
					stopCount := 0
					switch source {
					case "caller":
						cancel()
					case "app":
						a.Cancel()
					case "stop":
						stopCount = 1
						go func() { stopped <- a.Stop() }()
					case "concurrent-host-stop":
						stopCount = 2
						go func() { stopped <- h.Stop(a) }()
						go func() { stopped <- h.Stop(a) }()
					}
					synctest.Wait()
					if childCtx.Err() != context.Canceled || s.State() != StateRunning || hostActiveCount(t, h, 1) != entry {
						t.Fatal("cancellation detached active Sequence")
					}
					sequenceOpen(t, s, done)
					select {
					case <-stopped:
						t.Fatal("Stop did not drain")
					default:
					}
					select {
					case <-returned:
						t.Fatal("host detached non-cooperative work")
					default:
					}
					if source != "caller" {
						if _, err := h.ExecuteSequence(context.Background(), nil); err != ErrHostNotRunning {
							t.Fatal("admission remained open")
						}
					}
					close(release)
					if err := <-returned; err != context.Canceled {
						t.Fatal(err)
					}
					for range stopCount {
						if err := <-stopped; err != nil {
							t.Fatal(err)
						}
					}
					sequenceTerminal(t, s, done, StateCanceled)
					hostEmpty(t, h, stopCount > 0)
				})
			})
		}
	}
}

func TestHostSequenceInterStepOwnership(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		s := newSequence(t, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			return result(), nil
		})), sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			t.Error("canceled gap constructed/executed next child")
			return result(), nil
		})))
		// White-box transition fixture: use the real host admission and the real
		// Sequence cancellation/nextRun/complete paths, with no timing hook.
		linked, release, err := h.admit(context.Background(), s.Cancel)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(linked, s.core.overallTimeout)
		defer cancel()
		s.core.mu.Lock()
		s.core.state, s.core.cancel = StateRunning, cancel
		s.core.mu.Unlock()
		entry := hostActiveCount(t, h, 1)
		first, err := s.core.nextRun(ctx, 0, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := first.Execute(ctx); err != nil {
			t.Fatal(err)
		}
		s.core.mu.Lock()
		s.core.active = nil
		s.core.mu.Unlock()
		stopped := make(chan error, 1)
		go func() { stopped <- h.Stop(a) }()
		synctest.Wait()
		s.core.mu.Lock()
		gapCanceled := s.core.canceled && s.core.active == nil
		s.core.mu.Unlock()
		if !gapCanceled || s.State() != StateRunning || hostActiveCount(t, h, 1) != entry {
			t.Fatal("gap was not owned/canceled")
		}
		sequenceOpen(t, s, s.Done())
		if child, err := s.core.nextRun(ctx, 1, result().Text); child != nil || err != context.Canceled {
			t.Fatal("next child constructed after Stop")
		}
		if got, err := s.core.complete(ctx, SequenceResult{}, ctx.Err()); got != (SequenceResult{}) || err != context.Canceled {
			t.Fatal("gap cancellation changed")
		}
		// Even terminal Done does not release the host call; its return/unwind does.
		select {
		case <-stopped:
			t.Fatal("Stop used Done instead of call lifetime")
		default:
		}
		release()
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		hostEmpty(t, h, true)
	})
}

func TestHostSequenceDuplicateClaimsAndMixedOwners(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		seqEntered, runEntered := make(chan context.Context, 1), make(chan context.Context, 1)
		seqRelease, runRelease := make(chan struct{}), make(chan struct{})
		s := newSequence(t, sequenceTextStep(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			seqEntered <- ctx
			<-seqRelease
			return ai.Result{}, ctx.Err()
		})))
		copyOfSequence := *s
		start, returned := make(chan struct{}), make(chan error, 16)
		for i := range 16 {
			go func() {
				<-start
				target := s
				if i%2 == 0 {
					target = &copyOfSequence
				}
				_, err := h.ExecuteSequence(context.Background(), target)
				returned <- err
			}()
		}
		close(start)
		seqCtx := <-seqEntered
		for range 15 {
			if err := <-returned; err != ErrSequenceConsumed {
				t.Fatal("duplicate claim escaped", err)
			}
		}
		winner := hostActiveCount(t, h, 1)
		r := newRun(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			runEntered <- ctx
			<-runRelease
			return ai.Result{}, ctx.Err()
		}))
		runReturned := make(chan error, 1)
		go func() { _, err := h.Execute(context.Background(), r); runReturned <- err }()
		runCtx := <-runEntered
		hostActiveCount(t, h, 2)
		stopped := make(chan error, 1)
		go func() { stopped <- a.Stop() }()
		synctest.Wait()
		if seqCtx.Err() != context.Canceled || runCtx.Err() != context.Canceled {
			t.Fatal("mixed owners not canceled")
		}
		close(runRelease)
		if err := <-runReturned; err != context.Canceled {
			t.Fatal(err)
		}
		if hostActiveCount(t, h, 1) != winner {
			t.Fatal("other call untracked winning Sequence")
		}
		select {
		case <-stopped:
			t.Fatal("mixed Stop returned early")
		default:
		}
		close(seqRelease)
		if err := <-returned; err != context.Canceled {
			t.Fatal(err)
		}
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		hostEmpty(t, h, true)
	})
}

func TestHostSequenceStopAdmissionRace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, a := runningHost(t)
		start, returned := make(chan struct{}), make(chan struct{}, 32)
		for range 32 {
			go func() {
				var calls int
				s := newSequence(t, sequenceTextStep(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
					calls++
					<-ctx.Done()
					return ai.Result{}, ctx.Err()
				})))
				<-start
				got, err := h.ExecuteSequence(context.Background(), s)
				if got != (SequenceResult{}) {
					t.Error("race returned partial data")
				}
				switch err {
				case ErrHostNotRunning:
					if calls != 0 || s.State() != StateReady {
						t.Error("rejected Sequence consumed")
					}
					s.Cancel()
				case ErrSequenceConsumed, context.Canceled:
					if s.State() != StateCanceled || calls > 1 {
						t.Error("admitted Sequence not owned")
					}
				default:
					t.Error("unexpected admission outcome", err)
				}
				returned <- struct{}{}
			}()
		}
		stopped := make(chan error, 1)
		go func() { <-start; stopped <- a.Stop() }()
		close(start)
		for range 32 {
			<-returned
		}
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		hostEmpty(t, h, true)
	})
}

func TestHostSequenceDeadlinesAndPreCancel(t *testing.T) {
	for _, callerLimit := range []time.Duration{time.Millisecond * 100, time.Second} {
		synctest.Test(t, func(t *testing.T) {
			h, _ := runningHost(t)
			ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), struct{}{}, "caller"), callerLimit)
			defer cancel()
			want := min(callerLimit, time.Millisecond*200)
			step := sequenceTextStep(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) != want || ctx.Value(struct{}{}) != "caller" {
					t.Fatal("host changed timeout/context")
				}
				<-ctx.Done()
				return ai.Result{}, ctx.Err()
			}))
			s, err := NewSequence([]SequenceStep{step}, time.Millisecond*200)
			if err != nil {
				t.Fatal(err)
			}
			got, err := h.ExecuteSequence(ctx, s)
			if got != (SequenceResult{}) || err != context.DeadlineExceeded {
				t.Fatal("deadline lost")
			}
			sequenceTerminal(t, s, s.Done(), StateCanceled)
			hostEmpty(t, h, false)
		})
	}
	h, _ := runningHost(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := newSequence(t, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
		t.Fatal("pre-canceled work")
		return result(), nil
	})))
	if got, err := h.ExecuteSequence(ctx, s); got != (SequenceResult{}) || err != context.Canceled {
		t.Fatal("pre-cancel changed")
	}
	sequenceTerminal(t, s, s.Done(), StateCanceled)
}

func TestHostSequenceFailurePanicAndRestart(t *testing.T) {
	h, a := runningHost(t)
	panicValue := &struct{ name string }{"private_panic_canary"}
	for _, panics := range []bool{false, true} {
		s := newSequence(t, sequenceTextStep(t, providerFunc(func(context.Context, ai.Request) (ai.Result, error) {
			if panics {
				panic(panicValue)
			}
			return result(), errors.Join(context.Canceled, ai.ErrProvider)
		})))
		if panics {
			func() {
				defer func() {
					if recover() != panicValue {
						t.Error("panic identity changed")
					}
				}()
				_, _ = h.ExecuteSequence(context.Background(), s)
			}()
		} else {
			got, err := h.ExecuteSequence(context.Background(), s)
			if got != (SequenceResult{}) || !errors.Is(err, ai.ErrProvider) || !errors.Is(err, context.Canceled) {
				t.Fatal("failure reclassified")
			}
		}
		sequenceTerminal(t, s, s.Done(), StateFailed)
		hostEmpty(t, h, false)
	}
	makeSequence := func() *Sequence {
		return newSequence(t, sequenceTextStep(t, providerFunc(func(ctx context.Context, _ ai.Request) (ai.Result, error) {
			if ctx.Err() != nil {
				t.Fatal("stale generation context")
			}
			return result(), nil
		})))
	}
	old := makeSequence()
	if _, err := h.ExecuteSequence(context.Background(), old); err != nil {
		t.Fatal("host unusable after panic", err)
	}
	oldCtx := a.Context()
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	hostEmpty(t, h, true)
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	if a.Context() == oldCtx || oldCtx.Err() != context.Canceled || h.core.appCtx != a.Context() {
		t.Fatal("generation not refreshed")
	}
	if _, err := h.ExecuteSequence(context.Background(), old); err != ErrSequenceConsumed {
		t.Fatal("restart reused old Sequence")
	}
	if _, err := h.ExecuteSequence(context.Background(), makeSequence()); err != nil {
		t.Fatal(err)
	}
	hostEmpty(t, h, false)
}
