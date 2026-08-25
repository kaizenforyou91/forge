package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

// RunningProcess owns the wait/reap lifecycle for one directly invoked child.
// The underlying command, process, executable path, and lease remain private.
type RunningProcess struct {
	pid         int
	entrypoint  compiler.RuntimeEntrypoint
	signerKeyID string

	cmd         *exec.Cmd
	ctx         context.Context
	lease       *executableLease
	termination *processTerminationControl
	stdout      *boundedOutputWriter
	stderr      *boundedOutputWriter

	done chan struct{}
	mu   sync.Mutex

	result  ProcessResult
	waitErr error

	terminateOnce sync.Once
	terminateErr  error
}

type processTerminationCause uint8

const (
	processTerminationCauseNone processTerminationCause = iota
	processTerminationCauseManual
	processTerminationCauseCancellation
)

type processTerminationControl struct {
	mu sync.Mutex

	completed bool
	winner    processTerminationCause
	kill      func() error
}

func (p *RunningProcess) PID() int {
	if p == nil {
		return 0
	}
	return p.pid
}

func (p *RunningProcess) Entrypoint() compiler.RuntimeEntrypoint {
	if p == nil {
		return compiler.RuntimeEntrypoint{}
	}
	return p.entrypoint
}

func (p *RunningProcess) SignerKeyID() string {
	if p == nil {
		return ""
	}
	return p.signerKeyID
}

// Terminate requests immediate termination of the direct child. The
// background waiter remains the sole owner of process reaping and lease
// release. Repeated and concurrent calls are idempotent.
func (p *RunningProcess) Terminate() error {
	if p == nil || p.termination == nil {
		return fmt.Errorf(
			"%w: running process is nil or incomplete",
			ErrProcessTerminationFailed,
		)
	}

	p.terminateOnce.Do(func() {
		err := p.termination.request(processTerminationCauseManual)
		if err == nil || errors.Is(err, os.ErrProcessDone) {
			return
		}
		p.terminateErr = fmt.Errorf(
			"%w: kill direct child: %w",
			ErrProcessTerminationFailed,
			err,
		)
	})
	return p.terminateErr
}

// Wait may be called repeatedly or concurrently. Exactly one background
// waiter owns cmd.Wait; callers receive defensive copies of cached output.
func (p *RunningProcess) Wait() (ProcessResult, error) {
	if p == nil || p.done == nil {
		return ProcessResult{}, fmt.Errorf(
			"%w: running process is nil or incomplete",
			ErrProcessWaitFailed,
		)
	}

	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.result.clone(), p.waitErr
}

func (p *RunningProcess) waitInBackground() {
	waitErr := p.cmd.Wait()
	winner := p.termination.complete()
	result := ProcessResult{ExitCode: -1}
	if p.cmd.ProcessState != nil {
		result.ExitCode = p.cmd.ProcessState.ExitCode()
	}
	result.Stdout, result.StdoutTruncated = p.stdout.snapshot()
	result.Stderr, result.StderrTruncated = p.stderr.snapshot()

	var resultErr error
	switch {
	case waitErr == nil:
		// A completed zero exit wins over a kill that raced too late.
	case winner == processTerminationCauseCancellation:
		result.Canceled = true
		contextErr := p.ctx.Err()
		if contextErr == nil {
			contextErr = context.Canceled
		}
		resultErr = contextErr
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) && !errors.Is(waitErr, contextErr) {
			resultErr = errors.Join(
				resultErr,
				fmt.Errorf("%w: wait after cancellation: %w", ErrProcessWaitFailed, waitErr),
			)
		}
	case winner == processTerminationCauseManual:
		result.Terminated = true
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			resultErr = fmt.Errorf(
				"%w: wait after manual termination: %w",
				ErrProcessWaitFailed,
				waitErr,
			)
		}
	default:
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			resultErr = fmt.Errorf(
				"%w: wait for direct child: %w",
				ErrProcessWaitFailed,
				waitErr,
			)
		}
	}

	if releaseErr := p.lease.release(); releaseErr != nil {
		if resultErr == nil {
			resultErr = releaseErr
		} else {
			resultErr = errors.Join(resultErr, releaseErr)
		}
	}

	p.mu.Lock()
	p.result = result.clone()
	p.waitErr = resultErr
	close(p.done)
	p.mu.Unlock()
}

func (c *processTerminationControl) request(cause processTerminationCause) error {
	if c == nil {
		return os.ErrProcessDone
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.completed {
		return os.ErrProcessDone
	}
	if c.winner != processTerminationCauseNone {
		if cause == processTerminationCauseCancellation {
			// Tell os/exec that cancellation did not win after another kill was
			// already successfully initiated.
			return os.ErrProcessDone
		}
		return nil
	}
	if c.kill == nil {
		return fmt.Errorf("direct-child kill function is unavailable")
	}

	err := c.kill()
	if err != nil {
		return err
	}
	c.winner = cause
	return nil
}

func (c *processTerminationControl) complete() processTerminationCause {
	if c == nil {
		return processTerminationCauseNone
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.completed = true
	return c.winner
}
