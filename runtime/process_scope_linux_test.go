//go:build linux

package runtime

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestLinuxScopePreparation(t *testing.T) {
	if _, err := prepareLinuxProcessScopeCommand(nil); err == nil {
		t.Fatal("nil accepted")
	}
	for _, a := range []*syscall.SysProcAttr{
		{Pgid: 42}, {Setsid: true}, {Foreground: true}, {Setctty: true}, {Ptrace: true},
		{Cloneflags: unix.CLONE_NEWPID}, {Unshareflags: unix.CLONE_NEWPID},
	} {
		c := &exec.Cmd{SysProcAttr: a}
		before := *a
		if _, err := prepareLinuxProcessScopeCommand(c); err == nil {
			t.Fatal("conflict accepted")
		}
		if c.SysProcAttr != a || a.Setpgid != before.Setpgid {
			t.Fatal("conflict mutated")
		}
	}
	for _, existing := range []*syscall.SysProcAttr{nil, {}, {Setpgid: true}, {Noctty: true, Pdeathsig: syscall.SIGKILL}} {
		c := &exec.Cmd{SysProcAttr: existing}
		p, err := prepareLinuxProcessScopeCommand(c)
		if err != nil {
			t.Fatal(err)
		}
		if !c.SysProcAttr.Setpgid || c.SysProcAttr.Pgid != 0 || c.Process != nil {
			t.Fatal("wrong pre-exec configuration")
		}
		if existing != nil && (c.SysProcAttr == existing || c.SysProcAttr.Noctty != existing.Noctty || c.SysProcAttr.Pdeathsig != existing.Pdeathsig) {
			t.Fatal("unrelated attributes or alias changed")
		}
		if _, err := newLinuxProcessScopePlatform(p); err == nil {
			t.Fatal("unstarted accepted")
		}
	}
	if _, err := prepareLinuxProcessScopeCommand(&exec.Cmd{Process: &os.Process{Pid: 123}}); err == nil {
		t.Fatal("started command accepted")
	}
}

func TestLinuxScopeIdentityFromPreparation(t *testing.T) {
	if _, err := newLinuxProcessScopePlatform(nil); err == nil {
		t.Fatal("missing receipt accepted")
	}
	for _, pid := range []int{-1, 0, 123} {
		c := &exec.Cmd{}
		p, err := prepareLinuxProcessScopeCommand(c)
		if err != nil {
			t.Fatal(err)
		}
		c.Process = &os.Process{Pid: pid} // Synthetic identity; no OS method is invoked.
		s, err := newLinuxProcessScopePlatform(p)
		if pid <= 0 {
			if err == nil {
				t.Fatal("invalid PID accepted")
			}
			continue
		}
		if err != nil || s.pgid != pid {
			t.Fatalf("leader derivation: %v", err)
		}
		if p.cmd != nil || p.attrs != nil {
			t.Fatal("receipt retained")
		}
		if _, err := newLinuxProcessScopePlatform(p); err == nil {
			t.Fatal("receipt reused")
		}
		if err := s.finalize(); err != nil {
			t.Fatal(err)
		}
	}
	for _, mutate := range []func(*exec.Cmd){
		func(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} },
		func(c *exec.Cmd) { c.SysProcAttr.Setpgid = false },
		func(c *exec.Cmd) { c.SysProcAttr.Pgid = 456 },
		func(c *exec.Cmd) { c.ProcessState = &os.ProcessState{} },
	} {
		c := &exec.Cmd{}
		p, _ := prepareLinuxProcessScopeCommand(c)
		c.Process = &os.Process{Pid: 123}
		mutate(c)
		if _, err := newLinuxProcessScopePlatform(p); err == nil {
			t.Fatal("mutated/reaped command accepted")
		}
	}
}

func TestLinuxScopeControlAndRetirement(t *testing.T) {
	for _, failure := range []error{nil, unix.ESRCH, unix.EPERM} {
		calls := 0
		s := &linuxProcessScopePlatform{pgid: 123, kill: func(pid int, signal syscall.Signal) error {
			calls++
			if pid != -123 || signal != unix.SIGKILL {
				t.Fatal("wrong group/signal")
			}
			return failure
		}}
		owner, err := newProcessScopeOwner(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := owner.activate(); err != nil {
			t.Fatal(err)
		}
		err = owner.request(processScopeNaturalExitCleanup)
		want := failure
		if failure == unix.ESRCH {
			want = os.ErrProcessDone
		}
		if !errors.Is(err, want) {
			t.Fatalf("control error %v, want %v", err, want)
		}
		if (owner.status().winner != processScopeNoControl) != (failure == nil) {
			t.Fatal("false success winner")
		}
		if err := owner.finalize(); err != nil {
			t.Fatal(err)
		}
		if err := s.finalize(); err != nil {
			t.Fatal(err)
		}
		if !errors.Is(s.terminate(), os.ErrProcessDone) {
			t.Fatal("retired group controlled")
		}
		if !errors.Is(s.waitLeaderExitNoReap(), os.ErrProcessDone) {
			t.Fatal("retired group observed")
		}
		if calls != 1 || s.pgid != 0 || s.kill != nil || s.waitid != nil {
			t.Fatal("retirement/call count failed")
		}
	}
	var missing *linuxProcessScopePlatform
	if !errors.Is(missing.terminate(), os.ErrProcessDone) || !errors.Is(missing.waitLeaderExitNoReap(), os.ErrProcessDone) || missing.finalize() != nil {
		t.Fatal("nil behavior")
	}
}

func TestLinuxScopeWaitidRetryAndFailure(t *testing.T) {
	for _, failure := range []error{nil, unix.EINVAL, unix.ECHILD} {
		calls := 0
		s := &linuxProcessScopePlatform{pgid: 123, waitid: func(kind, pid int, info *unix.Siginfo, flags int, usage *unix.Rusage) error {
			calls++
			if kind != unix.P_PID || pid != 123 || info == nil || flags != unix.WEXITED|unix.WNOWAIT || usage != nil {
				t.Fatal("wait contract changed")
			}
			if calls == 1 {
				return unix.EINTR
			}
			return failure
		}}
		if err := s.waitLeaderExitNoReap(); !errors.Is(err, failure) {
			t.Fatalf("wait failure lost: %v", err)
		}
		if calls != 2 || s.observed != (failure == nil) {
			t.Fatal("EINTR/observation mismatch")
		}
		if failure == nil {
			if err := s.waitLeaderExitNoReap(); err != nil || calls != 2 {
				t.Fatal("observation repeated")
			}
		}
		if failure == unix.ECHILD && s.pgid != 0 {
			t.Fatal("lost child identity still usable")
		}
		if err := s.finalize(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLinuxScopeBlockingObservationAllowsControl(t *testing.T) {
	entered, killed := make(chan struct{}), make(chan struct{})
	s := &linuxProcessScopePlatform{pgid: 123}
	s.waitid = func(int, int, *unix.Siginfo, int, *unix.Rusage) error { close(entered); <-killed; return nil }
	s.kill = func(pid int, sig syscall.Signal) error {
		if pid != -123 || sig != unix.SIGKILL {
			t.Error("wrong control")
		}
		close(killed)
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- s.waitLeaderExitNoReap() }()
	<-entered
	if s.waitMu.TryLock() {
		s.waitMu.Unlock()
		close(killed)
		t.Fatal("observation did not own retirement lock")
	}
	if err := s.terminate(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := s.finalize(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(s.terminate(), os.ErrProcessDone) {
		t.Fatal("control after retirement")
	}
}
