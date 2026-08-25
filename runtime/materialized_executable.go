package runtime

import (
	"fmt"
	"os"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

// MaterializedExecutable owns one private executable materialization and its
// cleanup lifecycle. Its filesystem paths are intentionally not exposed.
type MaterializedExecutable struct {
	mu sync.Mutex

	directory string
	path      string

	expectedSize   int64
	expectedSHA256 [32]byte

	entrypoint  compiler.RuntimeEntrypoint
	signerKeyID string
	targetOS    string
	targetArch  string

	closed      bool
	cleanupDone bool
	removeAll   func(string) error
}

func (e *MaterializedExecutable) Entrypoint() compiler.RuntimeEntrypoint {
	if e == nil {
		return compiler.RuntimeEntrypoint{}
	}

	return e.entrypoint
}

func (e *MaterializedExecutable) SignerKeyID() string {
	if e == nil {
		return ""
	}

	return e.signerKeyID
}

func (e *MaterializedExecutable) TargetOS() string {
	if e == nil {
		return ""
	}

	return e.targetOS
}

func (e *MaterializedExecutable) TargetArch() string {
	if e == nil {
		return ""
	}

	return e.targetArch
}

// Close marks the materialization closed and removes its entire private
// directory. Failed cleanup remains retryable; successful cleanup is
// idempotent. Metadata remains available after Close.
func (e *MaterializedExecutable) Close() error {
	if e == nil {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.closed = true
	if e.cleanupDone {
		return nil
	}

	removeAll := e.removeAll
	if removeAll == nil {
		removeAll = os.RemoveAll
	}

	if err := removeAll(e.directory); err != nil {
		return fmt.Errorf(
			"%w: remove private directory %q: %w",
			ErrExecutableMaterializationFailed,
			e.directory,
			err,
		)
	}

	e.cleanupDone = true
	e.directory = ""
	e.path = ""

	return nil
}
