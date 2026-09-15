//go:build linux

package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func linuxScopeFixtureCommand(mode string) *exec.Cmd {
	c := exec.Command(os.Args[0], "-test.run=^TestLinuxScopeFixture$", "-test.timeout=90s")
	// No inherited credentials or external tools. Race helper exit delay is not
	// synchronization; all assertions use pipe messages, EOF and wait events.
	c.Env = []string{"FORGE_SCOPE_FIXTURE=" + mode, "GORACE=atexit_sleep_ms=0"}
	return c
}

func TestLinuxScopeFixture(t *testing.T) {
	mode := os.Getenv("FORGE_SCOPE_FIXTURE")
	if mode == "" {
		return
	}
	os.Exit(runLinuxScopeFixture(mode))
}

func runLinuxScopeFixture(mode string) int {
	// ExtraFiles arrives as blocking descriptors. Make the control pipe pollable
	// before NewFile so its failsafe deadline is supported by Go's netpoller.
	if err := unix.SetNonblock(3, true); err != nil {
		return 79
	}
	control, status := os.NewFile(3, "scope-control"), os.NewFile(4, "scope-status")
	defer control.Close()
	defer status.Close()
	// A failsafe only: EOF/explicit commands drive normal fixture lifetime.
	if err := control.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return 80
	}
	if mode == "escape" {
		// This child inherited its parent's group but is not its group leader.
		// Deliberately leave it, demonstrating the explicitly excluded behavior.
		if _, err := unix.Setsid(); err != nil {
			return 81
		}
	}
	pgid, err := unix.Getpgid(0)
	if err != nil {
		return 82
	}
	if _, err = fmt.Fprintf(status, "%d %d\n", os.Getpid(), pgid); err != nil {
		return 83
	}
	if mode == "leader" || mode == "escape-leader" {
		childControl, childStatus, life := os.NewFile(5, "child-control"), os.NewFile(6, "child-status"), os.NewFile(7, "child-life")
		childMode := "leaf"
		if mode == "escape-leader" {
			childMode = "escape"
		}
		child := linuxScopeFixtureCommand(childMode)
		child.ExtraFiles = []*os.File{childControl, childStatus, life}
		err := child.Start() // Ordinary inheritance; no separate group for descendant.
		childControl.Close()
		childStatus.Close()
		life.Close()
		if err != nil {
			return 84
		}
		if _, err = fmt.Fprintln(status, "spawned"); err != nil {
			return 85
		}
		// Intentionally do not reap/wait the descendant: the parent test orders
		// leader exit before group cleanup and observes the descendant's pipe EOF.
	} else if mode == "leaf" || mode == "escape" {
		life := os.NewFile(5, "inherited-life")
		defer life.Close()
	} else if mode != "direct" {
		return 86
	}
	scan := bufio.NewScanner(control)
	for scan.Scan() {
		switch scan.Text() {
		case "ping":
			if _, err := fmt.Fprintln(status, "pong"); err != nil {
				return 87
			}
		case "exit":
			return 0
		default:
			return 88
		}
	}
	if scan.Err() != nil {
		return 89
	}
	return 0
}

func linuxScopeTestPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	if err := r.SetReadDeadline(time.Now().Add(45 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := w.SetWriteDeadline(time.Now().Add(45 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return r, w
}

type linuxScopeNativeProcess struct {
	cmd     *exec.Cmd
	scope   *linuxProcessScopePlatform
	owner   *processScopeOwner
	control *os.File
	status  *bufio.Reader
	reaped  bool
}

func startLinuxScopeNativeProcess(t *testing.T, mode string, extra []*os.File) *linuxScopeNativeProcess {
	t.Helper()
	controlRead, controlWrite := linuxScopeTestPipe(t)
	statusRead, statusWrite := linuxScopeTestPipe(t)
	c := linuxScopeFixtureCommand(mode)
	c.ExtraFiles = append([]*os.File{controlRead, statusWrite}, extra...)
	p, err := prepareLinuxProcessScopeCommand(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Start(); err != nil {
		t.Fatal(err)
	}
	n := &linuxScopeNativeProcess{cmd: c, control: controlWrite, status: bufio.NewReader(statusRead)}
	// Register immediately after Start, including the unlikely binding failure.
	// Never signal a group after reaping; escaped descendants use independent
	// control-pipe EOF/exit cleanup instead of stale numeric identity signaling.
	t.Cleanup(func() {
		controlWrite.Close()
		if !n.reaped {
			if n.scope != nil {
				if err := n.scope.terminate(); err != nil && !errors.Is(err, os.ErrProcessDone) {
					t.Errorf("cleanup control: %v", err)
				}
				if err := n.scope.waitLeaderExitNoReap(); err != nil {
					t.Errorf("cleanup observation: %v", err)
				}
				if err := n.scope.finalize(); err != nil {
					t.Errorf("cleanup retirement: %v", err)
				}
			} else {
				if err := unix.Kill(-c.Process.Pid, unix.SIGKILL); err != nil && !errors.Is(err, unix.ESRCH) {
					t.Errorf("cleanup prepared group: %v", err)
				}
			}
			if err := c.Wait(); err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Errorf("cleanup reap: %v", err)
				}
			}
			n.reaped = true
		}
	})
	for _, f := range c.ExtraFiles {
		f.Close()
	}
	n.scope, err = newLinuxProcessScopePlatform(p)
	if err != nil {
		t.Fatal(err)
	}
	n.owner, err = newProcessScopeOwner(n.scope)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.owner.activate(); err != nil {
		t.Fatal(err)
	}
	return n
}

func linuxScopeReadLine(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	s, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("fixture handshake: %v", err)
	}
	return strings.TrimSuffix(s, "\n")
}

func linuxScopeReady(t *testing.T, r *bufio.Reader) (int, int) {
	t.Helper()
	parts := strings.Fields(linuxScopeReadLine(t, r))
	if len(parts) != 2 {
		t.Fatal("invalid ready message")
	}
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	pgid, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if pid <= 0 || pgid <= 0 {
		t.Fatal("invalid fixture identity")
	}
	return pid, pgid
}

func linuxScopeSend(t *testing.T, w *os.File, command string) {
	t.Helper()
	if _, err := fmt.Fprintln(w, command); err != nil {
		t.Fatal(err)
	}
}

func (n *linuxScopeNativeProcess) retireAndReap(t *testing.T) {
	t.Helper()
	if err := n.owner.finalize(); err != nil {
		t.Fatal(err)
	}
	if n.scope.pgid != 0 {
		t.Fatal("identity not retired before reap")
	}
	err := n.cmd.Wait()
	n.reaped = true // No subsequent signal, including on assertion failure.
	if err != nil {
		t.Fatalf("direct leader result changed: %v", err)
	}
	if n.cmd.ProcessState.ExitCode() != 0 {
		t.Fatal("natural direct-child result lost")
	}
	if err := n.cmd.Wait(); err == nil {
		t.Fatal("second reap succeeded")
	}
	if err := n.scope.terminate(); err != os.ErrProcessDone {
		t.Fatalf("post-reap control: %v", err)
	}
}

func TestLinuxScopeNativeMembershipAndNonReap(t *testing.T) {
	n := startLinuxScopeNativeProcess(t, "direct", nil)
	pid, pgid := linuxScopeReady(t, n.status)
	actual, err := unix.Getpgid(n.cmd.Process.Pid)
	if err != nil || pid != n.cmd.Process.Pid || pgid != pid || actual != pid || n.scope.pgid != pid {
		t.Fatalf("group leader mismatch: %v", err)
	}
	linuxScopeSend(t, n.control, "exit")
	if err := n.scope.waitLeaderExitNoReap(); err != nil {
		t.Fatal(err)
	}
	// A second native WNOWAIT observation proves the first did not consume it.
	var info unix.Siginfo
	if err := unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil); err != nil {
		t.Fatal(err)
	}
	n.retireAndReap(t)
}

func TestLinuxScopeNativeLeaderFirstPipeAndOutsideSentinel(t *testing.T) {
	childRead, childWrite := linuxScopeTestPipe(t)
	childStatusRead, childStatusWrite := linuxScopeTestPipe(t)
	lifeRead, lifeWrite := linuxScopeTestPipe(t)
	n := startLinuxScopeNativeProcess(t, "leader", []*os.File{childRead, childStatusWrite, lifeWrite})
	t.Cleanup(func() {
		childWrite.Close()
		if _, err := io.Copy(io.Discard, lifeRead); err != nil {
			t.Errorf("descendant cleanup EOF: %v", err)
		}
	})
	pid, pgid := linuxScopeReady(t, n.status)
	if pid != pgid || pid != n.cmd.Process.Pid {
		t.Fatal("leader membership")
	}
	if linuxScopeReadLine(t, n.status) != "spawned" {
		t.Fatal("missing descendant admission")
	}
	childPID, childPGID := linuxScopeReady(t, bufio.NewReader(childStatusRead))
	if childPID == pid || childPGID != pgid {
		t.Fatal("descendant did not inherit group")
	}
	sentinel := startLinuxScopeNativeProcess(t, "direct", nil)
	sentinelPID, sentinelPGID := linuxScopeReady(t, sentinel.status)
	if sentinelPID != sentinelPGID || sentinelPGID == pgid {
		t.Fatal("sentinel not independently scoped")
	}
	linuxScopeSend(t, n.control, "exit")
	if err := n.scope.waitLeaderExitNoReap(); err != nil {
		t.Fatal(err)
	}
	if err := n.owner.request(processScopeNaturalExitCleanup); err != nil {
		t.Fatal(err)
	}
	if n.owner.status().winner != processScopeNaturalExitCleanup {
		t.Fatal("natural classification changed")
	}
	n.retireAndReap(t)
	// Both parent and leader closed their copies. EOF proves this fixture's
	// descendant no longer retains the inherited write handle, not universal reap.
	if b, err := io.ReadAll(lifeRead); err != nil || len(b) != 0 {
		t.Fatalf("descendant inherited pipe remained open: %v", err)
	}
	linuxScopeSend(t, sentinel.control, "ping")
	if linuxScopeReadLine(t, sentinel.status) != "pong" {
		t.Fatal("outside scope was affected")
	}
	linuxScopeSend(t, sentinel.control, "exit")
	if err := sentinel.scope.waitLeaderExitNoReap(); err != nil {
		t.Fatal(err)
	}
	sentinel.retireAndReap(t)
	childWrite.Close()
}

func TestLinuxScopeNativeEscapeLimitation(t *testing.T) {
	childRead, childWrite := linuxScopeTestPipe(t)
	childStatusRead, childStatusWrite := linuxScopeTestPipe(t)
	lifeRead, lifeWrite := linuxScopeTestPipe(t)
	n := startLinuxScopeNativeProcess(t, "escape-leader", []*os.File{childRead, childStatusWrite, lifeWrite})
	t.Cleanup(func() {
		childWrite.Close()
		if _, err := io.Copy(io.Discard, lifeRead); err != nil {
			t.Errorf("escape cleanup EOF: %v", err)
		}
	})
	pid, pgid := linuxScopeReady(t, n.status)
	if pid != pgid || linuxScopeReadLine(t, n.status) != "spawned" {
		t.Fatal("leader not ready")
	}
	childStatus := bufio.NewReader(childStatusRead)
	escapedPID, escapedPGID := linuxScopeReady(t, childStatus)
	if escapedPID != escapedPGID || escapedPGID == pgid {
		t.Fatal("fixture did not leave session/group")
	}
	linuxScopeSend(t, n.control, "exit")
	if err := n.scope.waitLeaderExitNoReap(); err != nil {
		t.Fatal(err)
	}
	if err := n.owner.request(processScopeNaturalExitCleanup); err != nil {
		t.Fatal(err)
	}
	n.retireAndReap(t)
	linuxScopeSend(t, childWrite, "ping")
	if linuxScopeReadLine(t, childStatus) != "pong" {
		t.Fatal("escaped fixture unexpectedly controlled")
	}
	// Explicit independent fixture cleanup; never send a signal to a remembered
	// escaped PID. EOF on failure also ends the trusted fixture's read loop.
	linuxScopeSend(t, childWrite, "exit")
	childWrite.Close()
	if b, err := io.ReadAll(lifeRead); err != nil || len(b) != 0 {
		t.Fatalf("escape fixture not cleaned: %v", err)
	}
}
