//go:build windows

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsProcessExecution struct {
	process    windows.Handle
	processID  int
	exitCode   int
	resultRead bool
	finished   bool
	scope      *windowsProcessScopePlatform
	cleanup    *windowsStartCleanup
	readers    [2]*os.File
	drained    chan error
}

type windowsStartCleanup struct {
	mu              sync.Mutex
	job             windows.Handle
	process         windows.Handle
	processClosed   bool
	jobClosePending bool
	jobQuiescent    bool
	processCloses   int
	jobCloses       int
	processCloseErr error
	jobCloseErr     error
	closeHandle     func(windows.Handle) error
}

type windowsJobAccounting struct {
	totalUserTime             int64
	totalKernelTime           int64
	thisPeriodTotalUserTime   int64
	thisPeriodTotalKernelTime int64
	totalPageFaultCount       uint32
	totalProcesses            uint32
	activeProcesses           uint32
	totalTerminatedProcesses  uint32
}

const windowsJobQuiescenceWaitMilliseconds = 20000

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

func waitWindowsJobQuiescence(job windows.Handle) error {
	state, err := windows.WaitForSingleObject(job, windowsJobQuiescenceWaitMilliseconds)
	if err != nil {
		return err
	}
	if state != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("Windows Job did not reach native quiescence: wait outcome %d", state)
	}
	var accounting windowsJobAccounting
	err = windows.QueryInformationJobObject(
		job,
		windows.JobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&accounting)),
		uint32(unsafe.Sizeof(accounting)),
		nil,
	)
	if err == nil && accounting.activeProcesses != 0 {
		return errors.New("Windows Job remained active after native completion")
	}
	return err
}

func (c *windowsStartCleanup) closeProcess(process windows.Handle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.processClosed {
		return nil
	}
	err := c.closeHandle(process)
	c.processClosed = true
	c.processCloses++
	c.processCloseErr = err
	return err
}

func (c *windowsStartCleanup) closeForB3(handle windows.Handle) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if handle == c.job {
		if c.process != 0 && !c.processClosed {
			c.jobClosePending = true
			return nil
		}
		err := c.closeHandle(handle)
		c.jobCloses++
		c.jobCloseErr = err
		return err
	}
	if handle != c.process {
		return c.closeHandle(handle)
	}
	err := errors.Join(waitWindowsExecution(handle), c.closeHandle(handle))
	c.processClosed = true
	c.processCloses++
	c.processCloseErr = err
	quiescenceErr := waitWindowsJobQuiescence(c.job)
	err = errors.Join(err, quiescenceErr)
	c.jobQuiescent = quiescenceErr == nil
	if c.jobClosePending {
		closeErr := c.closeHandle(c.job)
		err = errors.Join(err, closeErr)
		c.jobCloses++
		c.jobCloseErr = closeErr
		c.jobClosePending = false
	}
	return err
}

// Preserve the B3 launch algorithm. Its existing per-call close seam composes
// B4's direct-child completion obligation on post-create failure, before B3
// relinquishes the process handle. This adds no second control authority.
func startWindowsExecution(ctx context.Context, spec *windowsProcessScopeLaunchSpec, n windowsScopeNative) (*windowsProcessScopeLaunch, *windowsStartCleanup, error) {
	cleanup := &windowsStartCleanup{closeHandle: n.close}
	newJob := n.newJob
	n.newJob = func() (windows.Handle, error) {
		job, err := newJob()
		cleanup.job = job
		return job, err
	}
	create, closeHandle := n.create, n.close
	n.create = func(app, cmd *uint16, inherit bool, flags uint32, env, dir *uint16, si *windows.StartupInfo, pi *windows.ProcessInformation) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := create(app, cmd, inherit, flags, env, dir, si, pi)
		cleanup.process = pi.Process
		return err
	}
	cleanup.closeHandle = closeHandle
	n.close = cleanup.closeForB3
	launch, err := startWindowsProcessInScopeWith(spec, n)
	return launch, cleanup, err
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
	launch, cleanup, err := startWindowsExecution(ctx, &windowsProcessScopeLaunchSpec{
		executable: path, commandLine: windows.EscapeArg(path), directory: directory,
		environment: runtimeProcessEnvironment(directory),
		stdio:       [3]windows.Handle{windows.Handle(stdin.Fd()), windows.Handle(writers[0].Fd()), windows.Handle(writers[1].Fd())},
	}, windowsScopeSystem())
	if err != nil {
		return nil, nil, errors.Join(err, closeFiles())
	}
	e := &windowsProcessExecution{
		process: launch.process, scope: launch.scope,
		cleanup: cleanup, readers: readers, drained: make(chan error, 2),
	}
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

// prepareFinalize captures the direct-child result, releases Forge's process
// handle, then proves the still-owned Job has no active processes. Job signaling
// is corroborated with native accounting; control success, elapsed time and pipe
// EOF are never treated as scope quiescence.
func (e *windowsProcessExecution) prepareFinalize() (int, bool, error) {
	var code uint32
	err := windows.GetExitCodeProcess(e.process, &code)
	exitCode := -1
	if err == nil {
		exitCode = int(code)
		e.exitCode, e.resultRead = exitCode, true
	}
	err = errors.Join(err, e.cleanup.closeProcess(e.process))
	e.process = 0

	e.scope.mu.Lock()
	job := e.scope.job
	e.scope.mu.Unlock()
	if job == 0 {
		return exitCode, e.resultRead, errors.Join(err, errors.New("Windows Job authority unavailable before quiescence"))
	}
	quiescenceErr := waitWindowsJobQuiescence(job)
	err = errors.Join(err, quiescenceErr)
	e.cleanup.mu.Lock()
	e.cleanup.jobQuiescent = quiescenceErr == nil
	e.cleanup.mu.Unlock()
	return exitCode, e.resultRead, err
}

func (e *windowsProcessExecution) finish() (int, error) {
	var err error
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
	e.finished = true
	return e.exitCode, err
}

func (e *windowsProcessExecution) completed() (int, int, bool) {
	return e.processID, e.exitCode, e.finished
}

func (e *windowsProcessExecution) safeToReleaseLease() bool {
	e.cleanup.mu.Lock()
	defer e.cleanup.mu.Unlock()
	return e.resultRead && e.finished && e.cleanup.processClosed &&
		e.cleanup.processCloses == 1 && e.cleanup.processCloseErr == nil &&
		e.cleanup.jobQuiescent && e.cleanup.jobCloses == 1 && e.cleanup.jobCloseErr == nil
}

func (e *windowsProcessExecution) terminalEvidence() processTerminalEvidence {
	e.cleanup.mu.Lock()
	defer e.cleanup.mu.Unlock()
	return processTerminalEvidence{
		directProcessClosed: e.cleanup.processClosed,
		directProcessCloses: e.cleanup.processCloses,
		directProcessError:  e.cleanup.processCloseErr,
		jobQuiescent:        e.cleanup.jobQuiescent,
		jobFinalized:        e.cleanup.jobCloses == 1 && e.cleanup.jobCloseErr == nil,
		jobFinalizes:        e.cleanup.jobCloses,
		jobFinalizeError:    e.cleanup.jobCloseErr,
		outputJoined:        e.finished && len(e.drained) == 0,
	}
}
