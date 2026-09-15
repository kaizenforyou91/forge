package runtime

import (
	"errors"
	"reflect"
	"sync"
)

// processScopePlatform is exclusively the control/release capability for one
// privately owned scope. Implementations must not call back into their owner.
// terminate success acknowledges a control request, never descendant quiescence.
// finalize must finish its resource-release attempt before returning, including
// on failure; the owner will not retry or detach that operation.
type processScopePlatform interface {
	terminate() error
	finalize() error
}

type processScopePhase uint8

const (
	processScopeInvalid processScopePhase = iota
	processScopePrepared
	processScopeActive
	processScopeFinalized
)

type processScopeControlCause uint8

const (
	processScopeNoControl processScopeControlCause = iota
	processScopeManualTermination
	processScopeCancellation
	processScopeNaturalExitCleanup
)

var (
	errProcessScopeInvalid             = errors.New("process scope is nil or incomplete")
	errProcessScopeState               = errors.New("process scope operation is invalid in this phase")
	errProcessScopeCause               = errors.New("process scope control cause is invalid")
	errProcessScopeControlInterrupted  = errors.New("process scope control did not return")
	errProcessScopeFinalizeInterrupted = errors.New("process scope finalization did not return")
)

// processScopeOwner must be used by pointer and must not be copied. It starts
// no goroutine and owns no PID, result, output, lease, or native platform logic.
// The mutex serializes entire platform calls, so finalization cannot retire a
// capability while control is using it. A blocking platform blocks its callers.
type processScopeOwner struct {
	mu                 sync.Mutex
	platform           processScopePlatform
	phase              processScopePhase
	winner             processScopeControlCause
	controlInterrupted bool
	// Keep the first returned control failure even if a later request succeeds.
	// This bounded diagnostic is not a history or a success/quiescence indicator.
	controlFailure  error
	finalizeFailure error
}

func newProcessScopeOwner(platform processScopePlatform) (*processScopeOwner, error) {
	if platform == nil {
		return nil, errProcessScopeInvalid
	}
	v := reflect.ValueOf(platform)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil, errProcessScopeInvalid
		}
	}
	// Both operations are required by the static interface; incomplete platform
	// implementations cannot be passed to this constructor.
	return &processScopeOwner{platform: platform, phase: processScopePrepared}, nil
}

func (s *processScopeOwner) activate() error {
	if s == nil {
		return errProcessScopeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase == processScopeInvalid {
		return errProcessScopeInvalid
	}
	if s.phase != processScopePrepared {
		return errProcessScopeState
	}
	s.phase = processScopeActive
	return nil
}

// request records only the first successful control cause. Natural-exit cleanup
// is deliberately distinct from direct-child cancellation/manual termination.
// An ordinary error permits another caller's later request, never an automatic
// retry. All requests after a winner are no-ops; inspect status for the winner.
func (s *processScopeOwner) request(cause processScopeControlCause) error {
	if s == nil {
		return errProcessScopeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase == processScopeInvalid {
		return errProcessScopeInvalid
	}
	if s.phase != processScopeActive {
		return errProcessScopeState
	}
	if cause != processScopeManualTermination && cause != processScopeCancellation && cause != processScopeNaturalExitCleanup {
		return errProcessScopeCause
	}
	if s.controlInterrupted {
		return errProcessScopeControlInterrupted
	}
	if s.winner != processScopeNoControl {
		return nil
	}
	// No recover: a trusted panic propagates unchanged. Leave control disabled
	// after unwinding because the partial platform action has unknown effects.
	s.controlInterrupted = true
	err := s.platform.terminate()
	s.controlInterrupted = false
	if err != nil {
		if s.controlFailure == nil {
			s.controlFailure = err
		}
		return err
	}
	s.winner = cause
	return nil
}

// finalize closes control admission and releases resources exactly once, even
// for PREPARED partial starts, without an implicit termination request. Future
// integration owns the ordering of active control and this terminal operation.
// FINALIZED means the release attempt ended, not that every descendant exited
// or that release succeeded; its cached error remains authoritative.
func (s *processScopeOwner) finalize() error {
	if s == nil {
		return errProcessScopeInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase == processScopeInvalid {
		return errProcessScopeInvalid
	}
	if s.phase == processScopeFinalized {
		return s.finalizeFailure
	}
	s.phase = processScopeFinalized
	s.finalizeFailure = errProcessScopeFinalizeInterrupted
	defer func() { s.platform = nil }()
	s.finalizeFailure = s.platform.finalize()
	return s.finalizeFailure
}

type processScopeStatus struct {
	phase              processScopePhase
	winner             processScopeControlCause
	controlFailure     error
	controlInterrupted bool
	finalizeFailure    error
}

// status observes completed operations; it waits behind in-flight platform work.
func (s *processScopeOwner) status() processScopeStatus {
	if s == nil {
		return processScopeStatus{phase: processScopeInvalid}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return processScopeStatus{
		phase: s.phase, winner: s.winner,
		controlFailure: s.controlFailure, controlInterrupted: s.controlInterrupted,
		finalizeFailure: s.finalizeFailure,
	}
}
