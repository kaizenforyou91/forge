//go:build windows

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

type windowsProcessExecution struct {
	process   windows.Handle
	processID int
	exitCode  int
	finished  bool
	readers   [2]*os.File
	drained   chan error
}

func waitWindowsExecution(process windows.Handle) error {
	state, err := windows.WaitForSingleObject(process, windows.INFINITE)
	if err != nil {
		return err
	}
	if state != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("unexpected direct-process wait outcome")
	}
	return nil
}

// Preserve the B3 launch algorithm. Its existing per-call close seam composes
// B4's direct-child completion obligation on post-create failure, before B3
// relinquishes the process handle. This adds no second control authority.
func startWindowsExecution(ctx context.Context, spec *windowsProcessScopeLaunchSpec, n windowsScopeNative) (*windowsProcessScopeLaunch, error) {
	var process windows.Handle
	create, closeHandle := n.create, n.close
	n.create = func(app, cmd *uint16, inherit bool, flags uint32, env, dir *uint16, si *windows.StartupInfo, pi *windows.ProcessInformation) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := create(app, cmd, inherit, flags, env, dir, si, pi)
		process = pi.Process
		return err
	}
	n.close = func(h windows.Handle) error {
		var waitErr error
		if process != 0 && h == process {
			waitErr = waitWindowsExecution(h)
		}
		return errors.Join(waitErr, closeHandle(h))
	}
	return startWindowsProcessInScopeWith(spec, n)
}

func startProcessExecution(ctx context.Context, path, directory string, stdout, stderr *boundedOutputWriter) (processExecution, *processScopeOwner, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return nil, nil, err
	}
	var readers, writers [2]*os.File
	closeFiles := func() error {
		var errs []error
		if stdin != nil {
			errs = append(errs, stdin.Close())
			stdin = nil
		}
		for i := range readers {
			if readers[i] != nil {
				errs = append(errs, readers[i].Close())
				readers[i] = nil
			}
			if writers[i] != nil {
				errs = append(errs, writers[i].Close())
				writers[i] = nil
			}
		}
		return errors.Join(errs...)
	}
	for i := range readers {
		readers[i], writers[i], err = os.Pipe()
		if err != nil {
			return nil, nil, errors.Join(err, closeFiles())
		}
	}
	launch, err := startWindowsExecution(ctx, &windowsProcessScopeLaunchSpec{
		executable: path, commandLine: windows.EscapeArg(path), directory: directory,
		environment: runtimeProcessEnvironment(directory),
		stdio:       [3]windows.Handle{windows.Handle(stdin.Fd()), windows.Handle(writers[0].Fd()), windows.Handle(writers[1].Fd())},
	}, windowsScopeSystem())
	if err != nil {
		return nil, nil, errors.Join(err, closeFiles())
	}
	e := &windowsProcessExecution{process: launch.process, readers: readers, drained: make(chan error, 2)}
	// Drainers own the read endpoints through finish. Parent write copies must
	// close even on failure; otherwise Forge itself would prevent EOF.
	for i, dst := range []*boundedOutputWriter{stdout, stderr} {
		r := readers[i]
		go func() { _, err := io.Copy(dst, r); e.drained <- err }()
		readers[i] = nil
	}
	err = closeFiles()
	pid, pidErr := windows.GetProcessId(launch.process)
	e.processID = int(pid)
	err = errors.Join(err, pidErr)
	if err != nil {
		controlErr := launch.scope.terminate()
		observeErr := e.observe()
		finalizeErr := launch.scope.finalize()
		_, finishErr := e.finish()
		return nil, nil, errors.Join(err, controlErr, observeErr, finalizeErr, finishErr)
	}
	owner, err := admitProcessExecution(e, launch.scope)
	if err != nil {
		return nil, nil, err
	}
	return e, owner, nil
}

func (e *windowsProcessExecution) pid() int       { return e.processID }
func (e *windowsProcessExecution) observe() error { return waitWindowsExecution(e.process) }
func (e *windowsProcessExecution) finish() (int, error) {
	var code uint32
	err := windows.GetExitCodeProcess(e.process, &code)
	exitCode := -1
	if err == nil {
		exitCode = int(code)
	}
	timer := time.NewTimer(runtimeProcessWaitDelay)
	defer timer.Stop()
	remaining := 2
	for remaining > 0 {
		select {
		case drainErr := <-e.drained:
			err = errors.Join(err, drainErr)
			remaining--
		case <-timer.C:
			err = errors.Join(err, errors.New("bounded output completion deadline exceeded"))
			for i, r := range e.readers {
				err = errors.Join(err, r.Close())
				e.readers[i] = nil
			}
			// Closing owned readers unblocks io.Copy. Join both before release.
			for remaining > 0 {
				err = errors.Join(err, <-e.drained)
				remaining--
			}
		}
	}
	for i, r := range e.readers {
		if r != nil {
			err = errors.Join(err, r.Close())
			e.readers[i] = nil
		}
	}
	err = errors.Join(err, windows.CloseHandle(e.process))
	e.process = 0
	e.exitCode, e.finished = exitCode, true
	return exitCode, err
}

func (e *windowsProcessExecution) completed() (int, int, bool) {
	return e.processID, e.exitCode, e.finished
}
