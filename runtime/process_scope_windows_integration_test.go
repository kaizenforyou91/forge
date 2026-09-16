//go:build windows

package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var windowsScopeIsProcessInJob = windowsScopeKernel.NewProc("IsProcessInJob")
var windowsScopeGetHandleInformation = windowsScopeKernel.NewProc("GetHandleInformation")

func windowsScopeMembership(process, job windows.Handle) (bool, error) {
	var member uint32
	r, _, err := windowsScopeIsProcessInJob.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&member)))
	if r == 0 {
		return false, err
	}
	return member != 0, nil
}

type windowsScopeFixture struct {
	input, output *os.File
	reader        *bufio.Reader
	launch        *windowsProcessScopeLaunch
	cmd           *exec.Cmd
}

func windowsScopeFixtureEnv(mode string) []string {
	// Explicit minimum environment; never copy credentials from the parent.
	return []string{"FORGE_WINDOWS_SCOPE_FIXTURE=" + mode, "SystemRoot=" + os.Getenv("SystemRoot"), "GORACE=atexit_sleep_ms=0"}
}

func openWindowsScopeFixture(mode string, scoped bool) (_ *windowsScopeFixture, err error) {
	stdin, control, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()
	status, stdout, err := os.Pipe()
	if err != nil {
		control.Close()
		return nil, err
	}
	defer stdout.Close()
	f := &windowsScopeFixture{input: control, output: status, reader: bufio.NewReader(status)}
	defer func() {
		if err != nil {
			control.Close()
			status.Close()
		}
	}()
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	args := []string{"-test.run=^TestWindowsScopeFixture$", "-test.timeout=90s"}
	if scoped {
		cmdline := syscall.EscapeArg(exe)
		for _, a := range args {
			cmdline += " " + syscall.EscapeArg(a)
		}
		f.launch, err = startWindowsProcessInScope(&windowsProcessScopeLaunchSpec{executable: exe, commandLine: cmdline, directory: dir, environment: windowsScopeFixtureEnv(mode), stdio: [3]windows.Handle{windows.Handle(stdin.Fd()), windows.Handle(stdout.Fd()), windows.Handle(stdout.Fd())}})
	} else {
		f.cmd = exec.Command(exe, args...)
		f.cmd.Env = windowsScopeFixtureEnv(mode)
		f.cmd.Stdin, f.cmd.Stdout, f.cmd.Stderr = stdin, stdout, stdout
		err = f.cmd.Start()
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *windowsScopeFixture) send(s string) error { _, err := fmt.Fprintln(f.input, s); return err }

func (f *windowsScopeFixture) line() (string, error) {
	type result struct {
		s string
		e error
	}
	ch := make(chan result, 1)
	go func() { s, e := f.reader.ReadString('\n'); ch <- result{strings.TrimSpace(s), e} }()
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	select {
	case r := <-ch:
		return r.s, r.e
	case <-timer.C:
		// Deadline is only a failsafe. Close owned handles and join the reader.
		_ = f.cleanup()
		<-ch
		return "", errors.New("fixture handshake deadline")
	}
}

func (f *windowsScopeFixture) cleanup() error {
	var cleanupErr error
	if f.launch != nil {
		// Failure-safety only: this is not the normal successful control winner.
		cleanupErr = errors.Join(cleanupErr, f.launch.scope.finalize())
		if f.launch.process != 0 {
			_, err := windowsScopeWait(f.launch.process)
			cleanupErr = errors.Join(cleanupErr, err, windows.CloseHandle(f.launch.process))
			f.launch.process = 0
		}
	}
	if f.cmd != nil && f.cmd.Process != nil {
		if err := f.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		if err := f.cmd.Wait(); err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				cleanupErr = errors.Join(cleanupErr, err)
			}
		}
		f.cmd = nil
	}
	_ = f.input.Close()
	_ = f.output.Close()
	return cleanupErr
}

func windowsScopeWait(handle windows.Handle) (uint32, error) {
	s, err := windows.WaitForSingleObject(handle, 20000)
	if err != nil {
		return 0, err
	}
	if s != windows.WAIT_OBJECT_0 {
		return 0, fmt.Errorf("fixture wait status %d", s)
	}
	var code uint32
	err = windows.GetExitCodeProcess(handle, &code)
	return code, err
}

func requireWindowsScopeLine(t *testing.T, f *windowsScopeFixture, want string) {
	t.Helper()
	s, err := f.line()
	if err != nil || s != want {
		t.Fatalf("fixture line %q / %v, want %q", s, err, want)
	}
}

func newWindowsScopeFixture(t *testing.T, mode string, scoped bool) *windowsScopeFixture {
	t.Helper()
	f, err := openWindowsScopeFixture(mode, scoped)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.cleanup(); err != nil {
			t.Errorf("fixture cleanup: %v", err)
		}
	})
	return f
}

func TestWindowsScopeFixture(t *testing.T) {
	mode := os.Getenv("FORGE_WINDOWS_SCOPE_FIXTURE")
	if mode == "" {
		return
	}
	os.Exit(runWindowsScopeFixture(mode))
}

func runWindowsScopeFixture(mode string) int {
	if mode == "descendant" {
		// An inherited stdout remains open until OS termination. EOF on a parent
		// control pipe cannot accidentally make this fixture exit successfully.
		e, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			return 81
		}
		defer windows.CloseHandle(e)
		fmt.Fprintln(os.Stdout, "descendant-ready")
		_, err = windows.WaitForSingleObject(e, windows.INFINITE)
		if err != nil {
			return 82
		}
		return 83
	}
	if mode == "leader" {
		exe, err := os.Executable()
		if err != nil {
			return 84
		}
		cmd := exec.Command(exe, "-test.run=^TestWindowsScopeFixture$", "-test.timeout=90s")
		cmd.Env = windowsScopeFixtureEnv("descendant")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err = cmd.Start(); err != nil {
			return 85
		}
		// Test-owned descendant stays in inherited Job. Parent fixture failure
		// cleanup is private Job closure, never process enumeration.
		defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	}
	if mode == "nested" {
		sibling, err := openWindowsScopeFixture("leaf", false)
		if err != nil {
			return 86
		}
		defer sibling.cleanup()
		if s, e := sibling.line(); e != nil || s != "ready" {
			return 87
		}
		inner, err := openWindowsScopeFixture("leaf", true)
		if err != nil {
			return 88
		}
		defer inner.cleanup()
		if s, e := inner.line(); e != nil || s != "ready" {
			return 89
		}
		member, err := windowsScopeMembership(inner.launch.process, inner.launch.scope.job)
		if err != nil || !member {
			return 90
		}
		if err = inner.launch.scope.terminate(); err != nil {
			return 91
		}
		if code, e := windowsScopeWait(inner.launch.process); e != nil || code != 1 {
			return 92
		}
		if _, e := inner.line(); !errors.Is(e, io.EOF) {
			return 93
		}
		if err = inner.launch.scope.finalize(); err != nil {
			return 94
		}
		if err = sibling.send("ping"); err != nil {
			return 95
		}
		if s, e := sibling.line(); e != nil || s != "pong" {
			return 96
		}
		// Intermediate helper and ordinary sibling share the test-owned outer
		// Job; inner termination/closure must leave both responsive.
		fmt.Fprintln(os.Stdout, "nested-ok")
	}
	fmt.Fprintln(os.Stdout, "ready")
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		switch scan.Text() {
		case "ping":
			fmt.Fprintln(os.Stdout, "pong")
		case "exit":
			return 0
		default:
			return 97
		}
	}
	return 0
}

func TestWindowsScopeNativeCreationAndHandlePolicy(t *testing.T) {
	v := windows.RtlGetVersion()
	t.Logf("native Windows %d.%d build %d; %s; x/sys v0.13.0; no elevation or host-policy mutation requested", v.MajorVersion, v.MinorVersion, v.BuildNumber, runtime.Version())
	f := newWindowsScopeFixture(t, "leaf", true)
	// JOB_LIST passed to CreateProcess is the guarantee; this is independent
	// post-create evidence, not a replacement for creation-time membership.
	member, err := windowsScopeMembership(f.launch.process, f.launch.scope.job)
	if err != nil || !member {
		t.Fatalf("membership %v %v", member, err)
	}
	var flags uint32
	r, _, e := windowsScopeGetHandleInformation.Call(uintptr(f.launch.scope.job), uintptr(unsafe.Pointer(&flags)))
	if r == 0 {
		t.Fatal(e)
	}
	if flags&windows.HANDLE_FLAG_INHERIT != 0 {
		t.Fatal("Job inheritable")
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	err = windows.QueryInformationJobObject(f.launch.scope.job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.BasicLimitInformation.LimitFlags != windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE {
		t.Fatalf("unexpected Job flags %x", info.BasicLimitInformation.LimitFlags)
	}
	requireWindowsScopeLine(t, f, "ready")
	if err = f.send("exit"); err != nil {
		t.Fatal(err)
	}
	if code, e := windowsScopeWait(f.launch.process); e != nil || code != 0 {
		t.Fatalf("normal result %d %v", code, e)
	}
}

func TestWindowsScopeNativeDescendantAndSentinel(t *testing.T) {
	sentinel := newWindowsScopeFixture(t, "leaf", false)
	requireWindowsScopeLine(t, sentinel, "ready")
	f := newWindowsScopeFixture(t, "leader", true)
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		s, e := f.line()
		if e != nil {
			t.Fatal(e)
		}
		seen[s] = true
	}
	if !seen["ready"] || !seen["descendant-ready"] {
		t.Fatal("missing descendant handshake", seen)
	}
	// OpenProcess is only for the sentinel membership assertion, not control.
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(sentinel.cmd.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	member, err := windowsScopeMembership(h, f.launch.scope.job)
	_ = windows.CloseHandle(h)
	if err != nil || member {
		t.Fatal("sentinel membership", member, err)
	}
	if err = f.launch.scope.terminate(); err != nil {
		t.Fatal(err)
	}
	if code, e := windowsScopeWait(f.launch.process); e != nil || code != 1 {
		t.Fatalf("termination result %d %v", code, e)
	}
	if _, e := f.line(); !errors.Is(e, io.EOF) {
		t.Fatal("descendant inherited pipe not closed", e)
	}
	if err = f.launch.scope.finalize(); err != nil {
		t.Fatal(err)
	}
	if err = sentinel.send("ping"); err != nil {
		t.Fatal(err)
	}
	requireWindowsScopeLine(t, sentinel, "pong")
}

func TestWindowsScopeNativeNestedJob(t *testing.T) {
	// The first launcher is the deterministic test-owned outer Job. Its helper
	// creates an inner Forge Job plus an ordinary outer sibling. Never open,
	// alter, close or break away from the runner's own host Job.
	f := newWindowsScopeFixture(t, "nested", true)
	requireWindowsScopeLine(t, f, "nested-ok")
	requireWindowsScopeLine(t, f, "ready")
	if err := f.send("ping"); err != nil {
		t.Fatal(err)
	}
	requireWindowsScopeLine(t, f, "pong")
	if err := f.send("exit"); err != nil {
		t.Fatal(err)
	}
	if code, e := windowsScopeWait(f.launch.process); e != nil || code != 0 {
		t.Fatalf("nested fixture %d %v", code, e)
	}
}

func TestWindowsScopeNativeKillOnClose(t *testing.T) {
	f := newWindowsScopeFixture(t, "leaf", true)
	requireWindowsScopeLine(t, f, "ready")
	// Isolated failure-safety evidence. B4 must perform normal terminal control
	// before active-scope finalization; CloseHandle is not a B1 control winner.
	if err := f.launch.scope.finalize(); err != nil {
		t.Fatal(err)
	}
	if _, e := windowsScopeWait(f.launch.process); e != nil {
		t.Fatal(e)
	}
	if _, e := f.line(); !errors.Is(e, io.EOF) {
		t.Fatal("kill-on-close pipe", e)
	}
}
