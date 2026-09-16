package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

// RunningProcess owns one direct child and its platform-bounded native scope.
// The underlying command, process, executable path, and lease remain private.
type RunningProcess struct {
	pid         int
	entrypoint  compiler.RuntimeEntrypoint
	signerKeyID string

	execution   processExecution
	scope       *processScopeOwner
	watchStop   chan struct{}
	watchDone   chan struct{}
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

	completed           bool
	winner              processTerminationCause
	control             func(processScopeControlCause) error
	cancellationFailure error
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

// Terminate requests immediate termination of the owned scope. The
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
			"%w: terminate owned scope: %w",
			ErrProcessTerminationFailed,
			err,
		)
	})
	return p.terminateErr
}

// Wait may be called repeatedly or concurrently. Exactly one background
// waiter owns native wait completion; callers receive defensive copies of cached output.
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

func (p *RunningProcess) watchCancellation() {
	defer close(p.watchDone)
	select {
	case <-p.ctx.Done():
		_ = p.termination.request(processTerminationCauseCancellation)
	case <-p.watchStop:
	}
}

func (p *RunningProcess) waitInBackground() {
	observeErr := p.execution.observe()
	winner := p.termination.complete()
	close(p.watchStop)
	<-p.watchDone
	var cleanupErr error
	if winner == processTerminationCauseNone {
		cleanupErr = naturalScopeCleanup(p.scope)
	}
	finalizeErr := p.scope.finalize()
	code, finishErr := p.execution.finish()
	result := ProcessResult{ExitCode: code}
	result.Stdout, result.StdoutTruncated = p.stdout.snapshot()
	result.Stderr, result.StderrTruncated = p.stderr.snapshot()
	var resultErr error
	// A zero direct-child exit wins over a successful control request that
	// raced too late. Cleanup infrastructure errors remain observable below.
	if code != 0 {
		switch winner {
		case processTerminationCauseCancellation:
			result.Canceled = true
			resultErr = p.ctx.Err()
			if resultErr == nil {
				resultErr = context.Canceled
			}
		case processTerminationCauseManual:
			result.Terminated = true
		}
	}
	if err := errors.Join(observeErr, cleanupErr, finalizeErr, finishErr); err != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("%w: complete owned process: %w", ErrProcessWaitFailed, err))
	}
	// The watcher is joined and admission is closed, so this evidence is stable.
	if err := p.termination.cancellationFailure; err != nil {
		resultErr = errors.Join(resultErr, p.ctx.Err(),
			fmt.Errorf("%w: %w: cancellation control: %w", ErrProcessWaitFailed, ErrProcessTerminationFailed, err))
	}
	resultErr = errors.Join(resultErr, p.lease.release())
	p.mu.Lock()
	p.result, p.waitErr = result.clone(), resultErr
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
			// Cancellation cannot replace an earlier successful control winner.
			return os.ErrProcessDone
		}
		return nil
	}
	if c.control == nil {
		return fmt.Errorf("owned scope control is unavailable")
	}

	scopeCause := processScopeManualTermination
	if cause == processTerminationCauseCancellation {
		scopeCause = processScopeCancellation
	}
	err := c.control(scopeCause)
	if err != nil {
		if cause == processTerminationCauseCancellation && !errors.Is(err, os.ErrProcessDone) {
			c.cancellationFailure = err
		}
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
