package runtime

import (
	"errors"
	"os"
	"os/exec"
)

// processExecution separates direct-child observation from final wait/output
// ownership. Only the RunningProcess waiter calls observe and finish, once.
// In particular, Linux finish must follow scope retirement, never precede it.
type processExecution interface {
	pid() int
	observe() error
	finish() (int, error)
	completed() (int, int, bool)
}

func processCommand(path, directory string, stdout, stderr *boundedOutputWriter) *exec.Cmd {
	c := exec.Command(path)
	c.Dir, c.Env = directory, runtimeProcessEnvironment(directory)
	c.Stdout, c.Stderr = stdout, stderr
	c.WaitDelay = runtimeProcessWaitDelay
	return c
}

func commandExit(c *exec.Cmd, err error) (int, error) {
	code := -1
	if c.ProcessState != nil {
		code = c.ProcessState.ExitCode()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		err = nil // Nonzero direct-child exit is an application result.
	}
	return code, err
}

// admitProcessExecution has exclusive access to a fresh platform and owner.
// Even a bookkeeping failure after native creation retains synchronous wait
// ownership. No cancellation watcher exists until this function succeeds.
func admitProcessExecution(e processExecution, platform processScopePlatform) (*processScopeOwner, error) {
	owner, err := newProcessScopeOwner(platform)
	if err == nil {
		err = owner.activate()
	}
	if err == nil {
		return owner, nil
	}
	controlErr := platform.terminate()
	observeErr := e.observe()
	finalizeErr := platform.finalize()
	_, finishErr := e.finish()
	return nil, errors.Join(err, controlErr, observeErr, finalizeErr, finishErr)
}

func naturalScopeCleanup(owner *processScopeOwner) error {
	err := owner.request(processScopeNaturalExitCleanup)
	if errors.Is(err, os.ErrProcessDone) {
		return nil // Absent group is not a successful control winner.
	}
	return err
}
