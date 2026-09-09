package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func fccNew(t *testing.T, definition tool.Definition, handler tool.Handler) *FunctionCallCoordinator {
	t.Helper()
	executor, err := tool.NewExecutor([]tool.Binding{{Definition: definition, Handler: handler}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFunctionCallCoordinator(executor)
	if err != nil || c == nil {
		t.Fatalf("constructor failed: %v", err)
	}
	return c
}

func fccDecision(t *testing.T, callID string) FunctionCallDecision {
	t.Helper()
	// Existing fixture uses public B2 and B1 with B2's SAME returned Catalog.
	return fcoDecision(t, "private_tool", `{"value":"private_argument"}`, "item_private", callID)
}

func fccFailure(t *testing.T, c *FunctionCallCoordinator, ctx context.Context, d FunctionCallDecision, want error) {
	t.Helper()
	output, err := c.Execute(ctx, d)
	if err != want || output.Body() != nil {
		t.Fatalf("unexpected failure/result: %v", err)
	}
}

func fccClaimCount(c *FunctionCallCoordinator) int {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	return len(c.state.claimed)
}

func TestFunctionCallCoordinatorConstructorAndInvalidDecision(t *testing.T) {
	if c, err := NewFunctionCallCoordinator(nil); c != nil || err != ai.ErrInvalidRequest {
		t.Fatal("nil executor accepted")
	}
	calls := 0
	c := fccNew(t, fcoDefinition(), func(context.Context, tool.Call) (string, error) { calls++; return "local result", nil })
	d := fccDecision(t, "call_fixture")
	for _, invalid := range []*FunctionCallCoordinator{nil, {}, {state: &functionCallCoordinationState{}}} {
		fccFailure(t, invalid, context.Background(), d, ai.ErrInvalidRequest)
	}
	// A non-nil zero executor is accepted without C1 introspection; it denies
	// execution through its own public contract, and that claim is consumed.
	zeroExecutor, err := NewFunctionCallCoordinator(&tool.Executor{})
	if err != nil {
		t.Fatal(err)
	}
	fccFailure(t, zeroExecutor, context.Background(), d, tool.ErrExecutionDenied)
	fccFailure(t, zeroExecutor, context.Background(), d, ErrFunctionCallReplay)
	for name, decision := range map[string]FunctionCallDecision{
		"zero":              {},
		"malformed name":    fcoDecision(t, "bad name", `{}`, "item_private", "call_fixture"),
		"unknown tool":      fcoDecision(t, "unknown", `{}`, "item_private", "call_fixture"),
		"invalid arguments": fcoDecision(t, "private_tool", `{}`, "item_private", "call_fixture"),
		"policy rejected":   fcoDecision(t, "private_tool", strings.Repeat(" ", 16385), "item_private", "call_fixture"),
		"no correlation":    {admission: d.Admission()},
		"missing call ID":   {admission: d.Admission(), itemID: "item_private"},
		"missing item ID":   {admission: d.Admission(), callID: "call_fixture"},
		"invalid call ID":   {admission: d.Admission(), itemID: "item_private", callID: "invalid id"},
		"oversized call ID": {admission: d.Admission(), itemID: "item_private", callID: strings.Repeat("c", 257)},
		"invalid item ID":   {admission: d.Admission(), itemID: "invalid item", callID: "call_fixture"},
	} {
		t.Run(name, func(t *testing.T) {
			fccFailure(t, c, context.Background(), decision, ai.ErrInvalidRequest)
			if calls != 0 || fccClaimCount(c) != 0 {
				t.Fatal("invalid decision consumed capacity or executed")
			}
		})
	}
	if _, err := c.Execute(context.Background(), d); err != nil || calls != 1 {
		t.Fatal("invalid decisions consumed valid call ID")
	}
}

func TestFunctionCallCoordinatorEndToEndAndSequentialReplay(t *testing.T) {
	d := fccDecision(t, "call_fixture")
	calls := 0
	c := fccNew(t, fcoDefinition(), func(_ context.Context, call tool.Call) (string, error) {
		calls++
		if call.Name != "private_tool" || string(call.Arguments) != `{"value":"private_argument"}` {
			t.Fatal("C1 received substituted call")
		}
		return "local result", nil
	})
	// Public C3 accepts only decision/context, never a caller ExecutionResult.
	// Pairing is performed internally through actual C1 and C2 contracts.
	output, err := c.Execute(context.Background(), d)
	want := `{"type":"function_call_output","call_id":"call_fixture","output":"local result"}`
	if err != nil || string(output.Body()) != want || calls != 1 {
		t.Fatal("offline pipeline mismatch")
	}
	fccFailure(t, c, context.Background(), d, ErrFunctionCallReplay)
	// Item ID and arguments are not replay identity.
	changed := fcoDecision(t, "private_tool", `{"value":"different"}`, "different_item", "call_fixture")
	fccFailure(t, c, context.Background(), changed, ErrFunctionCallReplay)
	if calls != 1 {
		t.Fatal("sequential replay executed")
	}
	copyOfCoordinator := *c
	fccFailure(t, &copyOfCoordinator, context.Background(), d, ErrFunctionCallReplay)
	// A new explicit coordinator is a new lifetime, even with the same executor.
	other, err := NewFunctionCallCoordinator(c.state.executor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Execute(context.Background(), d); err != nil || calls != 2 {
		t.Fatal("separate coordinator did not start fresh lifetime")
	}
}

func TestFunctionCallCoordinatorConcurrentReplay(t *testing.T) {
	const workers = 32
	var calls atomic.Int32
	entered := make(chan struct{}, workers)
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	c := fccNew(t, fcoDefinition(), func(context.Context, tool.Call) (string, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		return "local result", nil
	})
	d := fccDecision(t, "call_fixture")
	type outcome struct {
		body []byte
		err  error
	}
	outcomes := make(chan outcome, workers)
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(workers)
	for range workers {
		go func() {
			ready.Done()
			<-start
			o, err := c.Execute(context.Background(), d)
			outcomes <- outcome{o.Body(), err}
		}()
	}
	ready.Wait()
	close(start)
	<-entered
	// The winner remains inside its handler while ALL duplicates must finish.
	// This proves the ID is claimed before completion and no lock spans C1.
	for range workers - 1 {
		got := <-outcomes
		if got.err != ErrFunctionCallReplay || got.body != nil {
			t.Fatal("concurrent duplicate not denied")
		}
	}
	if calls.Load() != 1 {
		t.Fatal("same ID entered multiple handlers")
	}
	releaseOnce.Do(func() { close(release) })
	winner := <-outcomes
	if winner.err != nil || string(winner.body) != `{"type":"function_call_output","call_id":"call_fixture","output":"local result"}` {
		t.Fatal("winner did not produce paired output")
	}
	if calls.Load() != 1 || fccClaimCount(c) != 1 {
		t.Fatal("duplicate claims or attempts")
	}
}

func TestFunctionCallCoordinatorFailuresRemainClaimed(t *testing.T) {
	for name, handler := range map[string]tool.Handler{
		"error": func(context.Context, tool.Call) (string, error) {
			return "private_output", errors.New("private_failure")
		},
		"panic": func(context.Context, tool.Call) (string, error) { panic("private_panic") },
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			c := fccNew(t, fcoDefinition(), func(ctx context.Context, call tool.Call) (string, error) { calls++; return handler(ctx, call) })
			d := fccDecision(t, "call_fixture")
			fccFailure(t, c, context.Background(), d, tool.ErrHandlerFailed)
			fccFailure(t, c, context.Background(), d, ErrFunctionCallReplay)
			if calls != 1 || fccClaimCount(c) != 1 {
				t.Fatal("failure released claim or retried")
			}
		})
	}
	for _, missing := range []bool{false, true} {
		definition := fcoDefinition()
		if missing {
			definition.Name = "other"
		} else {
			definition.Parameters[0].Type = tool.Number
		}
		calls := 0
		c := fccNew(t, definition, func(context.Context, tool.Call) (string, error) { calls++; return "", nil })
		d := fccDecision(t, "call_fixture")
		if d.Admission().Status() != tool.Admitted {
			t.Fatal("B1 fixture not admitted")
		}
		fccFailure(t, c, context.Background(), d, tool.ErrExecutionDenied)
		fccFailure(t, c, context.Background(), d, ErrFunctionCallReplay)
		if calls != 0 || fccClaimCount(c) != 1 {
			t.Fatal("C1 denial bypassed or claim released")
		}
	}
}

func TestFunctionCallCoordinatorPreClaimContext(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer stop()
	for _, tc := range []struct {
		ctx  context.Context
		want error
	}{{nil, ai.ErrInvalidRequest}, {canceled, context.Canceled}, {expired, context.DeadlineExceeded}} {
		calls := 0
		c := fccNew(t, fcoDefinition(), func(context.Context, tool.Call) (string, error) { calls++; return "ok", nil })
		d := fccDecision(t, "call_fixture")
		fccFailure(t, c, tc.ctx, d, tc.want)
		if calls != 0 || fccClaimCount(c) != 0 {
			t.Fatal("pre-claim context consumed capacity")
		}
		if _, err := c.Execute(context.Background(), d); err != nil || calls != 1 {
			t.Fatal("healthy retry before claim denied")
		}
	}
}

// Deterministically cancel when C1 checks context, after C3's initial check and
// claim. No sleeps or production hooks are required for this boundary test.
type fccCancelAtC1 struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *fccCancelAtC1) Err() error {
	c.checks++
	if c.checks == 2 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestFunctionCallCoordinatorPostClaimCancellation(t *testing.T) {
	for _, duringHandler := range []bool{false, true} {
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		var ctx context.Context = &fccCancelAtC1{Context: base, cancel: cancel}
		if duringHandler {
			ctx = base
		}
		calls := 0
		c := fccNew(t, fcoDefinition(), func(received context.Context, _ tool.Call) (string, error) {
			calls++
			if received != ctx {
				t.Fatal("context substituted")
			}
			cancel()
			return "private_output", nil
		})
		d := fccDecision(t, "call_fixture")
		fccFailure(t, c, ctx, d, context.Canceled)
		fccFailure(t, c, context.Background(), d, ErrFunctionCallReplay)
		wantCalls := 0
		if duringHandler {
			wantCalls = 1
		}
		if calls != wantCalls || fccClaimCount(c) != 1 {
			t.Fatal("post-claim cancellation enabled retry")
		}
	}
}

func TestFunctionCallCoordinatorDifferentIDsConcurrentPairing(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	var calls atomic.Int32
	c := fccNew(t, fcoDefinition(), func(_ context.Context, call tool.Call) (string, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		return string(call.Arguments), nil
	})
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	for i := range 2 {
		id, args := fmt.Sprintf("call_%d", i), fmt.Sprintf(`{"value":"result_%d"}`, i)
		d := fcoDecision(t, "private_tool", args, "same_item_ID", id)
		go func() {
			defer func() { done <- struct{}{} }()
			ready.Done()
			<-start
			o, err := c.Execute(context.Background(), d)
			if err != nil {
				t.Error(err)
				return
			}
			var item map[string]string
			if json.Unmarshal(o.Body(), &item) != nil || len(item) != 3 || item["call_id"] != id || item["output"] != args || item["type"] != "function_call_output" {
				t.Error("concurrent decision/result pairing mismatch")
			}
		}()
	}
	ready.Wait()
	close(start)
	<-entered
	<-entered // both handlers must overlap; lock cannot cover handler execution
	once.Do(func() { close(release) })
	<-done
	<-done
	if calls.Load() != 2 || fccClaimCount(c) != 2 {
		t.Fatal("different IDs conflated")
	}
}

func TestFunctionCallCoordinatorCapacity(t *testing.T) {
	calls := 0
	c := fccNew(t, fcoDefinition(), func(context.Context, tool.Call) (string, error) {
		calls++
		if calls%2 == 0 {
			return "", errors.New("private_failure")
		}
		return "ok", nil
	})
	for i := range maxCoordinatedFunctionCalls {
		d := fccDecision(t, fmt.Sprintf("call_%d", i))
		o, err := c.Execute(context.Background(), d)
		if i%2 == 0 {
			if err != nil || o.Body() == nil {
				t.Fatal("claim below capacity rejected")
			}
		} else if err != tool.ErrHandlerFailed || o.Body() != nil {
			t.Fatal("failed claim classification changed")
		}
	}
	if calls != 1024 || fccClaimCount(c) != 1024 {
		t.Fatal("capacity bound changed or failed claims released")
	}
	fccFailure(t, c, context.Background(), fccDecision(t, "call_overflow"), ErrFunctionCallCapacity)
	for _, id := range []string{"call_0", "call_1", "call_1023"} {
		fccFailure(t, c, context.Background(), fccDecision(t, id), ErrFunctionCallReplay)
	}
	if calls != 1024 || fccClaimCount(c) != 1024 {
		t.Fatal("overflow executed, evicted, or inserted")
	}
}

func TestFunctionCallCoordinatorOwnershipAndDiagnostics(t *testing.T) {
	d := fccDecision(t, "call_private")
	calls := 0
	c := fccNew(t, fcoDefinition(), func(context.Context, tool.Call) (string, error) { calls++; return "private_output", nil })
	if _, err := c.Execute(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	d.callID, d.itemID, d.admission = "changed_call", "changed_item", tool.Admission{}
	fccFailure(t, c, context.Background(), fccDecision(t, "call_private"), ErrFunctionCallReplay)
	if calls != 1 || fccClaimCount(c) != 1 {
		t.Fatal("caller reassignment altered claim")
	}
	for _, value := range []any{c, *c, FunctionCallCoordinator{}, (*FunctionCallCoordinator)(nil), ErrFunctionCallReplay, ErrFunctionCallCapacity} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			rendered := fmt.Sprintf(format, value)
			for _, secret := range []string{"call_private", "item_private", "private_tool", "private_argument", "private_output", "changed_call", "claimed", "1024"} {
				if strings.Contains(rendered, secret) {
					t.Fatal("diagnostic disclosed state or payload")
				}
			}
		}
	}
	for err, want := range map[error]string{ErrFunctionCallReplay: "openai: function call replay denied", ErrFunctionCallCapacity: "openai: function call capacity reached"} {
		if err.Error() != want || errors.Unwrap(err) != nil {
			t.Fatal("error is not fixed/redacted")
		}
	}
}
