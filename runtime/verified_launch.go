package runtime

import (
	"errors"
	"reflect"
	"sync"
)

// verifiedLaunchPlatform owns only the opaque platform binding resources for
// one launch attempt. finalize must synchronously finish its one release attempt
// before returning. Native admission belongs to later platform packages.
type verifiedLaunchPlatform interface {
	finalize() error
}

type verifiedLaunchPhase uint8

const (
	verifiedLaunchInvalid verifiedLaunchPhase = iota
	verifiedLaunchPrepared
	verifiedLaunchFinalized
)

var (
	errVerifiedLaunchInvalid             = errors.New("verified launch owner is nil or incomplete")
	errVerifiedLaunchState               = errors.New("verified launch operation is invalid in this phase")
	errVerifiedLaunchNativeCreated       = errors.New("native creation was already recorded")
	errVerifiedLaunchAdmissionProven     = errors.New("verified admission was already recorded")
	errVerifiedLaunchAdmissionOrder      = errors.New("verified admission requires native creation evidence")
	errVerifiedLaunchFinalizeInterrupted = errors.New("verified launch finalization did not return")
)

// verifiedLaunchOwner is pointer-owned, single-use, and must not be copied. It
// starts no goroutine and owns no path, descriptor, handle, PID, process result,
// lease, or native launch operation. The mutex serializes evidence transitions,
// status reads, and the complete platform finalization call.
type verifiedLaunchOwner struct {
	mu sync.Mutex

	platform verifiedLaunchPlatform
	phase    verifiedLaunchPhase

	nativeCreated   bool
	admissionProven bool

	finalizationStarted     bool
	finalizationInterrupted bool
	finalizationFailure     error
}

func newVerifiedLaunchOwner(platform verifiedLaunchPlatform) (*verifiedLaunchOwner, error) {
	if platform == nil {
		return nil, errVerifiedLaunchInvalid
	}
	v := reflect.ValueOf(platform)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil, errVerifiedLaunchInvalid
		}
	}
	return &verifiedLaunchOwner{platform: platform, phase: verifiedLaunchPrepared}, nil
}

func (v *verifiedLaunchOwner) recordNativeCreated() error {
	if v == nil {
		return errVerifiedLaunchInvalid
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.phase == verifiedLaunchInvalid {
		return errVerifiedLaunchInvalid
	}
	if v.phase != verifiedLaunchPrepared {
		return errVerifiedLaunchState
	}
	if v.platform == nil {
		return errVerifiedLaunchInvalid
	}
	if v.nativeCreated {
		return errVerifiedLaunchNativeCreated
	}
	v.nativeCreated = true
	return nil
}

func (v *verifiedLaunchOwner) recordAdmissionProven() error {
	if v == nil {
		return errVerifiedLaunchInvalid
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.phase == verifiedLaunchInvalid {
		return errVerifiedLaunchInvalid
	}
	if v.phase != verifiedLaunchPrepared {
		return errVerifiedLaunchState
	}
	if v.platform == nil {
		return errVerifiedLaunchInvalid
	}
	if !v.nativeCreated {
		return errVerifiedLaunchAdmissionOrder
	}
	if v.admissionProven {
		return errVerifiedLaunchAdmissionProven
	}
	v.admissionProven = true
	return nil
}

// finalize retires binding authority exactly once. FINALIZED records that the
// one release attempt began; it does not imply admission proof or successful
// release. A trusted platform panic propagates, while the initially installed
// interrupted evidence and retired reference prevent a second attempt.
func (v *verifiedLaunchOwner) finalize() error {
	if v == nil {
		return errVerifiedLaunchInvalid
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.phase == verifiedLaunchInvalid || (!v.finalizationStarted && v.platform == nil) {
		return errVerifiedLaunchInvalid
	}
	if v.phase == verifiedLaunchFinalized {
		if v.finalizationInterrupted {
			return errVerifiedLaunchFinalizeInterrupted
		}
		return v.finalizationFailure
	}

	v.phase = verifiedLaunchFinalized
	v.finalizationStarted = true
	v.finalizationInterrupted = true
	platform := v.platform
	defer func() { v.platform = nil }()
	v.finalizationFailure = platform.finalize()
	v.finalizationInterrupted = false
	return v.finalizationFailure
}

type verifiedLaunchStatus struct {
	phase verifiedLaunchPhase

	nativeCreated   bool
	admissionProven bool

	finalizationStarted     bool
	finalizationInterrupted bool
	finalizationFailure     error
}

// status returns immutable-by-value evidence and never exposes platform
// authority. It waits behind any in-flight transition or finalization call.
func (v *verifiedLaunchOwner) status() verifiedLaunchStatus {
	if v == nil {
		return verifiedLaunchStatus{phase: verifiedLaunchInvalid}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return verifiedLaunchStatus{
		phase:                   v.phase,
		nativeCreated:           v.nativeCreated,
		admissionProven:         v.admissionProven,
		finalizationStarted:     v.finalizationStarted,
		finalizationInterrupted: v.finalizationInterrupted,
		finalizationFailure:     v.finalizationFailure,
	}
}
