package ai

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type fakeProvider func(context.Context, Request) (Result, error)

func (f fakeProvider) Execute(ctx context.Context, r Request) (Result, error) { return f(ctx, r) }
func validRequest() Request {
	return Request{Text: "  Ringkas catatan dummy.\n", Model: "gpt-4.1-mini-2025-04-14", MaxOutputTokens: 1024}
}
func validResult() Result { return Result{Text: "Demo Jumat.", Model: "gpt-4.1-mini-2025-04-14"} }
func requireFailure(t *testing.T, result Result, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || !reflect.DeepEqual(result, Result{}) {
		t.Fatalf("result must be empty; category=%v want=%v", err, want)
	}
}
func TestExecutorPreservesRequestAndCallsOnce(t *testing.T) {
	req := validRequest()
	calls := 0
	usage := &Usage{InputTokens: 10, OutputTokens: 3}
	e, err := NewExecutor(fakeProvider(func(ctx context.Context, got Request) (Result, error) {
		calls++
		if got != req {
			t.Fatal("request was rewritten")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing deadline")
		}
		r := validResult()
		r.Usage = usage
		return r, nil
	}), 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	result, err := e.Execute(context.Background(), req)
	if err != nil || result.Text != "Demo Jumat." || calls != 1 {
		t.Fatal("execution mismatch", err, calls)
	}
	usage.OutputTokens = 99
	if result.Usage.OutputTokens != 3 {
		t.Fatal("usage alias leaked")
	}
}
func TestExecutorRejectsInvalidRequestsBeforeProvider(t *testing.T) {
	cases := map[string]func(*Request){
		"empty":         func(r *Request) { r.Text = "" },
		"blank":         func(r *Request) { r.Text = " \t\n\u2003" },
		"utf8":          func(r *Request) { r.Text = "\xff" },
		"input+1":       func(r *Request) { r.Text = strings.Repeat("a", MaxInputBytes+1) },
		"model missing": func(r *Request) { r.Model = "" },
		"model+1":       func(r *Request) { r.Model = strings.Repeat("a", MaxModelBytes+1) },
		"model prefix":  func(r *Request) { r.Model = "-abc" },
		"model space":   func(r *Request) { r.Model = "a b" },
		"model slash":   func(r *Request) { r.Model = "a/b" },
		"model unicode": func(r *Request) { r.Model = "模型" },
		"zero tokens":   func(r *Request) { r.MaxOutputTokens = 0 },
		"tokens-1":      func(r *Request) { r.MaxOutputTokens = 15 },
		"tokens+1":      func(r *Request) { r.MaxOutputTokens = 2049 },
		"negative":      func(r *Request) { r.MaxOutputTokens = -1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			e, _ := NewExecutor(fakeProvider(func(context.Context, Request) (Result, error) { calls++; return validResult(), nil }), time.Second)
			r := validRequest()
			mutate(&r)
			result, err := e.Execute(context.Background(), r)
			requireFailure(t, result, err, ErrInvalidRequest)
			if calls != 0 || strings.Contains(err.Error(), r.Text) && r.Text != "" && r.Text != " " && r.Text != "\n" && len(r.Text) > 10 {
				t.Fatal("provider called or input leaked")
			}
		})
	}
}
func TestRequestAndResultLimits(t *testing.T) {
	for _, tokens := range []int{16, 2048} {
		r := Request{Text: strings.Repeat("é", MaxInputBytes/2), Model: strings.Repeat("a", 128), MaxOutputTokens: tokens}
		if err := r.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	r := validRequest()
	r.Model = "A0._:-z"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	out := validResult()
	out.Text = strings.Repeat("a", MaxTextBytes)
	if err := out.Validate(); err != nil {
		t.Fatal(err)
	}
	out.Text += "a"
	if !errors.Is(out.Validate(), ErrResponseTooLarge) {
		t.Fatal("text bound")
	}
}
func TestInvalidConstructionAndNil(t *testing.T) {
	p := fakeProvider(func(context.Context, Request) (Result, error) { t.Fatal("unexpected provider"); return Result{}, nil })
	for _, timeout := range []time.Duration{0, -1, MaxTimeout + 1} {
		if _, err := NewExecutor(p, timeout); !errors.Is(err, ErrInvalidRequest) {
			t.Fatal("invalid timeout accepted")
		}
	}
	if _, err := NewExecutor(p, MaxTimeout); err != nil {
		t.Fatal(err)
	}
	var typedNil fakeProvider
	for _, p := range []Provider{nil, typedNil} {
		if _, err := NewExecutor(p, time.Second); !errors.Is(err, ErrInvalidRequest) {
			t.Fatal("nil accepted")
		}
	}
	e, _ := NewExecutor(p, time.Second)
	r, err := e.Execute(nil, validRequest())
	requireFailure(t, r, err, ErrInvalidRequest)
	var zero *Executor
	r, err = zero.Execute(context.Background(), validRequest())
	requireFailure(t, r, err, ErrInvalidRequest)
}
func TestResultValidationAndNoPartialOnError(t *testing.T) {
	cases := map[string]struct {
		r    Result
		err  error
		want error
	}{
		"blank":           {Result{Text: " ", Model: "m"}, nil, ErrMalformedResponse},
		"utf8":            {Result{Text: "\xff", Model: "m"}, nil, ErrMalformedResponse},
		"model":           {Result{Text: "OK"}, nil, ErrMalformedResponse},
		"large":           {Result{Text: strings.Repeat("x", MaxTextBytes+1), Model: "m"}, nil, ErrResponseTooLarge},
		"negative input":  {Result{Text: "OK", Model: "m", Usage: &Usage{InputTokens: -1}}, nil, ErrMalformedResponse},
		"negative output": {Result{Text: "OK", Model: "m", Usage: &Usage{OutputTokens: -1}}, nil, ErrMalformedResponse},
		"too many tokens": {Result{Text: "OK", Model: "m", Usage: &Usage{OutputTokens: 1025}}, nil, ErrMalformedResponse},
		"partial error":   {validResult(), fmt.Errorf("dummy-secret: %w", ErrIncompleteResponse), ErrIncompleteResponse},
		"unknown error":   {validResult(), errors.New("dummy-secret"), ErrProvider},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e, _ := NewExecutor(fakeProvider(func(context.Context, Request) (Result, error) { return tc.r, tc.err }), time.Second)
			result, err := e.Execute(context.Background(), validRequest())
			requireFailure(t, result, err, tc.want)
			if strings.Contains(fmt.Sprintf("%+v", err), "dummy-secret") {
				t.Fatal("secret leaked")
			}
		})
	}
	e, _ := NewExecutor(fakeProvider(func(context.Context, Request) (Result, error) { return validResult(), nil }), time.Second)
	r, err := e.Execute(context.Background(), validRequest())
	if err != nil || r.Usage != nil {
		t.Fatal("absent usage invented")
	}
}
func TestTerminalSafety(t *testing.T) {
	for c := rune(0); c <= 0x9f; c++ {
		if c >= 0x20 && c < 0x7f || c == '\t' || c == '\n' {
			continue
		}
		r := Result{Text: "before" + string(c) + "after", Model: "m"}
		if !errors.Is(r.Validate(), ErrMalformedResponse) {
			t.Fatalf("control U+%04X accepted", c)
		}
	}
	for _, text := range []string{"hello\tworld\n", "rm -rf /", "$(echo dummy)", "中文 café"} {
		if err := (Result{Text: text, Model: "m"}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestCancellationAndDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		calls := 0
		e, _ := NewExecutor(fakeProvider(func(context.Context, Request) (Result, error) { calls++; return validResult(), nil }), time.Second)
		r, err := e.Execute(canceled, validRequest())
		requireFailure(t, r, err, context.Canceled)
		if calls != 0 {
			t.Fatal("pre-canceled call")
		}
		for _, callerShorter := range []bool{false, true} {
			parent := context.Background()
			duration := time.Second
			if callerShorter {
				var stop context.CancelFunc
				parent, stop = context.WithTimeout(parent, 100*time.Millisecond)
				defer stop()
				duration = 100 * time.Millisecond
			}
			start := time.Now()
			var requestContext context.Context
			e, _ := NewExecutor(fakeProvider(func(ctx context.Context, _ Request) (Result, error) {
				requestContext = ctx
				calls++
				<-ctx.Done()
				return validResult(), nil
			}), time.Second)
			r, err := e.Execute(parent, validRequest())
			requireFailure(t, r, err, context.DeadlineExceeded)
			if time.Since(start) != duration || requestContext.Err() != context.DeadlineExceeded {
				t.Fatal("deadline not enforced")
			}
		}
		ctx, stop := context.WithCancel(context.Background())
		entered := make(chan struct{})
		done := make(chan struct{})
		e, _ = NewExecutor(fakeProvider(func(ctx context.Context, _ Request) (Result, error) {
			close(entered)
			<-ctx.Done()
			close(done)
			return Result{}, fmt.Errorf("secret: %w", ctx.Err())
		}), time.Second)
		go func() { <-entered; stop() }()
		r, err = e.Execute(ctx, validRequest())
		requireFailure(t, r, err, context.Canceled)
		<-done
	})
}
func TestRequestLocalContextAndSafeCategories(t *testing.T) {
	var previous context.Context
	e, _ := NewExecutor(fakeProvider(func(ctx context.Context, _ Request) (Result, error) {
		if ctx == previous || ctx.Err() != nil {
			t.Fatal("context reused")
		}
		previous = ctx
		return validResult(), nil
	}), time.Second)
	for i := 0; i < 2; i++ {
		if _, err := e.Execute(context.Background(), validRequest()); err != nil {
			t.Fatal(err)
		}
		if previous.Err() != context.Canceled {
			t.Fatal("request context not released")
		}
	}
	for _, category := range []error{ErrInvalidRequest, ErrAuthentication, ErrAuthorization, ErrRateLimited, ErrQuotaExceeded, ErrTransport, ErrMalformedResponse, ErrResponseTooLarge, ErrIncompleteResponse, ErrRefused, ErrProvider, context.Canceled, context.DeadlineExceeded} {
		safe := SafeError(fmt.Errorf("dummy-secret: %w", category))
		if !errors.Is(safe, category) || strings.Contains(safe.Error(), "dummy-secret") {
			t.Fatal("unsafe classification")
		}
	}
	safe := SafeError(errors.Join(fmt.Errorf("secret: %w", context.Canceled), fmt.Errorf("secret: %w", ErrTransport)))
	if !errors.Is(safe, context.Canceled) || !errors.Is(safe, ErrTransport) {
		t.Fatal("lost joined failure")
	}
}

func TestConcurrentExecutorRequestContexts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := make(chan context.Context, 2)
		release := make(chan struct{})
		e, err := NewExecutor(fakeProvider(func(ctx context.Context, r Request) (Result, error) {
			entered <- ctx
			<-release
			return Result{Text: r.Text, Model: r.Model}, nil
		}), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		results := make(chan Result, 2)
		errs := make(chan error, 2)
		for _, text := range []string{"one", "two"} {
			r := validRequest()
			r.Text = text
			go func() { result, err := e.Execute(context.Background(), r); results <- result; errs <- err }()
		}
		a, b := <-entered, <-entered
		if a == b {
			t.Fatal("shared request context")
		}
		close(release)
		got := map[string]bool{}
		for i := 0; i < 2; i++ {
			r := <-results
			got[r.Text] = true
			if err := <-errs; err != nil {
				t.Fatal(err)
			}
		}
		if !got["one"] || !got["two"] || a.Err() != context.Canceled || b.Err() != context.Canceled {
			t.Fatal("request state leak")
		}
	})
}

func TestSafeErrorPreservesUnknownFailures(t *testing.T) {
	const secret = "dummy-UNKNOWN-FAILURE-CANARY"
	unknown := errors.New(secret)
	wrapped := func(err error) error { return fmt.Errorf("%s: %w", secret, err) }
	cases := []struct {
		name string
		err  error
		want []error
	}{
		{"nil", nil, nil},
		{"pure cancellation", context.Canceled, []error{context.Canceled}},
		{"wrapped cancellation", wrapped(context.Canceled), []error{context.Canceled}},
		{"unknown", unknown, []error{ErrProvider}},
		{"wrapped unknown", wrapped(unknown), []error{ErrProvider}},
		{"typed mixed", errors.Join(context.Canceled, ErrTransport), []error{context.Canceled, ErrTransport}},
		{"unknown sibling", errors.Join(context.Canceled, unknown), []error{context.Canceled, ErrProvider}},
		{"reversed siblings", errors.Join(unknown, context.Canceled), []error{context.Canceled, ErrProvider}},
		{"outer wrapper", wrapped(errors.Join(context.Canceled, unknown)), []error{context.Canceled, ErrProvider}},
		{"nested joins", errors.Join(wrapped(context.Canceled), errors.Join(ErrTransport, wrapped(unknown))), []error{context.Canceled, ErrTransport, ErrProvider}},
		{"nested reversed", errors.Join(errors.Join(wrapped(unknown), ErrTransport), wrapped(context.Canceled)), []error{context.Canceled, ErrTransport, ErrProvider}},
		{"known non-cancellation", errors.Join(ErrAuthentication, unknown), []error{ErrAuthentication, ErrProvider}},
		{"multiple percent w", fmt.Errorf("%s: %w / %w", secret, context.Canceled, unknown), []error{context.Canceled, ErrProvider}},
		{"duplicate categories", errors.Join(context.Canceled, context.Canceled, unknown, unknown), []error{context.Canceled, ErrProvider}},
	}
	for _, category := range []error{context.DeadlineExceeded, ErrInvalidRequest, ErrAuthentication, ErrAuthorization, ErrRateLimited, ErrQuotaExceeded, ErrTransport, ErrMalformedResponse, ErrResponseTooLarge, ErrIncompleteResponse, ErrRefused, ErrProvider} {
		cases = append(cases, struct {
			name string
			err  error
			want []error
		}{"wrapped " + category.Error(), wrapped(category), []error{category}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err
			for pass := 1; pass <= 2; pass++ {
				got = SafeError(got)
				if len(tc.want) == 0 {
					if got != nil {
						t.Fatal("nil became failure")
					}
					continue
				}
				if got == nil {
					t.Fatal("failure disappeared")
				}
				for _, category := range tc.want {
					if !errors.Is(got, category) {
						t.Errorf("pass %d: lost category %v", pass, category)
					}
				}
				if got.Error() != errors.Join(tc.want...).Error() {
					t.Errorf("pass %d: unexpected sanitized categories or ordering", pass)
				}
				if errors.Is(got, unknown) {
					t.Error("raw cause retained")
				}
				assertSanitizedTree(t, got, secret)
			}
		})
	}
}

func assertSanitizedTree(t *testing.T, err error, secret string) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(fmt.Sprintf("%v %+v %#v", err, err, err), secret) {
		t.Fatal("secret in sanitized error tree")
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range joined.Unwrap() {
			assertSanitizedTree(t, child, secret)
		}
		return
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		assertSanitizedTree(t, wrapped.Unwrap(), secret)
	}
}

func TestExecutorPreservesUnknownFailure(t *testing.T) {
	const secret = "dummy-EXECUTOR-UNKNOWN-CANARY"
	unknown := errors.New(secret)
	calls := 0
	executor, err := NewExecutor(fakeProvider(func(context.Context, Request) (Result, error) {
		calls++
		return validResult(), errors.Join(context.Canceled, unknown)
	}), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(context.Background(), validRequest())
	requireFailure(t, result, err, context.Canceled)
	if !errors.Is(err, ErrProvider) {
		t.Error("unknown failure lost its safe category")
	}
	if calls != 1 {
		t.Error("provider call count changed")
	}
	if errors.Is(err, unknown) {
		t.Error("raw provider cause retained")
	}
	assertSanitizedTree(t, err, secret)
}
