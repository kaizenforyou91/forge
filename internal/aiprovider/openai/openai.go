// Package openai implements the bounded, non-streaming Responses API adapter.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kaizenforyou91/forge/pkg/ai"
)

const (
	endpoint   = "https://api.openai.com/v1/responses"
	maxHeaders = 16 * 1024
	maxBody    = 1024 * 1024
)

// Client owns its HTTP transport. Constructing it does not perform I/O.
type Client struct {
	key  string
	http *http.Client
}

func (c *Client) String() string   { return "OpenAI Responses client (credential redacted)" }
func (c *Client) GoString() string { return c.String() }

// New accepts a nonempty printable-ASCII bearer credential (at most 16 KiB).
// The caller supplies the key; this package never reads environment or files.
func New(key string) (*Client, error) {
	if len(key) == 0 || len(key) > 16*1024 {
		return nil, ai.ErrAuthentication
	}
	for _, b := range []byte(key) {
		if b <= 0x20 || b >= 0x7f {
			return nil, ai.ErrAuthentication
		}
	}
	return &Client{key: key, http: newHTTPClient(newTransport())}, nil
}

func newTransport() *http.Transport {
	// Preserve environment proxy and system TLS roots; do not clone or mutate a
	// possibly customized global transport. HTTP/1 only, fresh connections and a
	// non-replayable POST body prevent transport resubmission on stale/H2 streams.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	return &http.Transport{
		Proxy:                  http.ProxyFromEnvironment,
		DialContext:            (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: -1}).DialContext,
		TLSHandshakeTimeout:    10 * time.Second,
		ResponseHeaderTimeout:  120 * time.Second,
		MaxResponseHeaderBytes: maxHeaders,
		DisableKeepAlives:      true,
		Protocols:              protocols,
	}
}
func newHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: transport,
		// ErrUseLastResponse leaves the redirect body open for our bounded read/close;
		// a normal redirect error makes net/http drain it internally.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type payload struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	Stream          bool   `json:"stream"`
	Background      bool   `json:"background"`
	Store           bool   `json:"store"`
}

func (c *Client) Execute(ctx context.Context, r ai.Request) (ai.Result, error) {
	if c == nil || c.http == nil || c.key == "" || ctx == nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	if err := r.Validate(); err != nil {
		return ai.Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return ai.Result{}, err
	}
	// Also bound direct adapter use; an executor/caller's shorter deadline wins.
	ctx, cancel := context.WithTimeout(ctx, ai.MaxTimeout)
	defer cancel()
	data, err := json.Marshal(payload{Model: r.Model, Input: r.Text, MaxOutputTokens: r.MaxOutputTokens})
	if err != nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return ai.Result{}, ai.ErrInvalidRequest
	}
	req.ContentLength = int64(len(data))
	req.GetBody = nil
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	// Suppress the Go default user-agent; no machine identity is supplied.
	req.Header["User-Agent"] = []string{""}
	resp, err := c.http.Do(req)
	if err != nil {
		return ai.Result{}, transportError(ctx, err)
	}
	if resp == nil || resp.Body == nil {
		return ai.Result{}, ai.ErrTransport
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	closeErr := resp.Body.Close()
	if readErr != nil || closeErr != nil {
		var safe []error
		if readErr != nil {
			safe = append(safe, transportError(ctx, readErr))
		}
		if closeErr != nil {
			safe = append(safe, ai.ErrTransport)
		}
		return ai.Result{}, ai.SafeError(errors.Join(safe...))
	}
	if err := ctx.Err(); err != nil {
		return ai.Result{}, err
	}
	if len(data) > maxBody {
		return ai.Result{}, ai.ErrResponseTooLarge
	}
	if resp.StatusCode != http.StatusOK {
		return ai.Result{}, statusError(resp.StatusCode, data)
	}
	result, err := decodeResult(data)
	if err != nil {
		return ai.Result{}, err
	}
	if result.Usage != nil && result.Usage.OutputTokens > int64(r.MaxOutputTokens) {
		return ai.Result{}, ai.ErrMalformedResponse
	}
	// A provider may reflect its Authorization credential even on a 200 response.
	if strings.Contains(result.Text, c.key) || strings.Contains(result.Model, c.key) {
		return ai.Result{}, ai.ErrMalformedResponse
	}
	if err := ctx.Err(); err != nil {
		return ai.Result{}, err
	}
	return result, nil
}

func transportError(ctx context.Context, err error) error {
	if cause := ctx.Err(); cause != nil {
		return cause
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return context.DeadlineExceeded
	}
	return ai.ErrTransport
}
func statusError(status int, data []byte) error {
	switch status {
	case 401:
		return ai.ErrAuthentication
	case 403:
		return ai.ErrAuthorization
	case 429:
		object, err := jsonObject(data)
		if err == nil {
			nested, err := jsonObject(object["error"])
			if err == nil {
				code, _ := jsonString(nested["code"])
				// Exact structured codes, per the official API error-code reference.
				switch code {
				case "insufficient_quota", "credit_balance_exhausted", "organization_spend_limit_exceeded", "project_spend_limit_exceeded", "organization_usage_limit_exceeded":
					return ai.ErrQuotaExceeded
				}
			}
		}
		return ai.ErrRateLimited
	default:
		return ai.ErrProvider
	}
}

func jsonObject(data []byte) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, ai.ErrMalformedResponse
	}
	var obj map[string]json.RawMessage
	// Unmarshal requires exactly one JSON value, followed only by whitespace.
	if json.Unmarshal(data, &obj) != nil || obj == nil {
		return nil, ai.ErrMalformedResponse
	}
	return obj, nil
}
func jsonString(data []byte) (string, error) {
	var value *string
	if json.Unmarshal(data, &value) != nil || value == nil {
		return "", ai.ErrMalformedResponse
	}
	return *value, nil
}
func isNull(data []byte) bool { return bytes.Equal(bytes.TrimSpace(data), []byte("null")) }

func decodeResult(data []byte) (ai.Result, error) {
	return decodeResultWithPolicy(data, ordinaryResponse)
}

// decodeFunctionRoundTripFinalResult accepts only leading opaque reasoning
// metadata followed by exactly one completed assistant message. Reasoning is
// never returned, replayed, or admitted as a tool proposal.
func decodeFunctionRoundTripFinalResult(data []byte) (ai.Result, error) {
	return decodeResultWithPolicy(data, functionRoundTripFinalResponse)
}

type resultDecodePolicy uint8

const (
	ordinaryResponse resultDecodePolicy = iota
	functionRoundTripFinalResponse
)

func decodeResultWithPolicy(data []byte, policy resultDecodePolicy) (ai.Result, error) {
	obj, err := jsonObject(data)
	if err != nil {
		return ai.Result{}, err
	}
	status, err := jsonString(obj["status"])
	if err != nil {
		return ai.Result{}, err
	}
	switch status {
	case "incomplete", "queued", "in_progress":
		return ai.Result{}, ai.ErrIncompleteResponse
	case "failed", "cancelled":
		return ai.Result{}, ai.ErrProvider
	case "completed":
	default:
		return ai.Result{}, ai.ErrMalformedResponse
	}
	// Required nullable fields must not hide provider errors or incomplete output.
	for _, field := range []string{"error", "incomplete_details"} {
		raw, ok := obj[field]
		if !ok {
			return ai.Result{}, ai.ErrMalformedResponse
		}
		if !isNull(raw) {
			details, err := jsonObject(raw)
			if err != nil {
				return ai.Result{}, err
			}
			if field == "error" {
				return ai.Result{}, ai.ErrProvider
			}
			if _, err := jsonString(details["reason"]); err != nil {
				return ai.Result{}, err
			}
			return ai.Result{}, ai.ErrIncompleteResponse
		}
	}
	model, err := jsonString(obj["model"])
	if err != nil {
		return ai.Result{}, err
	}
	var items []json.RawMessage
	if json.Unmarshal(obj["output"], &items) != nil || len(items) == 0 {
		return ai.Result{}, ai.ErrMalformedResponse
	}
	var text strings.Builder
	for i, raw := range items {
		item, err := jsonObject(raw)
		if err != nil {
			return ai.Result{}, err
		}
		kind, err := jsonString(item["type"])
		if err != nil {
			return ai.Result{}, err
		}
		if policy == functionRoundTripFinalResponse && i < len(items)-1 {
			if kind != "reasoning" {
				return ai.Result{}, ai.ErrMalformedResponse
			}
			continue
		}
		if kind != "message" {
			return ai.Result{}, ai.ErrMalformedResponse
		}
		role, err := jsonString(item["role"])
		if err != nil || role != "assistant" {
			return ai.Result{}, ai.ErrMalformedResponse
		}
		state, err := jsonString(item["status"])
		if err != nil {
			return ai.Result{}, err
		}
		if state == "incomplete" || state == "in_progress" || state == "queued" {
			return ai.Result{}, ai.ErrIncompleteResponse
		}
		if state != "completed" {
			return ai.Result{}, ai.ErrMalformedResponse
		}
		var content []json.RawMessage
		if json.Unmarshal(item["content"], &content) != nil || len(content) == 0 {
			return ai.Result{}, ai.ErrMalformedResponse
		}
		for _, raw := range content {
			part, err := jsonObject(raw)
			if err != nil {
				return ai.Result{}, err
			}
			kind, err := jsonString(part["type"])
			if err != nil {
				return ai.Result{}, err
			}
			if kind == "refusal" {
				return ai.Result{}, ai.ErrRefused
			}
			if kind != "output_text" {
				return ai.Result{}, ai.ErrMalformedResponse
			}
			value, err := jsonString(part["text"])
			if err != nil {
				return ai.Result{}, err
			}
			if len(value) > ai.MaxTextBytes-text.Len() {
				return ai.Result{}, ai.ErrResponseTooLarge
			}
			text.WriteString(value)
		}
	}
	result := ai.Result{Text: text.String(), Model: model}
	if raw, ok := obj["usage"]; ok && !isNull(raw) {
		usage, err := jsonObject(raw)
		if err != nil {
			return ai.Result{}, err
		}
		var in, out *int64
		if json.Unmarshal(usage["input_tokens"], &in) != nil || in == nil || json.Unmarshal(usage["output_tokens"], &out) != nil || out == nil {
			return ai.Result{}, ai.ErrMalformedResponse
		}
		result.Usage = &ai.Usage{InputTokens: *in, OutputTokens: *out}
	}
	if err := result.Validate(); err != nil {
		return ai.Result{}, err
	}
	return result, nil
}
