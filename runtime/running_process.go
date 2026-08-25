package runtime

import (
	"context"
	"errors"
	"fmt"
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

	cmd    *exec.Cmd
	ctx    context.Context
	lease  *executableLease
	stdout *boundedOutputWriter
	stderr *boundedOutputWriter

	done chan struct{}
	mu   sync.Mutex

	result  ProcessResult
	waitErr error
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
	result := ProcessResult{ExitCode: -1}
	if p.cmd.ProcessState != nil {
		result.ExitCode = p.cmd.ProcessState.ExitCode()
	}
	result.Stdout, result.StdoutTruncated = p.stdout.snapshot()
	result.Stderr, result.StderrTruncated = p.stderr.snapshot()

	var resultErr error
	contextErr := p.ctx.Err()
	switch {
	case waitErr == nil:
		// A completed zero exit wins over later context cancellation.
	case contextErr != nil:
		result.Canceled = true
		resultErr = contextErr
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
