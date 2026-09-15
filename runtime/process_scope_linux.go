//go:build linux

package runtime

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// linuxProcessScopeCommand is a single-use preparation receipt, not a launcher.
// The trusted caller exclusively owns cmd and must not mutate its attributes,
// reap its child, or install another waiter during the scope-control lifetime.
// Use by pointer; do not copy. No production runner consumes this receipt yet.
type linuxProcessScopeCommand struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	attrs *syscall.SysProcAttr
}

func linuxProcessScopeAttributesCompatible(a *syscall.SysProcAttr) bool {
	return a == nil || (!a.Setsid && !a.Setctty && !a.Foreground && !a.Ptrace &&
		a.Pgid == 0 && a.Cloneflags == 0 && a.Unshareflags == 0)
}

// prepareLinuxProcessScopeCommand uses Go's child-side setpgid(0, 0) before
// exec. Go reports setup failure through Start's error pipe, without running
// user code outside the requested group. This function itself never starts work.
// Reject session/terminal/tracing/namespace configurations requiring a different
// ownership proof. Preserve unrelated attributes and do not mutate their alias.
func prepareLinuxProcessScopeCommand(cmd *exec.Cmd) (*linuxProcessScopeCommand, error) {
	if cmd == nil || cmd.Process != nil || cmd.ProcessState != nil || !linuxProcessScopeAttributesCompatible(cmd.SysProcAttr) {
		return nil, errors.New("invalid Linux scope command configuration")
	}
	a := &syscall.SysProcAttr{}
	if cmd.SysProcAttr != nil {
		*a = *cmd.SysProcAttr
	}
	a.Setpgid = true
	a.Pgid = 0
	cmd.SysProcAttr = a
	return &linuxProcessScopeCommand{cmd: cmd, attrs: a}, nil
}

// newLinuxProcessScopePlatform derives identity only from the prepared command's
// successfully started, unreaped direct child. No external PID/PGID is accepted.
// Post-Start Getpgid is not the authority for pre-exec membership. The receipt
// requires exclusive caller ownership; it cannot detect an unauthorized reaper.
func newLinuxProcessScopePlatform(prepared *linuxProcessScopeCommand) (*linuxProcessScopePlatform, error) {
	if prepared == nil {
		return nil, errors.New("missing Linux scope preparation")
	}
	prepared.mu.Lock()
	defer prepared.mu.Unlock()
	c := prepared.cmd
	if c == nil || c.SysProcAttr != prepared.attrs || !linuxProcessScopeAttributesCompatible(c.SysProcAttr) ||
		c.SysProcAttr == nil || !c.SysProcAttr.Setpgid || c.Process == nil || c.Process.Pid <= 0 || c.ProcessState != nil {
		return nil, errors.New("Linux scope requires its prepared started leader")
	}
	s := &linuxProcessScopePlatform{pgid: c.Process.Pid, kill: unix.Kill, waitid: unix.Waitid}
	prepared.cmd, prepared.attrs = nil, nil
	return s, nil
}

// linuxProcessScopePlatform owns only the live control identity. waitMu prevents
// retirement while observation is pending; mu permits cancellation control during
// that blocking observation. No method starts a goroutine or reaps a process.
// B4 must retain sole reap ownership: observe -> control -> retire -> cmd.Wait.
// Retirement before reap closes the stale-identifier window; the future runner
// still owns child reap, output completion and lease release after this step.
type linuxProcessScopePlatform struct {
	waitMu   sync.Mutex
	mu       sync.Mutex
	pgid     int
	observed bool
	kill     func(int, syscall.Signal) error
	waitid   func(int, int, *unix.Siginfo, int, *unix.Rusage) error
}

var _ processScopePlatform = (*linuxProcessScopePlatform)(nil)

func (s *linuxProcessScopePlatform) terminate() error {
	if s == nil {
		return os.ErrProcessDone
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pgid <= 0 || s.kill == nil {
		return os.ErrProcessDone
	}
	err := s.kill(-s.pgid, unix.SIGKILL)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	// nil acknowledges the kernel control request only, not quiescence/reaping.
	return err
}

func (s *linuxProcessScopePlatform) waitLeaderExitNoReap() error {
	if s == nil {
		return os.ErrProcessDone
	}
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	s.mu.Lock()
	if s.pgid <= 0 || s.waitid == nil {
		s.mu.Unlock()
		return os.ErrProcessDone
	}
	if s.observed {
		s.mu.Unlock()
		return nil
	}
	pid, wait := s.pgid, s.waitid
	s.mu.Unlock()
	var info unix.Siginfo
	var err error
	for {
		err = wait(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.observed = true
	}
	if errors.Is(err, unix.ECHILD) {
		// No waitable-child ownership: never leave a usable numeric identity.
		s.pgid, s.kill, s.waitid = 0, nil, nil
	}
	return err
}

// finalize retires control only. Linux has no owned group handle to close.
// It performs no implicit kill, no reap, and no all-descendants-exited assertion.
// A concurrent observation must return first; control can still unblock it.
func (s *linuxProcessScopePlatform) finalize() error {
	if s == nil {
		return nil
	}
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pgid, s.kill, s.waitid = 0, nil, nil
	return nil
}
