package runtime

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestMaterializedExecutableExecutionLeaseCloseBeforeAcquire(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	directory := materialized.directory

	if err := materialized.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("private directory still exists or cannot be inspected: %v", err)
	}

	lease, err := materialized.acquireExecutionLease()
	if !errors.Is(err, ErrMaterializedExecutableClosed) {
		t.Fatalf("expected ErrMaterializedExecutableClosed, got %v", err)
	}
	if lease != nil {
		t.Fatal("expected no lease after Close")
	}
}

func TestMaterializedExecutableExecutionLeaseAcquireBeforeClose(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	directory := materialized.directory
	path := materialized.path

	lease, err := materialized.acquireExecutionLease()
	if err != nil {
		t.Fatal(err)
	}

	err = materialized.Close()
	if !errors.Is(err, ErrMaterializedExecutableBusy) {
		t.Fatalf("expected ErrMaterializedExecutableBusy, got %v", err)
	}
	if !materialized.closed || !materialized.executionClaimed || !materialized.leaseActive {
		t.Fatalf(
			"unexpected leased lifecycle: closed=%v executionClaimed=%v leaseActive=%v",
			materialized.closed,
			materialized.executionClaimed,
			materialized.leaseActive,
		)
	}
	if materialized.cleanupDone {
		t.Fatal("Close cleaned an actively leased materialization")
	}
	for _, existingPath := range []string{directory, path} {
		if _, err := os.Lstat(existingPath); err != nil {
			t.Fatalf("leased path %q is unavailable: %v", existingPath, err)
		}
	}

	if err := lease.release(); err != nil {
		t.Fatal(err)
	}
	if materialized.leaseActive || !materialized.cleanupDone {
		t.Fatalf(
			"unexpected released lifecycle: leaseActive=%v cleanupDone=%v",
			materialized.leaseActive,
			materialized.cleanupDone,
		)
	}
	if materialized.directory != "" || materialized.path != "" {
		t.Fatal("successful pending cleanup retained filesystem paths")
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("pending cleanup did not remove private directory: %v", err)
	}
	if err := materialized.Close(); err != nil {
		t.Fatalf("final Close failed: %v", err)
	}
}

func TestMaterializedExecutableExecutionLeaseIsSingleUse(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	directory := materialized.directory
	path := materialized.path

	lease, err := materialized.acquireExecutionLease()
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.release(); err != nil {
		t.Fatal(err)
	}
	if !materialized.executionClaimed || materialized.leaseActive {
		t.Fatalf(
			"unexpected released claim: executionClaimed=%v leaseActive=%v",
			materialized.executionClaimed,
			materialized.leaseActive,
		)
	}
	if materialized.cleanupDone {
		t.Fatal("release without Close unexpectedly cleaned the materialization")
	}
	for _, existingPath := range []string{directory, path} {
		if _, err := os.Lstat(existingPath); err != nil {
			t.Fatalf("released materialization path %q is unavailable: %v", existingPath, err)
		}
	}

	second, err := materialized.acquireExecutionLease()
	if !errors.Is(err, ErrMaterializedExecutableAlreadyUsed) {
		t.Fatalf("expected ErrMaterializedExecutableAlreadyUsed, got %v", err)
	}
	if second != nil {
		t.Fatal("expected no second lease")
	}
	if err := materialized.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMaterializedExecutableExecutionLeaseRejectsBusyReacquisition(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	lease, err := materialized.acquireExecutionLease()
	if err != nil {
		t.Fatal(err)
	}

	second, err := materialized.acquireExecutionLease()
	if !errors.Is(err, ErrMaterializedExecutableBusy) {
		t.Fatalf("expected ErrMaterializedExecutableBusy, got %v", err)
	}
	if second != nil {
		t.Fatal("expected no concurrent lease")
	}

	if err := lease.release(); err != nil {
		t.Fatal(err)
	}
	if err := materialized.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMaterializedExecutableExecutionLeaseCleanupCanRetryAfterReleaseFailure(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	directory := materialized.directory
	cleanupFailure := errors.New("injected pending cleanup failure")
	var cleanupCalls atomic.Int32
	materialized.removeAll = func(path string) error {
		if cleanupCalls.Add(1) == 1 {
			return cleanupFailure
		}
		return os.RemoveAll(path)
	}

	lease, err := materialized.acquireExecutionLease()
	if err != nil {
		t.Fatal(err)
	}
	if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
		t.Fatalf("expected pending Close to report busy, got %v", err)
	}
	if cleanupCalls.Load() != 0 {
		t.Fatal("Close attempted cleanup while the lease was active")
	}

	err = lease.release()
	if !errors.Is(err, ErrExecutableMaterializationFailed) ||
		!errors.Is(err, cleanupFailure) {
		t.Fatalf("expected materialization and cleanup errors, got %v", err)
	}
	if !materialized.closed || !materialized.executionClaimed || materialized.leaseActive || materialized.cleanupDone {
		t.Fatalf(
			"unexpected failed-release lifecycle: closed=%v claimed=%v active=%v cleaned=%v",
			materialized.closed,
			materialized.executionClaimed,
			materialized.leaseActive,
			materialized.cleanupDone,
		)
	}
	if materialized.directory != directory || materialized.path == "" {
		t.Fatal("failed pending cleanup discarded retry evidence")
	}
	if _, statErr := os.Lstat(directory); statErr != nil {
		t.Fatalf("failed pending cleanup removed the directory: %v", statErr)
	}

	if repeatedErr := lease.release(); !errors.Is(repeatedErr, cleanupFailure) {
		t.Fatalf("repeated release did not preserve its result: %v", repeatedErr)
	}
	if cleanupCalls.Load() != 1 {
		t.Fatalf("release retried cleanup, calls = %d", cleanupCalls.Load())
	}
	if err := materialized.Close(); err != nil {
		t.Fatalf("Close cleanup retry failed: %v", err)
	}
	if cleanupCalls.Load() != 2 || !materialized.cleanupDone {
		t.Fatalf("cleanup retry state: calls=%d cleanupDone=%v", cleanupCalls.Load(), materialized.cleanupDone)
	}
}

func TestMaterializedExecutableExecutionLeaseReleaseIsConcurrencySafe(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	var cleanupCalls atomic.Int32
	materialized.removeAll = func(path string) error {
		cleanupCalls.Add(1)
		return os.RemoveAll(path)
	}

	lease, err := materialized.acquireExecutionLease()
	if err != nil {
		t.Fatal(err)
	}
	if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
		t.Fatalf("expected pending Close to report busy, got %v", err)
	}

	const callers = 32
	errorsChannel := make(chan error, callers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(callers)
	for range callers {
		go func() {
			defer waitGroup.Done()
			errorsChannel <- lease.release()
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)

	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent release failed: %v", err)
		}
	}
	if cleanupCalls.Load() != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleanupCalls.Load())
	}
	if materialized.leaseActive || !materialized.cleanupDone {
		t.Fatalf(
			"unexpected lifecycle: leaseActive=%v cleanupDone=%v",
			materialized.leaseActive,
			materialized.cleanupDone,
		)
	}
}

func TestMaterializedExecutableExecutionLeaseRejectsZeroAndMalformedState(t *testing.T) {
	var zero MaterializedExecutable
	if lease, err := zero.acquireExecutionLease(); !errors.Is(err, ErrMaterializedExecutableInvalid) || lease != nil {
		t.Fatalf("zero acquisition = (%v, %v), want nil ErrMaterializedExecutableInvalid", lease, err)
	}

	var nilExecutable *MaterializedExecutable
	if lease, err := nilExecutable.acquireExecutionLease(); !errors.Is(err, ErrMaterializedExecutableInvalid) || lease != nil {
		t.Fatalf("nil acquisition = (%v, %v), want nil ErrMaterializedExecutableInvalid", lease, err)
	}

	tests := map[string]func(*MaterializedExecutable){
		"directory":          func(e *MaterializedExecutable) { e.directory = " " },
		"path":               func(e *MaterializedExecutable) { e.path = " " },
		"size":               func(e *MaterializedExecutable) { e.expectedSize = 0 },
		"entrypoint module":  func(e *MaterializedExecutable) { e.entrypoint.Module = " " },
		"entrypoint version": func(e *MaterializedExecutable) { e.entrypoint.Version = " " },
		"signer":             func(e *MaterializedExecutable) { e.signerKeyID = " " },
		"target OS":          func(e *MaterializedExecutable) { e.targetOS = " " },
		"target arch":        func(e *MaterializedExecutable) { e.targetArch = " " },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			executable := validExecutionLeaseState()
			mutate(executable)
			lease, err := executable.acquireExecutionLease()
			if !errors.Is(err, ErrMaterializedExecutableInvalid) {
				t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
			}
			if lease != nil {
				t.Fatal("malformed state produced a lease")
			}
		})
	}
}

func TestMaterializedExecutableExecutionLeaseAcquisitionErrorPrecedence(t *testing.T) {
	tests := []struct {
		name string
		set  func(*MaterializedExecutable)
		want error
	}{
		{
			name: "cleanup done precedes busy and used",
			set: func(e *MaterializedExecutable) {
				e.cleanupDone = true
				e.leaseActive = true
				e.executionClaimed = true
			},
			want: ErrMaterializedExecutableClosed,
		},
		{
			name: "closed precedes busy and used",
			set: func(e *MaterializedExecutable) {
				e.closed = true
				e.leaseActive = true
				e.executionClaimed = true
			},
			want: ErrMaterializedExecutableClosed,
		},
		{
			name: "busy precedes used",
			set: func(e *MaterializedExecutable) {
				e.leaseActive = true
				e.executionClaimed = true
			},
			want: ErrMaterializedExecutableBusy,
		},
		{
			name: "used precedes malformed",
			set: func(e *MaterializedExecutable) {
				e.executionClaimed = true
				e.path = ""
			},
			want: ErrMaterializedExecutableAlreadyUsed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executable := validExecutionLeaseState()
			test.set(executable)
			lease, err := executable.acquireExecutionLease()
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
			if lease != nil {
				t.Fatal("error precedence test produced a lease")
			}
		})
	}
}

func TestMaterializedExecutableExecutionLeaseSnapshotsEvidence(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	wantDirectory := materialized.directory
	wantPath := materialized.path
	wantSize := materialized.expectedSize
	wantSHA256 := materialized.expectedSHA256
	wantEntrypoint := materialized.entrypoint
	wantSigner := materialized.signerKeyID
	wantOS := materialized.targetOS
	wantArch := materialized.targetArch

	lease, err := materialized.acquireExecutionLease()
	if err != nil {
		t.Fatal(err)
	}
	if lease.directory != wantDirectory || lease.path != wantPath ||
		lease.expectedSize != wantSize || lease.expectedSHA256 != wantSHA256 ||
		lease.entrypoint != wantEntrypoint || lease.signerKeyID != wantSigner ||
		lease.targetOS != wantOS || lease.targetArch != wantArch {
		t.Fatal("lease did not snapshot complete execution evidence")
	}

	if err := materialized.Close(); !errors.Is(err, ErrMaterializedExecutableBusy) {
		t.Fatalf("expected Close to report busy, got %v", err)
	}
	if err := lease.release(); err != nil {
		t.Fatal(err)
	}
	if lease.directory != wantDirectory || lease.path != wantPath {
		t.Fatal("cleanup mutated the lease snapshot")
	}
	if materialized.Entrypoint() != wantEntrypoint ||
		materialized.SignerKeyID() != wantSigner ||
		materialized.TargetOS() != wantOS ||
		materialized.TargetArch() != wantArch {
		t.Fatal("public metadata changed after pending cleanup")
	}
}

func TestExecutableLeaseReleaseRejectsNilOrIncompleteLease(t *testing.T) {
	var nilLease *executableLease
	if err := nilLease.release(); !errors.Is(err, ErrMaterializedExecutableInvalid) {
		t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
	}
	if err := (&executableLease{}).release(); !errors.Is(err, ErrMaterializedExecutableInvalid) {
		t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
	}
}

func validExecutionLeaseState() *MaterializedExecutable {
	data := []byte("execution-lease-state")
	directory := filepath.Join("private", "materialization")
	return &MaterializedExecutable{
		directory:      directory,
		path:           filepath.Join(directory, "application"),
		expectedSize:   int64(len(data)),
		expectedSHA256: sha256.Sum256(data),
		entrypoint: compiler.RuntimeEntrypoint{
			Module:  "lease-app",
			Version: "v1",
		},
		signerKeyID: "lease-test-signer",
		targetOS:    goruntime.GOOS,
		targetArch:  goruntime.GOARCH,
	}
}
