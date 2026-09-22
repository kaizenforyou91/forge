//go:build !linux && !windows

package runtime

import (
	"context"
	"os"
	"os/exec"
)

// Other GOOS retain exactly the historical direct-child guarantee.
type directProcessExecution struct {
	cmd     *exec.Cmd
	waitErr error
}
type directProcessScope struct{ process *os.Process }

func (s *directProcessScope) terminate() error { return s.process.Kill() }
func (s *directProcessScope) finalize() error  { s.process = nil; return nil }

func startProcessExecution(ctx context.Context, path, directory string, stdout, stderr *boundedOutputWriter) (processExecution, *processScopeOwner, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	cmd := processCommand(path, directory, stdout, stderr)
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	e := &directProcessExecution{cmd: cmd}
	owner, err := admitProcessExecution(e, &directProcessScope{process: cmd.Process})
	if err != nil {
		return nil, nil, err
	}
	return e, owner, nil
}
func (e *directProcessExecution) pid() int       { return e.cmd.Process.Pid }
func (e *directProcessExecution) observe() error { e.waitErr = e.cmd.Wait(); return nil }
func (e *directProcessExecution) prepareFinalize() (int, bool, error) {
	return -1, false, nil
}
func (e *directProcessExecution) finish() (int, error) { return commandExit(e.cmd, e.waitErr) }

func (e *directProcessExecution) completed() (int, int, bool) {
	if e.cmd.ProcessState == nil {
		return 0, -1, false
	}
	return e.cmd.ProcessState.Pid(), e.cmd.ProcessState.ExitCode(), true
}

func (e *directProcessExecution) safeToReleaseLease() bool {
	return e.cmd.ProcessState != nil
}

func (e *directProcessExecution) terminalEvidence() processTerminalEvidence {
	return processTerminalEvidence{outputJoined: e.cmd.ProcessState != nil}
}
