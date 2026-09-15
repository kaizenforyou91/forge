package runtime

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type fakeProcessScopePlatform struct {
	terminateCalls atomic.Int32
	finalizeCalls  atomic.Int32
	control        func() error
	release        func() error
}

func (f *fakeProcessScopePlatform) terminate() error {
	f.terminateCalls.Add(1)
	if f.control != nil {
		return f.control()
	}
	return nil
}

func (f *fakeProcessScopePlatform) finalize() error {
	f.finalizeCalls.Add(1)
	if f.release != nil {
		return f.release()
	}
	return nil
}

func scopeOwnerForTest(t *testing.T, f *fakeProcessScopePlatform, active bool) *processScopeOwner {
	t.Helper()
	s, err := newProcessScopeOwner(f)
	if err != nil {
		t.Fatal(err)
	}
	if active {
		if err := s.activate(); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func assertScopeCalls(t *testing.T, f *fakeProcessScopePlatform, control, release int32) {
	t.Helper()
	if got := f.terminateCalls.Load(); got != control {
		t.Fatalf("terminate calls = %d, want %d", got, control)
	}
	if got := f.finalizeCalls.Load(); got != release {
		t.Fatalf("finalize calls = %d, want %d", got, release)
	}
}

func TestProcessScopeConstructionAndPartialStart(t *testing.T) {
	var typedNil *fakeProcessScopePlatform
	for _, p := range []processScopePlatform{nil, typedNil} {
		if s, err := newProcessScopeOwner(p); s != nil || !errors.Is(err, errProcessScopeInvalid) {
			t.Fatalf("invalid platform accepted: %v", err)
		}
	}
	for _, s := range []*processScopeOwner{nil, {}} {
		if !errors.Is(s.activate(), errProcessScopeInvalid) || !errors.Is(s.request(processScopeManualTermination), errProcessScopeInvalid) || !errors.Is(s.finalize(), errProcessScopeInvalid) {
			t.Fatal("nil/incomplete owner accepted")
		}
	}
	f := &fakeProcessScopePlatform{}
	s := scopeOwnerForTest(t, f, false)
	if s.status().phase != processScopePrepared {
		t.Fatal("not prepared")
	}
	assertScopeCalls(t, f, 0, 0)
	if !errors.Is(s.request(processScopeManualTermination), errProcessScopeState) {
		t.Fatal("prepared control accepted")
	}
	if err := s.finalize(); err != nil {
		t.Fatal(err)
	}
	if err := s.finalize(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(s.activate(), errProcessScopeState) {
		t.Fatal("reactivation accepted")
	}
	if !errors.Is(s.request(processScopeCancellation), errProcessScopeState) {
		t.Fatal("retired control accepted")
	}
	if s.status().phase != processScopeFinalized || s.platform != nil {
		t.Fatal("references not retired")
	}
	assertScopeCalls(t, f, 0, 1)
}

func TestProcessScopeActivationAndControlCauses(t *testing.T) {
	for _, cause := range []processScopeControlCause{processScopeManualTermination, processScopeCancellation, processScopeNaturalExitCleanup} {
		t.Run(map[processScopeControlCause]string{processScopeManualTermination: "manual", processScopeCancellation: "cancel", processScopeNaturalExitCleanup: "natural"}[cause], func(t *testing.T) {
			f := &fakeProcessScopePlatform{}
			s := scopeOwnerForTest(t, f, true)
			if !errors.Is(s.activate(), errProcessScopeState) {
				t.Fatal("second activation accepted")
			}
			for _, invalid := range []processScopeControlCause{processScopeNoControl, 255} {
				if !errors.Is(s.request(invalid), errProcessScopeCause) {
					t.Fatal("invalid cause accepted")
				}
			}
			assertScopeCalls(t, f, 0, 0)
			if err := s.request(cause); err != nil {
				t.Fatal(err)
			}
			for _, later := range []processScopeControlCause{processScopeManualTermination, processScopeCancellation, processScopeNaturalExitCleanup} {
				if err := s.request(later); err != nil {
					t.Fatal(err)
				}
			}
			status := s.status()
			// Success is not finalization/quiescence. In particular natural cleanup
			// records neither of the two direct-child control classifications.
			if status.winner != cause || status.phase != processScopeActive {
				t.Fatal("control meaning changed")
			}
			assertScopeCalls(t, f, 1, 0)
			if err := s.finalize(); err != nil {
				t.Fatal(err)
			}
			if err := s.finalize(); err != nil {
				t.Fatal(err)
			}
			if s.status().winner != cause || s.platform != nil {
				t.Fatal("terminal status/reference mismatch")
			}
			if !errors.Is(s.request(cause), errProcessScopeState) {
				t.Fatal("control after finalization accepted")
			}
			assertScopeCalls(t, f, 1, 1)
		})
	}
}

func TestProcessScopeFailedControlCanBeFollowedByAnotherRequest(t *testing.T) {
	failure := errors.New("fake control failure")
	f := &fakeProcessScopePlatform{}
	f.control = func() error {
		if f.terminateCalls.Load() == 1 {
			return failure
		}
		return nil
	}
	s := scopeOwnerForTest(t, f, true)
	if err := s.request(processScopeManualTermination); err != failure {
		t.Fatal("failure lost")
	}
	status := s.status()
	if status.winner != processScopeNoControl || status.controlFailure != failure || status.controlInterrupted {
		t.Fatal("failed control published success")
	}
	assertScopeCalls(t, f, 1, 0)
	if err := s.request(processScopeCancellation); err != nil {
		t.Fatal(err)
	}
	if s.status().winner != processScopeCancellation || s.status().controlFailure != failure {
		t.Fatal("winner/failure evidence lost")
	}
	if err := s.finalize(); err != nil {
		t.Fatal(err)
	}
	assertScopeCalls(t, f, 2, 1)
}

func TestProcessScopeConcurrentControlWinner(t *testing.T) {
	for _, pair := range [][2]processScopeControlCause{
		{processScopeManualTermination, processScopeCancellation},
		{processScopeManualTermination, processScopeNaturalExitCleanup},
		{processScopeCancellation, processScopeNaturalExitCleanup},
	} {
		f := &fakeProcessScopePlatform{}
		s := scopeOwnerForTest(t, f, true)
		start := make(chan struct{})
		results := make(chan error, 32)
		var wg sync.WaitGroup
		for i := range 32 {
			wg.Add(1)
			go func(cause processScopeControlCause) { defer wg.Done(); <-start; results <- s.request(cause) }(pair[i%2])
		}
		close(start)
		wg.Wait()
		close(results)
		for err := range results {
			if err != nil {
				t.Fatal(err)
			}
		}
		winner := s.status().winner
		if winner != pair[0] && winner != pair[1] {
			t.Fatal("no winner")
		}
		assertScopeCalls(t, f, 1, 0)
		if err := s.finalize(); err != nil {
			t.Fatal(err)
		}
		assertScopeCalls(t, f, 1, 1)
	}
}

func TestProcessScopeConcurrentFinalizationCachesFailure(t *testing.T) {
	for _, active := range []bool{false, true} {
		failure := errors.New("fake release failure")
		f := &fakeProcessScopePlatform{release: func() error { return failure }}
		s := scopeOwnerForTest(t, f, active)
		start := make(chan struct{})
		results := make(chan error, 32)
		var wg sync.WaitGroup
		for range 32 {
			wg.Add(1)
			go func() { defer wg.Done(); <-start; results <- s.finalize() }()
		}
		close(start)
		wg.Wait()
		close(results)
		for err := range results {
			if err != failure {
				t.Fatal("cached failure changed")
			}
		}
		if s.status().finalizeFailure != failure || s.platform != nil {
			t.Fatal("finalization evidence/release missing")
		}
		if !errors.Is(s.request(processScopeManualTermination), errProcessScopeState) {
			t.Fatal("control after failed finalization accepted")
		}
		assertScopeCalls(t, f, 0, 1)
	}
}

func TestProcessScopeControlBeforeFinalizeOrdering(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var returned atomic.Bool
	f := &fakeProcessScopePlatform{control: func() error { close(entered); <-release; returned.Store(true); return nil }}
	f.release = func() error {
		if !returned.Load() {
			return errors.New("finalization overtook control")
		}
		return nil
	}
	s := scopeOwnerForTest(t, f, true)
	controlDone := make(chan error, 1)
	go func() { controlDone <- s.request(processScopeManualTermination) }()
	<-entered // The owner holds its lock inside control, not merely before calling it.
	if s.mu.TryLock() {
		s.mu.Unlock()
		close(release)
		t.Fatal("platform control is not serialized")
	}
	finalizeStarted, finalizeDone := make(chan struct{}), make(chan error, 1)
	go func() { close(finalizeStarted); finalizeDone <- s.finalize() }()
	<-finalizeStarted
	assertScopeCalls(t, f, 1, 0)
	close(release)
	if err := <-controlDone; err != nil {
		t.Fatal(err)
	}
	if err := <-finalizeDone; err != nil {
		t.Fatal(err)
	}
	if s.status().winner != processScopeManualTermination {
		t.Fatal("winner lost")
	}
	assertScopeCalls(t, f, 1, 1)
}

func TestProcessScopeFinalizeBeforeControlAndActivation(t *testing.T) {
	for _, active := range []bool{false, true} {
		entered, release := make(chan struct{}), make(chan struct{})
		f := &fakeProcessScopePlatform{release: func() error { close(entered); <-release; return nil }}
		s := scopeOwnerForTest(t, f, active)
		finalizeDone := make(chan error, 1)
		go func() { finalizeDone <- s.finalize() }()
		<-entered
		if s.mu.TryLock() {
			s.mu.Unlock()
			close(release)
			t.Fatal("platform finalization is not serialized")
		}
		controlDone, activateDone := make(chan error, 1), make(chan error, 1)
		go func() { controlDone <- s.request(processScopeCancellation) }()
		go func() { activateDone <- s.activate() }()
		assertScopeCalls(t, f, 0, 1)
		close(release)
		if err := <-finalizeDone; err != nil {
			t.Fatal(err)
		}
		if !errors.Is(<-controlDone, errProcessScopeState) || !errors.Is(<-activateDone, errProcessScopeState) {
			t.Fatal("operation after retirement accepted")
		}
		if s.platform != nil {
			t.Fatal("platform retained")
		}
		assertScopeCalls(t, f, 0, 1)
	}
}

func TestProcessScopeConcurrentActivation(t *testing.T) {
	f := &fakeProcessScopePlatform{}
	s := scopeOwnerForTest(t, f, false)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.activate() == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal("activation did not have one winner")
	}
	if err := s.finalize(); err != nil {
		t.Fatal(err)
	}
	assertScopeCalls(t, f, 0, 1)
}

func TestProcessScopePanicDoesNotReuseControlOrFinalization(t *testing.T) {
	marker := &struct{}{}
	assertPanic := func(action func()) {
		t.Helper()
		defer func() {
			if got := recover(); got != marker {
				t.Errorf("original panic changed: %v", got)
			}
		}()
		action()
		t.Error("panic swallowed")
	}
	f := &fakeProcessScopePlatform{control: func() error { panic(marker) }}
	s := scopeOwnerForTest(t, f, true)
	assertPanic(func() { _ = s.request(processScopeManualTermination) })
	if !s.status().controlInterrupted || s.status().winner != processScopeNoControl {
		t.Fatal("panic published success")
	}
	if !errors.Is(s.request(processScopeCancellation), errProcessScopeControlInterrupted) {
		t.Fatal("panicked control reused")
	}
	if err := s.finalize(); err != nil {
		t.Fatal(err)
	}
	assertScopeCalls(t, f, 1, 1)

	f = &fakeProcessScopePlatform{release: func() error { panic(marker) }}
	s = scopeOwnerForTest(t, f, false)
	assertPanic(func() { _ = s.finalize() })
	if !errors.Is(s.finalize(), errProcessScopeFinalizeInterrupted) {
		t.Fatal("interrupted finalization lost")
	}
	if s.status().phase != processScopeFinalized || s.platform != nil {
		t.Fatal("panic retained/reopened owner")
	}
	if !errors.Is(s.activate(), errProcessScopeState) || !errors.Is(s.request(processScopeNaturalExitCleanup), errProcessScopeState) {
		t.Fatal("panic retired owner reused")
	}
	assertScopeCalls(t, f, 0, 1)
}
