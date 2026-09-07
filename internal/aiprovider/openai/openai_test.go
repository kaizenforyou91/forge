package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
)

const canary = "dummy-credential-CANARY-123"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	reader   io.Reader
	closed   int
	closeErr error
}

func (b *trackedBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *trackedBody) Close() error               { b.closed++; return b.closeErr }

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }
func request() ai.Request {
	return ai.Request{Text: "  Dummy note.\n", Model: "gpt-4.1-mini-2025-04-14", MaxOutputTokens: 1024}
}
func message(text string) map[string]any {
	return map[string]any{"type": "message", "status": "completed", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": text}}}
}
func responseObject(text string) map[string]any {
	return map[string]any{"status": "completed", "error": nil, "incomplete_details": nil, "model": request().Model, "output": []any{message(text)}}
}
func encoded(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
func testClient(t *testing.T, rt http.RoundTripper) *Client {
	t.Helper()
	c, err := New(canary)
	if err != nil {
		t.Fatal(err)
	}
	c.http = newHTTPClient(rt)
	return c
}
func responseClient(t *testing.T, status int, body *trackedBody, calls *int) *Client {
	return testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		*calls++
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: body}, nil
	}))
}
func assertSafe(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(fmt.Sprintf("%+v %#v", err, err), canary) {
		t.Fatal("credential leaked")
	}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range multi.Unwrap() {
			assertSafe(t, e)
		}
	}
	if single, ok := err.(interface{ Unwrap() error }); ok {
		assertSafe(t, single.Unwrap())
	}
}
func requireFailure(t *testing.T, result ai.Result, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || !reflect.DeepEqual(result, ai.Result{}) {
		t.Fatalf("category=%v want=%v; result must be empty", err, want)
	}
	assertSafe(t, err)
}
func TestWireContractAndAggregation(t *testing.T) {
	calls := 0
	obj := responseObject("first")
	obj["output"] = []any{message("first "), message("second\n")}
	obj["usage"] = map[string]any{"input_tokens": 10, "output_tokens": 4, "total_tokens": 14}
	obj["extra_metadata"] = map[string]any{"ignored": true}
	body := &trackedBody{reader: strings.NewReader(encoded(obj))}
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.String() != endpoint || r.GetBody != nil || r.ContentLength <= 0 {
			t.Fatal("method/endpoint/replay contract")
		}
		if r.Header.Get("Authorization") != "Bearer "+canary || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("headers")
		}
		if len(r.Header) != 3 || r.Header.Get("User-Agent") != "" || r.Header.Get("Idempotency-Key") != "" {
			t.Fatal("unexpected header")
		}
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		var data map[string]any
		if err := json.Unmarshal(got, &data); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"model": request().Model, "input": request().Text, "max_output_tokens": float64(1024), "stream": false, "background": false, "store": false}
		if !reflect.DeepEqual(data, want) {
			t.Fatal("payload mismatch")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
	}))
	result, err := c.Execute(context.Background(), request())
	if err != nil || calls != 1 || body.closed != 1 || result.Text != "first second\n" || result.Usage == nil || result.Usage.OutputTokens != 4 {
		t.Fatal("response mismatch", err)
	}
	if strings.Contains(fmt.Sprintf("%v %+v %#v", c, c, c), canary) {
		t.Fatal("client formatting leaks key")
	}
}
func TestDecoderFailuresAndClosure(t *testing.T) {
	cases := map[string]struct {
		body string
		want error
	}{
		"malformed":         {"{", ai.ErrMalformedResponse},
		"array":             {"[]", ai.ErrMalformedResponse},
		"null":              {"null", ai.ErrMalformedResponse},
		"trailing":          {encoded(responseObject("OK")) + " {}", ai.ErrMalformedResponse},
		"garbage":           {encoded(responseObject("OK")) + "!", ai.ErrMalformedResponse},
		"utf8":              {strings.Replace(encoded(responseObject("OK")), "OK", "\xff", 1), ai.ErrMalformedResponse},
		"wrong status type": {`{"status":123}`, ai.ErrMalformedResponse},
		"unknown status":    {`{"status":"other"}`, ai.ErrMalformedResponse},
		"incomplete":        {`{"status":"incomplete"}`, ai.ErrIncompleteResponse},
		"queued":            {`{"status":"queued"}`, ai.ErrIncompleteResponse},
		"in_progress":       {`{"status":"in_progress"}`, ai.ErrIncompleteResponse},
		"failed":            {`{"status":"failed"}`, ai.ErrProvider},
	}
	for name, change := range map[string]func(map[string]any){
		"missing model":       func(o map[string]any) { delete(o, "model") },
		"missing error":       func(o map[string]any) { delete(o, "error") },
		"null model":          func(o map[string]any) { o["model"] = nil },
		"empty output":        func(o map[string]any) { o["output"] = []any{} },
		"wrong output type":   func(o map[string]any) { o["output"] = "text" },
		"top-level text only": func(o map[string]any) { delete(o, "output"); o["output_text"] = "ignored" },
		"tool":                func(o map[string]any) { o["output"] = []any{map[string]any{"type": "function_call", "name": "shell"}} },
		"role":                func(o map[string]any) { m := message("OK"); m["role"] = "user"; o["output"] = []any{m} },
		"message status":      func(o map[string]any) { m := message("OK"); delete(m, "status"); o["output"] = []any{m} },
		"content type": func(o map[string]any) {
			m := message("OK")
			m["content"] = []any{map[string]any{"type": "audio"}}
			o["output"] = []any{m}
		},
		"null text": func(o map[string]any) {
			m := message("OK")
			m["content"] = []any{map[string]any{"type": "output_text", "text": nil}}
			o["output"] = []any{m}
		},
		"empty text":          func(o map[string]any) { o["output"] = []any{message(" \n")} },
		"ansi":                func(o map[string]any) { o["output"] = []any{message("\x1b[31mOK")} },
		"C1":                  func(o map[string]any) { o["output"] = []any{message("\u009b31mOK")} },
		"CR":                  func(o map[string]any) { o["output"] = []any{message("OK\r")} },
		"secret text":         func(o map[string]any) { o["output"] = []any{message(canary)} },
		"secret model":        func(o map[string]any) { o["model"] = canary },
		"bad usage":           func(o map[string]any) { o["usage"] = map[string]any{"input_tokens": -1, "output_tokens": 1} },
		"missing usage field": func(o map[string]any) { o["usage"] = map[string]any{"input_tokens": 1} },
		"usage fractional":    func(o map[string]any) { o["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1.5} },
		"usage null count":    func(o map[string]any) { o["usage"] = map[string]any{"input_tokens": nil, "output_tokens": 1} },
		"usage over limit":    func(o map[string]any) { o["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1025} },
	} {
		obj := responseObject("OK")
		change(obj)
		cases[name] = struct {
			body string
			want error
		}{encoded(obj), ai.ErrMalformedResponse}
	}
	for _, kind := range []string{"refusal", "message incomplete", "details", "provider error", "oversized text"} {
		o := responseObject("OK")
		want := ai.ErrMalformedResponse
		switch kind {
		case "refusal":
			m := message("OK")
			m["content"] = []any{map[string]any{"type": "output_text", "text": "partial"}, map[string]any{"type": "refusal", "refusal": canary}}
			o["output"] = []any{m}
			want = ai.ErrRefused
		case "message incomplete":
			m := message("partial")
			m["status"] = "incomplete"
			o["output"] = []any{m}
			want = ai.ErrIncompleteResponse
		case "details":
			o["incomplete_details"] = map[string]any{"reason": "max_output_tokens"}
			want = ai.ErrIncompleteResponse
		case "provider error":
			o["error"] = map[string]any{"message": canary}
			want = ai.ErrProvider
		case "oversized text":
			o["output"] = []any{message(strings.Repeat("x", ai.MaxTextBytes)), message("x")}
			want = ai.ErrResponseTooLarge
		}
		cases[kind] = struct {
			body string
			want error
		}{encoded(o), want}
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			b := &trackedBody{reader: strings.NewReader(tc.body)}
			r, err := responseClient(t, 200, b, &calls).Execute(context.Background(), request())
			requireFailure(t, r, err, tc.want)
			if calls != 1 || b.closed != 1 {
				t.Fatal("retry or missing close")
			}
		})
	}
}
func TestHTTPStatusCategories(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		want   error
	}{
		{401, "", ai.ErrAuthentication}, {403, "", ai.ErrAuthorization}, {429, "rate_limit_exceeded", ai.ErrRateLimited},
		{429, "credit_balance_exhausted", ai.ErrQuotaExceeded}, {429, "organization_spend_limit_exceeded", ai.ErrQuotaExceeded},
		{429, "project_spend_limit_exceeded", ai.ErrQuotaExceeded}, {429, "organization_usage_limit_exceeded", ai.ErrQuotaExceeded},
		{429, "insufficient_quota", ai.ErrQuotaExceeded}, {429, "slow_down", ai.ErrRateLimited},
		{429, "unknown", ai.ErrRateLimited}, {400, "", ai.ErrProvider}, {404, "", ai.ErrProvider}, {500, "", ai.ErrProvider}, {503, "", ai.ErrProvider},
	} {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.code), func(t *testing.T) {
			// A human message containing quota words must not affect classification.
			raw := encoded(map[string]any{"error": map[string]any{"code": tc.code, "message": canary + " insufficient_quota credit_balance_exhausted"}})
			calls := 0
			b := &trackedBody{reader: strings.NewReader(raw)}
			r, err := responseClient(t, tc.status, b, &calls).Execute(context.Background(), request())
			requireFailure(t, r, err, tc.want)
			if calls != 1 || b.closed != 1 {
				t.Fatal("retry/closure")
			}
		})
	}
}
func TestBoundedReadAndCloseFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		body *trackedBody
		want error
	}{
		{"body+1", &trackedBody{reader: strings.NewReader(strings.Repeat("x", maxBody+1))}, ai.ErrResponseTooLarge},
		{"read", &trackedBody{reader: readerFunc(func([]byte) (int, error) { return 0, errors.New(canary) })}, ai.ErrTransport},
		{"close", &trackedBody{reader: strings.NewReader(encoded(responseObject("OK"))), closeErr: errors.New(canary)}, ai.ErrTransport},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r, err := responseClient(t, 200, tc.body, &calls).Execute(context.Background(), request())
			requireFailure(t, r, err, tc.want)
			if tc.body.closed != 1 || calls != 1 {
				t.Fatal("close/call")
			}
		})
	}
	// Exact body and text limits are accepted; absent usage remains absent.
	data := encoded(responseObject(strings.Repeat("x", ai.MaxTextBytes)))
	data += strings.Repeat(" ", maxBody-len(data))
	calls := 0
	b := &trackedBody{reader: strings.NewReader(data)}
	r, err := responseClient(t, 200, b, &calls).Execute(context.Background(), request())
	if err != nil || len(r.Text) != ai.MaxTextBytes || r.Usage != nil || b.closed != 1 {
		t.Fatal("exact limits", err)
	}
	calls = 0
	b = &trackedBody{reader: strings.NewReader(strings.Repeat("x", maxBody+1))}
	r, err = responseClient(t, 401, b, &calls).Execute(context.Background(), request())
	requireFailure(t, r, err, ai.ErrResponseTooLarge)
	if b.closed != 1 {
		t.Fatal("error body not closed")
	}
	reads := 0
	b = &trackedBody{reader: readerFunc(func(p []byte) (int, error) {
		reads += len(p)
		for i := range p {
			p[i] = 'x'
		}
		return len(p), nil
	})}
	calls = 0
	r, err = responseClient(t, 200, b, &calls).Execute(context.Background(), request())
	requireFailure(t, r, err, ai.ErrResponseTooLarge)
	if reads != maxBody+1 || b.closed != 1 {
		t.Fatal("unbounded drain")
	}
}
func TestConstructorAndValidationNoNetwork(t *testing.T) {
	for _, key := range []string{"", " ", "a b", "a\nb", "\xff", strings.Repeat("a", 16*1024+1)} {
		if _, err := New(key); !errors.Is(err, ai.ErrAuthentication) {
			t.Fatal("credential accepted")
		}
	}
	calls := 0
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New(canary) }))
	r := request()
	r.MaxOutputTokens = 0
	result, err := c.Execute(context.Background(), r)
	requireFailure(t, result, err, ai.ErrInvalidRequest)
	result, err = c.Execute(nil, request())
	requireFailure(t, result, err, ai.ErrInvalidRequest)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = c.Execute(ctx, request())
	requireFailure(t, result, err, context.Canceled)
	if calls != 0 {
		t.Fatal("invalid request sent")
	}
	result, err = c.Execute(context.Background(), request())
	requireFailure(t, result, err, ai.ErrTransport)
	if calls != 1 {
		t.Fatal("transport retried")
	}
}
func TestRequestAndBodyCancellation(t *testing.T) {
	for _, phase := range []string{"request", "body", "deadline"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if phase == "deadline" {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(ctx, time.Second)
					defer stop()
				}
				entered := make(chan struct{})
				calls := 0
				var b *trackedBody
				c := testClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if phase == "request" {
						close(entered)
						<-req.Context().Done()
						return nil, fmt.Errorf("%s: %w", canary, req.Context().Err())
					}
					b = &trackedBody{reader: readerFunc(func([]byte) (int, error) {
						close(entered)
						<-req.Context().Done()
						return 0, fmt.Errorf("%s: %w", canary, req.Context().Err())
					})}
					return &http.Response{StatusCode: 200, Body: b, Header: make(http.Header)}, nil
				}))
				if phase != "deadline" {
					go func() { <-entered; cancel() }()
				}
				result, err := c.Execute(ctx, request())
				want := context.Canceled
				if phase == "deadline" {
					want = context.DeadlineExceeded
				}
				requireFailure(t, result, err, want)
				if calls != 1 || b != nil && b.closed != 1 {
					t.Fatal("cancel cleanup/counter")
				}
			})
		})
	}
}

// Route only to this test's loopback server; there is no production fallback.
func loopbackClient(t *testing.T, handler http.HandlerFunc) (*Client, *atomic.Int32) {
	t.Helper()
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { count.Add(1); handler(w, r) }))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := newTransport()
	transport.Proxy = nil
	t.Cleanup(transport.CloseIdleConnections)
	c := testClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		copied := req.Clone(req.Context())
		u := *req.URL
		u.Scheme = target.Scheme
		u.Host = target.Host
		copied.URL = &u
		copied.Host = target.Host
		return transport.RoundTrip(copied)
	}))
	return c, &count
}
func TestRealTransportHeadersRedirectAndDisconnect(t *testing.T) {
	tr := newTransport()
	if tr.MaxResponseHeaderBytes != maxHeaders || !tr.DisableKeepAlives || tr.Proxy == nil || tr.TLSClientConfig != nil || tr.Protocols.HTTP2() || !tr.Protocols.HTTP1() {
		t.Fatal("transport policy")
	}
	for _, scenario := range []string{"headers", "redirect", "disconnect", "rate"} {
		t.Run(scenario, func(t *testing.T) {
			c, count := loopbackClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch scenario {
				case "headers":
					w.Header().Set("X-Large", strings.Repeat("x", maxHeaders+1))
					fmt.Fprint(w, encoded(responseObject("OK")))
				case "redirect":
					w.Header().Set("Location", "/should-not-follow")
					w.WriteHeader(307)
					fmt.Fprint(w, canary)
				case "disconnect":
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					conn.Close()
				case "rate":
					w.WriteHeader(429)
					fmt.Fprint(w, canary)
				}
			})
			result, err := c.Execute(context.Background(), request())
			want := ai.ErrTransport
			if scenario == "redirect" {
				want = ai.ErrProvider
			}
			if scenario == "rate" {
				want = ai.ErrRateLimited
			}
			requireFailure(t, result, err, want)
			if count.Load() != 1 {
				t.Fatal("HTTP retried or redirected", count.Load())
			}
		})
	}
	// Injected redirect verifies body ownership even without a Location.
	calls := 0
	b := &trackedBody{reader: bytes.NewBufferString(canary)}
	c := responseClient(t, 302, b, &calls)
	result, err := c.Execute(context.Background(), request())
	requireFailure(t, result, err, ai.ErrProvider)
	if b.closed != 1 {
		t.Fatal("redirect body not closed")
	}
}

func TestCancellationPlusCloseFailureAndRedirectOwnership(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := &trackedBody{reader: readerFunc(func([]byte) (int, error) { cancel(); return 0, fmt.Errorf("%s: %w", canary, context.Canceled) }), closeErr: errors.New(canary)}
	calls := 0
	result, err := responseClient(t, 200, b, &calls).Execute(ctx, request())
	requireFailure(t, result, err, context.Canceled)
	if !errors.Is(err, ai.ErrTransport) || b.closed != 1 || calls != 1 {
		t.Fatal("cleanup classification lost")
	}

	b = &trackedBody{reader: strings.NewReader(canary)}
	calls = 0
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": []string{endpoint + "/redirect"}}, Body: b}, nil
	}))
	result, err = c.Execute(context.Background(), request())
	requireFailure(t, result, err, ai.ErrProvider)
	if calls != 1 || b.closed != 1 {
		t.Fatal("redirect followed or body leaked")
	}
}
