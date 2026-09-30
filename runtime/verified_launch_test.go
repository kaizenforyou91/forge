package runtime

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type fakeVerifiedLaunchPlatform struct {
	finalizeCalls atomic.Int32
	release       func() error
}

func (f *fakeVerifiedLaunchPlatform) finalize() error {
	f.finalizeCalls.Add(1)
	if f.release != nil {
		return f.release()
	}
	return nil
}

func verifiedLaunchOwnerForTest(t *testing.T, f *fakeVerifiedLaunchPlatform) *verifiedLaunchOwner {
	t.Helper()
	v, err := newVerifiedLaunchOwner(f)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestVerifiedLaunchConstructionAndInvalidOwner(t *testing.T) {
	var typedNil *fakeVerifiedLaunchPlatform
	for _, platform := range []verifiedLaunchPlatform{nil, typedNil} {
		if owner, err := newVerifiedLaunchOwner(platform); owner != nil || !errors.Is(err, errVerifiedLaunchInvalid) {
			t.Fatalf("invalid platform accepted: %v", err)
		}
	}

	for _, owner := range []*verifiedLaunchOwner{nil, {}} {
		if !errors.Is(owner.recordNativeCreated(), errVerifiedLaunchInvalid) ||
			!errors.Is(owner.recordAdmissionProven(), errVerifiedLaunchInvalid) ||
			!errors.Is(owner.finalize(), errVerifiedLaunchInvalid) {
			t.Fatal("nil/incomplete owner accepted")
		}
		if owner.status().phase != verifiedLaunchInvalid {
			t.Fatal("invalid owner published usable state")
		}
	}

	f := &fakeVerifiedLaunchPlatform{}
	owner := verifiedLaunchOwnerForTest(t, f)
	got := owner.status()
	if got.phase != verifiedLaunchPrepared || got.nativeCreated || got.admissionProven || got.finalizationStarted || got.finalizationInterrupted || got.finalizationFailure != nil {
		t.Fatalf("new owner status = %+v", got)
	}
}

func TestVerifiedLaunchAdmissionEvidenceOrdering(t *testing.T) {
	f := &fakeVerifiedLaunchPlatform{}
	owner := verifiedLaunchOwnerForTest(t, f)
	if !errors.Is(owner.recordAdmissionProven(), errVerifiedLaunchAdmissionOrder) {
		t.Fatal("admission accepted before native creation")
	}
	if err := owner.recordNativeCreated(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(owner.recordNativeCreated(), errVerifiedLaunchNativeCreated) {
		t.Fatal("second native-created transition accepted")
	}
	status := owner.status()
	if !status.nativeCreated || status.admissionProven {
		t.Fatal("native creation implied admission")
	}
	if err := owner.recordAdmissionProven(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(owner.recordAdmissionProven(), errVerifiedLaunchAdmissionProven) {
		t.Fatal("second admission transition accepted")
	}
	status = owner.status()
	if !status.nativeCreated || !status.admissionProven {
		t.Fatal("admission evidence lost")
	}
}

func TestVerifiedLaunchFinalizationPreservesEvidence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		created  bool
		admitted bool
	}{
		{name: "prepared"},
		{name: "created", created: true},
		{name: "admitted", created: true, admitted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeVerifiedLaunchPlatform{}
			owner := verifiedLaunchOwnerForTest(t, f)
			if tc.created {
				if err := owner.recordNativeCreated(); err != nil {
					t.Fatal(err)
				}
			}
			if tc.admitted {
				if err := owner.recordAdmissionProven(); err != nil {
					t.Fatal(err)
				}
			}
			if err := owner.finalize(); err != nil {
				t.Fatal(err)
			}
			if err := owner.finalize(); err != nil {
				t.Fatal(err)
			}
			status := owner.status()
			if status.phase != verifiedLaunchFinalized || status.nativeCreated != tc.created || status.admissionProven != tc.admitted || !status.finalizationStarted || status.finalizationInterrupted || status.finalizationFailure != nil {
				t.Fatalf("final status = %+v", status)
			}
			if owner.platform != nil || f.finalizeCalls.Load() != 1 {
				t.Fatal("platform authority not retired exactly once")
			}
			if !errors.Is(owner.recordNativeCreated(), errVerifiedLaunchState) || !errors.Is(owner.recordAdmissionProven(), errVerifiedLaunchState) {
				t.Fatal("finalized owner accepted evidence mutation")
			}
		})
	}
}

func TestVerifiedLaunchFinalizationFailureIsCached(t *testing.T) {
	failure := errors.New("binding release failed")
	f := &fakeVerifiedLaunchPlatform{release: func() error { return failure }}
	owner := verifiedLaunchOwnerForTest(t, f)
	if err := owner.recordNativeCreated(); err != nil {
		t.Fatal(err)
	}
	if err := owner.finalize(); err != failure {
		t.Fatal("finalization failure lost")
	}
	if err := owner.finalize(); err != failure {
		t.Fatal("cached failure changed")
	}
	status := owner.status()
	if status.finalizationFailure != failure || status.finalizationInterrupted || status.admissionProven {
		t.Fatalf("failure evidence changed: %+v", status)
	}
	if f.finalizeCalls.Load() != 1 || owner.platform != nil {
		t.Fatal("failed finalization retried or retained platform")
	}
}

func TestVerifiedLaunchConcurrentEvidenceTransitions(t *testing.T) {
	f := &fakeVerifiedLaunchPlatform{}
	owner := verifiedLaunchOwnerForTest(t, f)
	start := make(chan struct{})
	results := make(chan error, 32)
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- owner.recordNativeCreated() }()
	}
	close(start)
	wg.Wait()
	close(results)
	var nativeWins int
	for err := range results {
		if err == nil {
			nativeWins++
			continue
		}
		if !errors.Is(err, errVerifiedLaunchNativeCreated) {
			t.Fatal(err)
		}
	}
	if nativeWins != 1 {
		t.Fatalf("native-created winners = %d", nativeWins)
	}

	start = make(chan struct{})
	results = make(chan error, 32)
	for range 32 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- owner.recordAdmissionProven() }()
	}
	close(start)
	wg.Wait()
	close(results)
	var admissionWins int
	for err := range results {
		if err == nil {
			admissionWins++
			continue
		}
		if !errors.Is(err, errVerifiedLaunchAdmissionProven) {
			t.Fatal(err)
		}
	}
	if admissionWins != 1 {
		t.Fatalf("admission winners = %d", admissionWins)
	}
	if err := owner.finalize(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifiedLaunchConcurrentFinalizationExactlyOnce(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	f := &fakeVerifiedLaunchPlatform{release: func() error { close(entered); <-release; return nil }}
	owner := verifiedLaunchOwnerForTest(t, f)
	start := make(chan struct{})
	results := make(chan error, 32)
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- owner.finalize() }()
	}
	close(start)
	<-entered
	if owner.mu.TryLock() {
		owner.mu.Unlock()
		close(release)
		t.Fatal("platform finalization is not serialized")
	}
	if f.finalizeCalls.Load() != 1 {
		t.Fatal("duplicate in-flight finalization")
	}
	close(release)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.finalizeCalls.Load() != 1 || owner.platform != nil {
		t.Fatal("finalization was not exactly once")
	}
}

func TestVerifiedLaunchFinalizeSerializesEvidenceMutation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	f := &fakeVerifiedLaunchPlatform{release: func() error { close(entered); <-release; return nil }}
	owner := verifiedLaunchOwnerForTest(t, f)
	finalized := make(chan error, 1)
	go func() { finalized <- owner.finalize() }()
	<-entered
	nativeResult := make(chan error, 1)
	admissionResult := make(chan error, 1)
	go func() { nativeResult <- owner.recordNativeCreated() }()
	go func() { admissionResult <- owner.recordAdmissionProven() }()
	close(release)
	if err := <-finalized; err != nil {
		t.Fatal(err)
	}
	if !errors.Is(<-nativeResult, errVerifiedLaunchState) || !errors.Is(<-admissionResult, errVerifiedLaunchState) {
		t.Fatal("evidence mutation overtook finalization")
	}
	status := owner.status()
	if status.nativeCreated || status.admissionProven {
		t.Fatal("finalization fabricated evidence")
	}
}

func TestVerifiedLaunchPanicPoisonsFinalization(t *testing.T) {
	marker := &struct{}{}
	f := &fakeVerifiedLaunchPlatform{release: func() error { panic(marker) }}
	owner := verifiedLaunchOwnerForTest(t, f)
	func() {
		defer func() {
			if got := recover(); got != marker {
				t.Fatalf("original panic changed: %v", got)
			}
		}()
		_ = owner.finalize()
		t.Fatal("panic swallowed")
	}()
	status := owner.status()
	if status.phase != verifiedLaunchFinalized || !status.finalizationStarted || !status.finalizationInterrupted || status.finalizationFailure != nil {
		t.Fatalf("panic evidence = %+v", status)
	}
	if owner.platform != nil || f.finalizeCalls.Load() != 1 {
		t.Fatal("panic retained/reused platform authority")
	}
	if !errors.Is(owner.finalize(), errVerifiedLaunchFinalizeInterrupted) {
		t.Fatal("interrupted result not cached")
	}
	if f.finalizeCalls.Load() != 1 {
		t.Fatal("panicked finalization retried")
	}
	if !errors.Is(owner.recordNativeCreated(), errVerifiedLaunchState) || !errors.Is(owner.recordAdmissionProven(), errVerifiedLaunchState) {
		t.Fatal("panicked owner reused")
	}
}
