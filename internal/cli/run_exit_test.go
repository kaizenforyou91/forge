package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestExitCodePolicy(t *testing.T) {
	childCause := errors.New("child returned nonzero")
	childExit := newExitStatusError(23, childCause)
	cleanupFailure := errors.New("cleanup failed")

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", err: nil, want: 0},
		{name: "pure child exit", err: childExit, want: 23},
		{name: "nested pure child exit", err: fmt.Errorf("run failed: %w", childExit), want: 23},
		{name: "pure cancellation", err: context.Canceled, want: 130},
		{name: "nested pure cancellation", err: fmt.Errorf("run canceled: %w", context.Canceled), want: 130},
		{name: "generic error", err: errors.New("infrastructure failed"), want: 1},
		{name: "child plus cleanup", err: errors.Join(childExit, cleanupFailure), want: 1},
		{name: "cancellation plus cleanup", err: errors.Join(context.Canceled, cleanupFailure), want: 1},
		{name: "wrapped child plus cleanup", err: fmt.Errorf("run failed: %w", errors.Join(childExit, cleanupFailure)), want: 1},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ExitCode(test.err); got != test.want {
				t.Fatalf("expected exit code %d, got %d for %v", test.want, got, test.err)
			}
		})
	}

	if !errors.Is(childExit, childCause) {
		t.Fatal("typed exit status does not preserve its cause")
	}
	if !strings.Contains(childExit.Error(), "status 23") || !strings.Contains(childExit.Error(), childCause.Error()) {
		t.Fatalf("typed exit status message lacks child evidence: %q", childExit.Error())
	}
}

func TestExitStatusRejectsUnavailableOrInvalidExactCodes(t *testing.T) {
	if err := newExitStatusError(0, nil); err != nil {
		t.Fatalf("exit status zero must not be an error: %v", err)
	}

	for _, code := range []int{-1, 0, 256} {
		t.Run(fmt.Sprintf("code_%d", code), func(t *testing.T) {
			err := newExitStatusError(code, errors.New("status unavailable"))
			if err == nil {
				t.Fatal("expected invalid status to remain a generic error")
			}
			if got := ExitCode(err); got != 1 {
				t.Fatalf("expected generic exit code 1, got %d", got)
			}
		})
	}
}

func TestExitStatusSupportsFullNonzeroProcessRange(t *testing.T) {
	for _, code := range []int{1, 127, 255} {
		err := newExitStatusError(code, nil)
		if err == nil {
			t.Fatalf("expected typed error for exit code %d", code)
		}
		if got := ExitCode(err); got != code {
			t.Fatalf("expected exit code %d, got %d", code, got)
		}
	}
}

func TestExistingCLIErrorStillMapsToGenericFailure(t *testing.T) {
	root := NewRootCommand()
	root.SetArgs([]string{"unknown-command"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if got := ExitCode(err); got != 1 {
		t.Fatalf("expected existing CLI error to map to exit code 1, got %d", got)
	}
}
