package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

// executableLease is the package-private, single-use capability that future
// process-runner code will hold from executable acquisition through process
// wait/reap. Its filesystem evidence is intentionally not publicly exposed.
type executableLease struct {
	owner *MaterializedExecutable

	path           string
	directory      string
	expectedSize   int64
	expectedSHA256 [32]byte

	entrypoint  compiler.RuntimeEntrypoint
	signerKeyID string
	targetOS    string
	targetArch  string

	releaseOnce sync.Once
	releaseErr  error
}

// acquireExecutionLease atomically consumes the materialization's sole
// execution claim and snapshots the private evidence needed by a future
// process runner. It deliberately performs no filesystem or process work.
func (e *MaterializedExecutable) acquireExecutionLease() (*executableLease, error) {
	if e == nil {
		return nil, fmt.Errorf(
			"%w: materialized executable is nil",
			ErrMaterializedExecutableInvalid,
		)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cleanupDone || e.closed {
		return nil, fmt.Errorf(
			"%w: execution acquisition is unavailable after Close",
			ErrMaterializedExecutableClosed,
		)
	}
	if e.leaseActive {
		return nil, fmt.Errorf(
			"%w: an execution lease is already active",
			ErrMaterializedExecutableBusy,
		)
	}
	if e.executionClaimed {
		return nil, fmt.Errorf(
			"%w: the single execution claim has been consumed",
			ErrMaterializedExecutableAlreadyUsed,
		)
	}
	if !e.validForExecutionLeaseLocked() {
		return nil, fmt.Errorf(
			"%w: internal execution evidence is incomplete",
			ErrMaterializedExecutableInvalid,
		)
	}

	// This mutation is the linearization point against Close and competing
	// acquisition attempts. executionClaimed is intentionally never reset.
	e.executionClaimed = true
	e.leaseActive = true

	return &executableLease{
		owner:          e,
		path:           e.path,
		directory:      e.directory,
		expectedSize:   e.expectedSize,
		expectedSHA256: e.expectedSHA256,
		entrypoint:     e.entrypoint,
		signerKeyID:    e.signerKeyID,
		targetOS:       e.targetOS,
		targetArch:     e.targetArch,
	}, nil
}

func (e *MaterializedExecutable) validForExecutionLeaseLocked() bool {
	return strings.TrimSpace(e.directory) != "" &&
		strings.TrimSpace(e.path) != "" &&
		e.expectedSize > 0 &&
		strings.TrimSpace(e.entrypoint.Module) != "" &&
		strings.TrimSpace(e.entrypoint.Version) != "" &&
		strings.TrimSpace(e.signerKeyID) != "" &&
		strings.TrimSpace(e.targetOS) != "" &&
		strings.TrimSpace(e.targetArch) != ""
}

// release relinquishes the active lease exactly once. When Close was already
// requested, the first release also performs the pending directory cleanup.
// A failed pending cleanup remains retryable through MaterializedExecutable.Close.
func (l *executableLease) release() error {
	if l == nil || l.owner == nil {
		return fmt.Errorf(
			"%w: execution lease is nil or incomplete",
			ErrMaterializedExecutableInvalid,
		)
	}

	l.releaseOnce.Do(func() {
		owner := l.owner
		owner.mu.Lock()
		defer owner.mu.Unlock()

		if !owner.leaseActive {
			l.releaseErr = fmt.Errorf(
				"%w: execution lease owner is not active",
				ErrMaterializedExecutableInvalid,
			)
			return
		}

		owner.leaseActive = false
		if owner.closed && !owner.cleanupDone {
			l.releaseErr = owner.cleanupLocked()
		}
	})

	return l.releaseErr
}
