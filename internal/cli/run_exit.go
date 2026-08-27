package cli

import (
	"context"
	"fmt"
)

const (
	genericFailureExitCode = 1
	canceledExitCode       = 130
	maxProcessExitCode     = 255
)

type exitStatusError struct {
	code int
	err  error
}

func newExitStatusError(code int, err error) error {
	if code == 0 && err == nil {
		return nil
	}
	if code < 1 || code > maxProcessExitCode {
		if err == nil {
			return fmt.Errorf("invalid process exit status %d", code)
		}
		return fmt.Errorf("invalid process exit status %d: %w", code, err)
	}
	if err == nil {
		err = fmt.Errorf("child process exited with status %d", code)
	}

	return &exitStatusError{code: code, err: err}
}

func (e *exitStatusError) Error() string {
	if e == nil {
		return "child process exit status is unavailable"
	}
	if e.err == nil {
		return fmt.Sprintf("child process exited with status %d", e.code)
	}
	return fmt.Sprintf("child process exited with status %d: %v", e.code, e.err)
}

func (e *exitStatusError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// ExitCode maps a CLI result to the process exit code used by cmd/forge.
// Exact child and cancellation statuses are preserved only when the complete
// error tree represents that outcome without a joined infrastructure failure.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	code, pure := pureCLIExitCode(err)
	if pure {
		return code
	}
	return genericFailureExitCode
}

func pureCLIExitCode(err error) (int, bool) {
	if err == nil {
		return 0, true
	}

	if status, ok := err.(*exitStatusError); ok {
		if status != nil && status.code >= 1 && status.code <= maxProcessExitCode {
			return status.code, true
		}
		return 0, false
	}
	if err == context.Canceled {
		return canceledExitCode, true
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) != 1 {
			return 0, false
		}
		return pureCLIExitCode(children[0])
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		child := wrapped.Unwrap()
		if child == nil {
			return 0, false
		}
		return pureCLIExitCode(child)
	}

	return 0, false
}
