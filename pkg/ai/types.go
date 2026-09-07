// Package ai executes one explicit text request through a caller-supplied provider.
// It has no network, credential, filesystem, or subprocess dependencies.
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxInputBytes   = 16 * 1024
	MaxModelBytes   = 128
	MinOutputTokens = 16
	MaxOutputTokens = 2048
	MaxTextBytes    = 64 * 1024
	MaxTimeout      = 120 * time.Second
)

// Request requires concrete values. The core does not supply defaults or rewrite
// Text. CLI defaults are 1024 tokens and a 30-second executor timeout.
type Request struct {
	Text            string
	Model           string
	MaxOutputTokens int
}

// Usage is optional; nil means the provider did not report usage.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
}

// Result represents complete text only. On any failure Execute returns Result{}.
type Result struct {
	Text  string
	Model string
	Usage *Usage
}

// Provider must honor ctx, finish its owned work before returning, and return
// only completed results. The executor cannot forcibly stop a broken provider.
type Provider interface {
	Execute(context.Context, Request) (Result, error)
}

// Validate checks the same request rules used by the executor, adapter and CLI.
func (r Request) Validate() error {
	if len(r.Text) > MaxInputBytes || !utf8.ValidString(r.Text) || strings.TrimSpace(r.Text) == "" {
		return fmt.Errorf("%w: text must be nonblank UTF-8 within 16384 bytes", ErrInvalidRequest)
	}
	if !validModel(r.Model) {
		return fmt.Errorf("%w: model must be an explicit ASCII identifier within 128 bytes", ErrInvalidRequest)
	}
	if r.MaxOutputTokens < MinOutputTokens || r.MaxOutputTokens > MaxOutputTokens {
		return fmt.Errorf("%w: output token limit must be 16..2048", ErrInvalidRequest)
	}
	return nil
}

// ValidateTimeout requires an explicit positive timeout of at most 120 seconds.
func ValidateTimeout(timeout time.Duration) error {
	if timeout <= 0 || timeout > MaxTimeout {
		return fmt.Errorf("%w: timeout must be positive and at most 120 seconds", ErrInvalidRequest)
	}
	return nil
}

func validModel(model string) bool {
	if len(model) == 0 || len(model) > MaxModelBytes {
		return false
	}
	for i, c := range []byte(model) {
		alphaNum := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		if !alphaNum && (i == 0 || c != '.' && c != '_' && c != ':' && c != '-') {
			return false
		}
	}
	return true
}

// Validate rejects invalid results, including terminal control sequences.
// Only TAB and LF are allowed among C0 controls; DEL and all C1 are rejected.
func (r Result) Validate() error {
	if len(r.Text) > MaxTextBytes {
		return ErrResponseTooLarge
	}
	if !utf8.ValidString(r.Text) || strings.TrimSpace(r.Text) == "" || !validModel(r.Model) {
		return ErrMalformedResponse
	}
	for _, c := range r.Text {
		if c < 0x20 && c != '\t' && c != '\n' || c >= 0x7f && c <= 0x9f {
			return ErrMalformedResponse
		}
	}
	if r.Usage != nil && (r.Usage.InputTokens < 0 || r.Usage.OutputTokens < 0) {
		return ErrMalformedResponse
	}
	return nil
}
