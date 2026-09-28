//go:build linux

package runtime

import (
	"context"
	"os/exec"

	"golang.org/x/sys/unix"
)

type linuxProcessExecution struct {
	cmd   *exec.Cmd
	scope *linuxProcessScopePlatform
}

func startProcessExecution(ctx context.Context, path, directory string, stdout, stderr *boundedOutputWriter) (processExecution, *processScopeOwner, processStartLeaseDisposition, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, processStartLeaseReleaseSafe, err
	}
	cmd := processCommand(path, directory, stdout, stderr)
	prepared, err := prepareLinuxProcessScopeCommand(cmd)
	if err != nil {
		return nil, nil, processStartLeaseReleaseSafe, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, processStartLeaseReleaseSafe, err
	}
	if err = cmd.Start(); err != nil {
		return nil, nil, processStartLeaseReleaseSafe, err
	}
	// cmd/receipt are exclusive locals: no attribute mutation, receipt reuse,
	// or reaper can invalidate the accepted B2 binding after successful Start.
	scope, err := newLinuxProcessScopePlatform(prepared)
	if err != nil {
		// Start succeeded with our exclusive pre-exec receipt and no reaper.
		// Even a rejected bookkeeping bind must not abandon that known child.
		// Use the same B2 platform solely for synchronous failure cleanup; its
		// identity is still this unreaped child's PID, never caller input.
		cleanup := &linuxProcessScopePlatform{pgid: cmd.Process.Pid, kill: unix.Kill, waitid: unix.Waitid}
		e := &linuxProcessExecution{cmd: cmd, scope: cleanup}
		disposition, cleanupErr := completeFailedProcessExecution(e, cleanup, err)
		return nil, nil, disposition, cleanupErr
	}
	e := &linuxProcessExecution{cmd: cmd, scope: scope}
	owner, disposition, err := admitProcessExecution(e, scope)
	if err != nil {
		return nil, nil, disposition, err
	}
	return e, owner, processStartLeaseDispositionUnset, nil
}

func (e *linuxProcessExecution) pid() int       { return e.cmd.Process.Pid }
func (e *linuxProcessExecution) observe() error { return e.scope.waitLeaderExitNoReap() }
func (e *linuxProcessExecution) prepareFinalize() (int, bool, error) {
	return -1, false, nil
}
func (e *linuxProcessExecution) finish() (int, error) {
	return commandExit(e.cmd, e.cmd.Wait())
}

func (e *linuxProcessExecution) completed() (int, int, bool) {
	if e.cmd.ProcessState == nil {
		return 0, -1, false
	}
	return e.cmd.ProcessState.Pid(), e.cmd.ProcessState.ExitCode(), true
}

func (e *linuxProcessExecution) safeToReleaseLease() bool {
	return e.cmd.ProcessState != nil
}

func (e *linuxProcessExecution) terminalEvidence() processTerminalEvidence {
	return processTerminalEvidence{outputJoined: e.cmd.ProcessState != nil}
}
