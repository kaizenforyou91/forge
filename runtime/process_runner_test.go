package runtime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
)

const processRunnerFixtureImportBase = "github.com/kaizenforyou91/forge/runtime/testdata/"

func TestProcessRunnerStartsAndWaits(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_success")
	t.Setenv("FORGE_RUNNER_SECRET_TEST", "must-not-leak")

	t.Run("direct success and bounded output", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		directory := materialized.directory
		path := materialized.path
		workDirectory := filepath.Join(directory, runtimeProcessWorkDirectoryName)

		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if err != nil {
			t.Fatal(err)
		}
		if process.PID() <= 0 {
			t.Fatalf("PID = %d, want > 0", process.PID())
		}
		if process.Entrypoint() != loaded.Entrypoint() {
			t.Fatalf("entrypoint = %#v, want %#v", process.Entrypoint(), loaded.Entrypoint())
		}
		if process.SignerKeyID() != loaded.SignerKeyID() {
			t.Fatalf("signer = %q, want %q", process.SignerKeyID(), loaded.SignerKeyID())
		}

		const waiters = 16
		concurrentResults := make(chan ProcessResult, waiters)
		concurrentErrors := make(chan error, waiters)
		var waitGroup sync.WaitGroup
		waitGroup.Add(waiters)
		for range waiters {
			go func() {
				defer waitGroup.Done()
				concurrent, waitErr := process.Wait()
				concurrentResults <- concurrent
				concurrentErrors <- waitErr
			}()
		}

		result, err := process.Wait()
		if err != nil {
			t.Fatal(err)
		}
		waitGroup.Wait()
		close(concurrentResults)
		close(concurrentErrors)
		for waitErr := range concurrentErrors {
			if waitErr != nil {
				t.Fatalf("concurrent Wait failed: %v", waitErr)
			}
		}
		for concurrent := range concurrentResults {
			if concurrent.ExitCode != 0 || !bytes.Equal(concurrent.Stdout, result.Stdout) {
				t.Fatal("concurrent Wait returned inconsistent cached result")
			}
		}
		if result.ExitCode != 0 || result.Canceled || result.Terminated {
			t.Fatalf("result = %#v, want successful non-canceled exit", result)
		}
		if len(result.Stdout) >= runtimeProcessOutputLimit || result.StdoutTruncated {
			t.Fatalf("stdout length/truncated = %d/%v", len(result.Stdout), result.StdoutTruncated)
		}
		if len(result.Stderr) >= runtimeProcessOutputLimit || result.StderrTruncated {
			t.Fatalf("stderr length/truncated = %d/%v", len(result.Stderr), result.StderrTruncated)
		}
		if outputValue(result.Stdout, "fixture") != "process-success" {
			t.Fatalf("unexpected stdout prefix: %q", result.Stdout[:min(128, len(result.Stdout))])
		}
		if outputValue(result.Stderr, "fixture") != "process-success-stderr" {
			t.Fatalf("unexpected stderr prefix: %q", result.Stderr[:min(128, len(result.Stderr))])
		}
		if outputValue(result.Stdout, "cwd") != workDirectory {
			t.Fatalf("child cwd = %q, want %q", outputValue(result.Stdout, "cwd"), workDirectory)
		}
		if outputValue(result.Stdout, "args") != "1" {
			t.Fatalf("child args count = %q, want 1", outputValue(result.Stdout, "args"))
		}
		if outputValue(result.Stdout, "stdin-bytes") != "0" {
			t.Fatalf("child stdin bytes = %q, want 0", outputValue(result.Stdout, "stdin-bytes"))
		}
		if outputValue(result.Stdout, "secret") != "" || outputValue(result.Stdout, "path") != "" {
			t.Fatal("parent secret or PATH leaked into the child environment")
		}
		assertControlledRunnerEnvironment(t, result.Stdout, workDirectory)

		if process.cmd.ProcessState == nil || !process.cmd.ProcessState.Exited() {
			t.Fatal("direct child was not reaped")
		}
		if materialized.leaseActive || materialized.cleanupDone {
			t.Fatalf(
				"unexpected post-Wait lifecycle: active=%v cleaned=%v",
				materialized.leaseActive,
				materialized.cleanupDone,
			)
		}
		for _, existingPath := range []string{directory, path, workDirectory} {
			if _, err := os.Lstat(existingPath); err != nil {
				t.Fatalf("release without Close removed %q: %v", existingPath, err)
			}
		}

		firstByte := result.Stdout[0]
		result.Stdout[0] ^= 0xff
		repeated, err := process.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if repeated.Stdout[0] != firstByte {
			t.Fatal("Wait returned aliased stdout bytes")
		}

		second, err := NewProcessRunner().Start(context.Background(), materialized)
		if !errors.Is(err, ErrMaterializedExecutableAlreadyUsed) || second != nil {
			t.Fatalf("second Start = (%v, %v), want AlreadyUsed", second, err)
		}
		if err := materialized.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(directory); !os.IsNotExist(err) {
			t.Fatalf("explicit Close did not remove directory: %v", err)
		}
	})

	t.Run("canceled context before claim", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		process, err := NewProcessRunner().Start(ctx, materialized)
		if !errors.Is(err, context.Canceled) || process != nil {
			t.Fatalf("canceled Start = (%v, %v), want context.Canceled", process, err)
		}
		if materialized.executionClaimed || materialized.leaseActive {
			t.Fatal("pre-canceled context consumed the execution claim")
		}

		process, err = NewProcessRunner().Start(context.Background(), materialized)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := process.Wait(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("invalid inputs do not claim", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		var nilRunner *ProcessRunner
		if _, err := nilRunner.Start(context.Background(), materialized); !errors.Is(err, ErrProcessStartFailed) {
			t.Fatalf("nil runner error = %v", err)
		}
		if _, err := NewProcessRunner().Start(nil, materialized); !errors.Is(err, ErrProcessStartFailed) {
			t.Fatalf("nil context error = %v", err)
		}
		if materialized.executionClaimed {
			t.Fatal("invalid runner input consumed the execution claim")
		}
		if _, err := NewProcessRunner().Start(context.Background(), nil); !errors.Is(err, ErrMaterializedExecutableInvalid) {
			t.Fatalf("nil executable error = %v", err)
		}
	})

	var nilProcess *RunningProcess
	if nilProcess.PID() != 0 || nilProcess.Entrypoint() != (compiler.RuntimeEntrypoint{}) || nilProcess.SignerKeyID() != "" {
		t.Fatal("nil RunningProcess metadata accessors are not zero-safe")
	}
	if _, err := nilProcess.Wait(); !errors.Is(err, ErrProcessWaitFailed) {
		t.Fatalf("nil Wait error = %v", err)
	}
	if err := nilProcess.Terminate(); !errors.Is(err, ErrProcessTerminationFailed) {
		t.Fatalf("nil Terminate error = %v", err)
	}
}

func TestProcessRunnerBoundsOutputAndStillReaps(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_success/output")
	materialized := materializeRunnerFixture(t, loaded)

	process, err := NewProcessRunner().Start(context.Background(), materialized)
	if err != nil {
		t.Fatal(err)
	}
	result, err := process.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.Canceled || result.Terminated {
		t.Fatalf("result = %#v, want successful non-canceled exit", result)
	}
	if len(result.Stdout) != runtimeProcessOutputLimit || !result.StdoutTruncated {
		t.Fatalf("stdout length/truncated = %d/%v", len(result.Stdout), result.StdoutTruncated)
	}
	if len(result.Stderr) != runtimeProcessOutputLimit || !result.StderrTruncated {
		t.Fatalf("stderr length/truncated = %d/%v", len(result.Stderr), result.StderrTruncated)
	}
	if outputValue(result.Stdout, "fixture") != "process-output" ||
		outputValue(result.Stderr, "fixture") != "process-output-stderr" {
		t.Fatal("bounded output did not retain deterministic prefixes")
	}
	if process.cmd.ProcessState == nil || !process.cmd.ProcessState.Exited() {
		t.Fatal("high-output child was not reaped")
	}
}

func TestProcessRunnerReturnsNonZeroApplicationExit(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_failure")
	materialized := materializeRunnerFixture(t, loaded)

	process, err := NewProcessRunner().Start(context.Background(), materialized)
	if err != nil {
		t.Fatal(err)
	}
	result, err := process.Wait()
	if err != nil {
		t.Fatalf("non-zero application exit became infrastructure error: %v", err)
	}
	if result.ExitCode != 23 || result.Canceled || result.Terminated {
		t.Fatalf("result = %#v, want exit code 23", result)
	}
	if !bytes.Contains(result.Stdout, []byte("fixture=process-failure")) ||
		!bytes.Contains(result.Stderr, []byte("fixture=process-failure-stderr")) {
		t.Fatalf("unexpected process output: stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
	repeated, err := process.Wait()
	if err != nil || repeated.ExitCode != result.ExitCode || repeated.Canceled || repeated.Terminated {
		t.Fatalf("repeated non-zero Wait = (%#v, %v)", repeated, err)
	}
}

func TestProcessRunnerCancellationCoordinatesPendingCleanup(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_wait")
	materialized := materializeRunnerFixture(t, loaded)
	directory := materialized.directory
	path := materialized.path
	ctx, cancel := context.WithCancel(context.Background())

	process, err := NewProcessRunner().Start(ctx, materialized)
	if err != nil {
		t.Fatal(err)
	}
	if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
		t.Fatalf("Close while running = %v, want Busy", err)
	}
	for _, existingPath := range []string{directory, path} {
		if _, err := os.Lstat(existingPath); err != nil {
			t.Fatalf("Close removed active process path %q: %v", existingPath, err)
		}
	}

	cancel()
	result, err := process.Wait()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Wait error = %v", err)
	}
	if !result.Canceled {
		t.Fatal("canceled process result did not record cancellation")
	}
	if result.Terminated {
		t.Fatal("context-canceled process was labeled manually terminated")
	}
	repeated, repeatedErr := process.Wait()
	if !errors.Is(repeatedErr, context.Canceled) ||
		repeated.Canceled != result.Canceled || repeated.Terminated != result.Terminated {
		t.Fatalf("repeated canceled Wait = (%#v, %v)", repeated, repeatedErr)
	}
	if process.cmd.ProcessState == nil || !process.cmd.ProcessState.Exited() {
		t.Fatal("canceled direct child was not reaped")
	}
	if materialized.leaseActive || !materialized.cleanupDone {
		t.Fatalf(
			"pending cleanup state: active=%v cleaned=%v",
			materialized.leaseActive,
			materialized.cleanupDone,
		)
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("pending cleanup did not remove directory: %v", err)
	}
	if err := materialized.Close(); err != nil {
		t.Fatalf("final Close failed: %v", err)
	}
}

func TestRunningProcessTerminationLifecycle(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_wait")

	t.Run("concurrent termination and wait", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		directory := materialized.directory
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if err != nil {
			t.Fatal(err)
		}

		var killCalls atomic.Int32
		process.termination.mu.Lock()
		originalKill := process.termination.kill
		process.termination.kill = func() error {
			killCalls.Add(1)
			return originalKill()
		}
		process.termination.mu.Unlock()

		waitResult := make(chan ProcessResult, 1)
		waitError := make(chan error, 1)
		go func() {
			result, waitErr := process.Wait()
			waitResult <- result
			waitError <- waitErr
		}()

		const terminators = 16
		terminationErrors := make(chan error, terminators)
		var waitGroup sync.WaitGroup
		waitGroup.Add(terminators)
		for range terminators {
			go func() {
				defer waitGroup.Done()
				terminationErrors <- process.Terminate()
			}()
		}
		waitGroup.Wait()
		close(terminationErrors)
		for terminateErr := range terminationErrors {
			if terminateErr != nil {
				t.Fatalf("concurrent Terminate failed: %v", terminateErr)
			}
		}

		result := <-waitResult
		if err := <-waitError; err != nil {
			t.Fatalf("manual termination Wait failed: %v", err)
		}
		if !result.Terminated || result.Canceled {
			t.Fatalf("manual termination result = %#v", result)
		}
		if killCalls.Load() != 1 {
			t.Fatalf("direct-child kill calls = %d, want 1", killCalls.Load())
		}
		if process.cmd.ProcessState == nil || !process.cmd.ProcessState.Exited() {
			t.Fatal("manually terminated child was not reaped")
		}
		if materialized.leaseActive || materialized.cleanupDone {
			t.Fatalf(
				"post-termination lifecycle: active=%v cleaned=%v",
				materialized.leaseActive,
				materialized.cleanupDone,
			)
		}
		if _, err := os.Lstat(directory); err != nil {
			t.Fatalf("manual termination triggered automatic cleanup: %v", err)
		}

		repeated, err := process.Wait()
		if err != nil || repeated.Terminated != result.Terminated || repeated.ExitCode != result.ExitCode {
			t.Fatalf("repeated terminated Wait = (%#v, %v)", repeated, err)
		}
		if err := process.Terminate(); err != nil {
			t.Fatalf("Terminate after completed manual termination failed: %v", err)
		}
	})

	t.Run("close then terminate", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		directory := materialized.directory
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if err != nil {
			t.Fatal(err)
		}
		if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
			t.Fatalf("Close while running = %v, want Busy", err)
		}
		if err := process.Terminate(); err != nil {
			t.Fatal(err)
		}
		result, err := process.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if !result.Terminated || result.Canceled {
			t.Fatalf("result = %#v, want manual termination", result)
		}
		if _, err := os.Lstat(directory); !os.IsNotExist(err) {
			t.Fatalf("pending cleanup did not remove directory: %v", err)
		}
		if err := materialized.Close(); err != nil {
			t.Fatalf("final Close failed: %v", err)
		}
	})

	t.Run("manual termination wins cancellation", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		ctx, cancel := context.WithCancel(context.Background())
		process, err := NewProcessRunner().Start(ctx, materialized)
		if err != nil {
			t.Fatal(err)
		}
		if err := process.Terminate(); err != nil {
			t.Fatal(err)
		}
		cancel()
		result, err := process.Wait()
		if err != nil {
			t.Fatalf("manual winner returned error: %v", err)
		}
		if !result.Terminated || result.Canceled {
			t.Fatalf("manual winner result = %#v", result)
		}
	})

	t.Run("cancellation wins manual termination", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		ctx, cancel := context.WithCancel(context.Background())
		process, err := NewProcessRunner().Start(ctx, materialized)
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		waitForProcessTerminationCause(t, process, processTerminationCauseCancellation)
		if err := process.Terminate(); err != nil {
			t.Fatalf("Terminate after cancellation winner failed: %v", err)
		}
		result, err := process.Wait()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation winner error = %v", err)
		}
		if !result.Canceled || result.Terminated {
			t.Fatalf("cancellation winner result = %#v", result)
		}
	})

	t.Run("termination failure is stable", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		ctx, cancel := context.WithCancel(context.Background())
		process, err := NewProcessRunner().Start(ctx, materialized)
		if err != nil {
			t.Fatal(err)
		}

		killFailure := errors.New("injected direct-child kill failure")
		var killCalls atomic.Int32
		process.termination.mu.Lock()
		originalKill := process.termination.kill
		process.termination.kill = func() error {
			killCalls.Add(1)
			return killFailure
		}
		process.termination.mu.Unlock()

		firstErr := process.Terminate()
		if !errors.Is(firstErr, ErrProcessTerminationFailed) || !errors.Is(firstErr, killFailure) {
			t.Fatalf("termination failure = %v", firstErr)
		}
		secondErr := process.Terminate()
		if !errors.Is(secondErr, ErrProcessTerminationFailed) || !errors.Is(secondErr, killFailure) {
			t.Fatalf("repeated termination failure = %v", secondErr)
		}
		if killCalls.Load() != 1 {
			t.Fatalf("failed kill attempts = %d, want 1", killCalls.Load())
		}

		process.termination.mu.Lock()
		process.termination.kill = originalKill
		process.termination.mu.Unlock()
		cancel()
		if _, err := process.Wait(); !errors.Is(err, context.Canceled) {
			t.Fatalf("cleanup cancellation error = %v", err)
		}
	})
}

func TestRunningProcessTerminateAfterNaturalExitIsBenign(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_success")
	materialized := materializeRunnerFixture(t, loaded)
	ctx, cancel := context.WithCancel(context.Background())
	process, err := NewProcessRunner().Start(ctx, materialized)
	if err != nil {
		t.Fatal(err)
	}
	result, err := process.Wait()
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := process.Terminate(); err != nil {
		t.Fatalf("Terminate after natural exit failed: %v", err)
	}
	repeated, err := process.Wait()
	if err != nil {
		t.Fatalf("late cancellation changed cached Wait error: %v", err)
	}
	if result.ExitCode != 0 || result.Canceled || result.Terminated ||
		repeated.Canceled || repeated.Terminated {
		t.Fatalf("natural exit did not win: first=%#v repeated=%#v", result, repeated)
	}
}

func TestRunningProcessNaturalExitCancellationRaceIsCoherent(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_success")

	for iteration := range 4 {
		materialized := materializeRunnerFixture(t, loaded)
		ctx, cancel := context.WithCancel(context.Background())
		process, err := NewProcessRunner().Start(ctx, materialized)
		if err != nil {
			t.Fatal(err)
		}
		go cancel()

		result, waitErr := process.Wait()
		if result.Terminated {
			t.Fatalf("iteration %d: cancellation race labeled manual termination", iteration)
		}
		if result.Canceled {
			if !errors.Is(waitErr, context.Canceled) {
				t.Fatalf("iteration %d: canceled result error = %v", iteration, waitErr)
			}
		} else if waitErr != nil || result.ExitCode != 0 {
			t.Fatalf(
				"iteration %d: natural winner = result %#v error %v",
				iteration,
				result,
				waitErr,
			)
		}
		closeMaterializedEventually(t, materialized)
	}
}

func TestRunningProcessCleanupFailurePreservesResult(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_wait")

	t.Run("manual termination", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		cleanupFailure := errors.New("injected process cleanup failure")
		installRetryableCleanupFailure(materialized, cleanupFailure)
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if err != nil {
			t.Fatal(err)
		}
		if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
			t.Fatalf("Close while running = %v, want Busy", err)
		}
		if err := process.Terminate(); err != nil {
			t.Fatal(err)
		}

		result, err := process.Wait()
		if !errors.Is(err, ErrExecutableMaterializationFailed) || !errors.Is(err, cleanupFailure) {
			t.Fatalf("cleanup Wait error = %v", err)
		}
		if !result.Terminated || result.Canceled || result.ExitCode != process.cmd.ProcessState.ExitCode() {
			t.Fatalf("cleanup failure discarded process result: %#v", result)
		}
		repeated, repeatedErr := process.Wait()
		if !errors.Is(repeatedErr, cleanupFailure) || repeated.ExitCode != result.ExitCode ||
			repeated.Terminated != result.Terminated || !bytes.Equal(repeated.Stdout, result.Stdout) {
			t.Fatalf("cached cleanup result changed: result=%#v error=%v", repeated, repeatedErr)
		}
		if err := materialized.Close(); err != nil {
			t.Fatalf("cleanup retry failed: %v", err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		cleanupFailure := errors.New("injected canceled cleanup failure")
		installRetryableCleanupFailure(materialized, cleanupFailure)
		ctx, cancel := context.WithCancel(context.Background())
		process, err := NewProcessRunner().Start(ctx, materialized)
		if err != nil {
			t.Fatal(err)
		}
		if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
			t.Fatalf("Close while running = %v, want Busy", err)
		}
		cancel()

		result, err := process.Wait()
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrExecutableMaterializationFailed) ||
			!errors.Is(err, cleanupFailure) {
			t.Fatalf("joined cancellation cleanup error = %v", err)
		}
		if !result.Canceled || result.Terminated || result.ExitCode != process.cmd.ProcessState.ExitCode() {
			t.Fatalf("cancellation cleanup failure discarded process result: %#v", result)
		}
		if err := materialized.Close(); err != nil {
			t.Fatalf("canceled cleanup retry failed: %v", err)
		}
	})
}

func TestProcessRunnerRejectsInvalidStartEvidence(t *testing.T) {
	loaded := loadProcessRunnerFixture(t, "process_success")

	t.Run("plain text executable", func(t *testing.T) {
		materialized := materializeTestPackage(t, []byte("not-an-executable"))
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if !errors.Is(err, ErrMaterializedExecutableInvalid) || process != nil {
			t.Fatalf("plain-text Start = (%v, %v), want invalid materialization", process, err)
		}
	})

	tests := map[string]func(*testing.T, *MaterializedExecutable){
		"wrong host": func(t *testing.T, executable *MaterializedExecutable) {
			executable.targetOS = "forge-wrong-os"
		},
		"size changed": func(t *testing.T, executable *MaterializedExecutable) {
			if err := os.Truncate(executable.path, executable.expectedSize-1); err != nil {
				t.Fatal(err)
			}
		},
		"digest changed": func(t *testing.T, executable *MaterializedExecutable) {
			data, err := os.ReadFile(executable.path)
			if err != nil {
				t.Fatal(err)
			}
			data[len(data)-1] ^= 0xff
			if err := os.WriteFile(executable.path, data, 0o700); err != nil {
				t.Fatal(err)
			}
		},
		"non regular path": func(t *testing.T, executable *MaterializedExecutable) {
			if err := os.Remove(executable.path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(executable.path, 0o700); err != nil {
				t.Fatal(err)
			}
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			materialized := materializeRunnerFixture(t, loaded)
			mutate(t, materialized)
			process, err := NewProcessRunner().Start(context.Background(), materialized)
			if !errors.Is(err, ErrMaterializedExecutableInvalid) || process != nil {
				t.Fatalf("Start = (%v, %v), want invalid materialization", process, err)
			}
			if materialized.leaseActive || !materialized.executionClaimed {
				t.Fatalf(
					"failed validation lifecycle: active=%v claimed=%v",
					materialized.leaseActive,
					materialized.executionClaimed,
				)
			}
		})
	}

	t.Run("symlink path", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		target := materialized.path + ".target"
		if err := os.Rename(materialized.path, target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, materialized.path); err != nil {
			t.Skipf("symlink creation is not available: %v", err)
		}
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if !errors.Is(err, ErrMaterializedExecutableInvalid) || process != nil {
			t.Fatalf("symlink Start = (%v, %v), want invalid materialization", process, err)
		}
	})

	t.Run("existing work directory", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		if err := os.Mkdir(filepath.Join(materialized.directory, runtimeProcessWorkDirectoryName), 0o700); err != nil {
			t.Fatal(err)
		}
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if !errors.Is(err, ErrProcessStartFailed) || process != nil {
			t.Fatalf("Start = (%v, %v), want ErrProcessStartFailed", process, err)
		}
		second, err := NewProcessRunner().Start(context.Background(), materialized)
		if !errors.Is(err, ErrMaterializedExecutableAlreadyUsed) || second != nil {
			t.Fatalf("second Start = (%v, %v), want AlreadyUsed", second, err)
		}
	})

	t.Run("closed before start", func(t *testing.T) {
		materialized := materializeRunnerFixture(t, loaded)
		if err := materialized.Close(); err != nil {
			t.Fatal(err)
		}
		process, err := NewProcessRunner().Start(context.Background(), materialized)
		if !errors.Is(err, ErrMaterializedExecutableClosed) || process != nil {
			t.Fatalf("closed Start = (%v, %v), want Closed", process, err)
		}
	})
}

func TestProcessRunnerStartPathIdentityValidation(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first")
	secondPath := filepath.Join(directory, "second")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("same"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	firstInfo, err := os.Lstat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateStartExecutablePath(secondPath, firstInfo, 4); !errors.Is(err, ErrMaterializedExecutableInvalid) {
		t.Fatalf("identity mismatch error = %v", err)
	}
}

func TestProcessResultBoundedWriterRetainsPrefixAndDrains(t *testing.T) {
	writer := newBoundedOutputWriter(8)
	data := []byte("0123456789abcdef")
	written, err := writer.Write(data)
	if err != nil || written != len(data) {
		t.Fatalf("Write = (%d, %v), want (%d, nil)", written, err, len(data))
	}
	retained, truncated := writer.snapshot()
	if string(retained) != "01234567" || !truncated {
		t.Fatalf("snapshot = (%q, %v)", retained, truncated)
	}
	retained[0] = 'x'
	second, _ := writer.snapshot()
	if string(second) != "01234567" {
		t.Fatal("bounded writer snapshot aliases internal storage")
	}
}

func loadProcessRunnerFixture(t *testing.T, fixture string) VerifiedRunnablePackage {
	t.Helper()

	safeName := strings.NewReplacer("_", "-", "/", "-").Replace(fixture)
	module := safeName
	version := "v1"
	importPath := processRunnerFixtureImportBase + fixture
	signer, trustStore := newTestSignerAndTrustStore(t)
	sources := compiler.NewPackageSourceRegistry()
	if err := sources.Register(compiler.PackageSource{
		Name:       module,
		Version:    version,
		ImportPath: importPath,
	}); err != nil {
		t.Fatal(err)
	}
	builder, err := compiler.NewGoApplicationExecutableBuilder(compiler.NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := compiler.NewRunnablePackageCompiler(
		sources,
		builder,
		compiler.NewZIPPackagerWithSigner(signer),
	)
	if err != nil {
		t.Fatal(err)
	}

	packagePath := filepath.Join(t.TempDir(), safeName+".zip")
	entrypoint := compiler.RuntimeEntrypoint{Module: module, Version: version}
	if err := coordinator.Compile(context.Background(), compiler.RunnablePackageRequest{
		Plan: manifest.BuildPlan{
			ManifestName:    "process-runner-fixture",
			ManifestVersion: "v1",
			Steps: []manifest.BuildStep{{
				Module: module + "@" + version,
			}},
		},
		Entrypoint:       entrypoint,
		WorkingDirectory: repositoryRoot(t),
		OutputPath:       packagePath,
	}); err != nil {
		t.Fatal(err)
	}

	loader, err := NewVerifiedRunnablePackageLoader(trustStore)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := loader.Load(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func materializeRunnerFixture(t *testing.T, loaded VerifiedRunnablePackage) *MaterializedExecutable {
	t.Helper()
	materialized, err := NewSecureExecutableMaterializer().Materialize(loaded)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := materialized.Close(); err != nil {
			t.Errorf("cleanup runner materialization: %v", err)
		}
	})
	return materialized
}

func waitForProcessTerminationCause(
	t *testing.T,
	process *RunningProcess,
	want processTerminationCause,
) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		process.termination.mu.Lock()
		got := process.termination.winner
		completed := process.termination.completed
		process.termination.mu.Unlock()
		if got == want {
			return
		}
		if completed || time.Now().After(deadline) {
			t.Fatalf("termination cause = %d (completed=%v), want %d", got, completed, want)
		}
		time.Sleep(time.Millisecond)
	}
}

func installRetryableCleanupFailure(executable *MaterializedExecutable, failure error) {
	var calls atomic.Int32
	executable.removeAll = func(path string) error {
		if calls.Add(1) == 1 {
			return failure
		}
		return os.RemoveAll(path)
	}
}

func closeMaterializedEventually(t *testing.T, executable *MaterializedExecutable) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		err := executable.Close()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("cleanup materialized executable after retry window: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func outputValue(output []byte, key string) string {
	prefix := key + "="
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSuffix(strings.TrimPrefix(line, prefix), "\r")
		}
	}
	return ""
}

func assertControlledRunnerEnvironment(t *testing.T, output []byte, workDirectory string) {
	t.Helper()
	equalPath := func(got, want string) bool {
		if goruntime.GOOS == "windows" {
			return strings.EqualFold(filepath.Clean(got), filepath.Clean(want))
		}
		return filepath.Clean(got) == filepath.Clean(want)
	}

	if goruntime.GOOS == "windows" {
		for _, key := range []string{"userprofile", "temp", "tmp"} {
			if got := outputValue(output, key); !equalPath(got, workDirectory) {
				t.Fatalf("child %s = %q, want %q", key, got, workDirectory)
			}
		}
		if outputValue(output, "home") != "" || outputValue(output, "tmpdir") != "" {
			t.Fatal("non-Windows environment variables leaked into Windows child")
		}
		return
	}

	for _, key := range []string{"home", "tmpdir"} {
		if got := outputValue(output, key); !equalPath(got, workDirectory) {
			t.Fatalf("child %s = %q, want %q", key, got, workDirectory)
		}
	}
	if outputValue(output, "userprofile") != "" ||
		outputValue(output, "temp") != "" ||
		outputValue(output, "tmp") != "" {
		t.Fatal("Windows environment variables leaked into non-Windows child")
	}
}
