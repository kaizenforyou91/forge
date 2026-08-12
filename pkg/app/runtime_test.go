package app

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRuntimeContext(t *testing.T) {
	app := New()

	if app.Context() == nil {
		t.Fatal("context is nil")
	}
}

func TestStopBeforeStart(t *testing.T) {
	app := New()

	if err := app.Stop(); err != nil {
		t.Fatal(err)
	}

	if app.Started() {
		t.Fatal("expected app not started")
	}
}

func TestRuntimeResetAfterStop(t *testing.T) {
	app := New()
	module := &LifecycleModule{name: "a"}
	app.Use(module)

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	firstCtx := app.Context()

	if err := app.Stop(); err != nil {
		t.Fatal(err)
	}

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	secondCtx := app.Context()

	if firstCtx == secondCtx {
		t.Fatal("expected context to reset after stop")
	}
}

func TestConcurrentStartDeterministic(t *testing.T) {
	app := New()
	module := newBlockingStartModule()
	app.Use(module)

	firstErrCh := make(chan error, 1)
	go func() {
		firstErrCh <- app.Start()
	}()

	waitForChannel(t, module.startStarted, time.Second, "Start did not begin")

	const concurrent = 4
	resultCh := make(chan error, concurrent)
	startGate := make(chan struct{})
	ready := sync.WaitGroup{}
	ready.Add(concurrent)

	for i := 0; i < concurrent; i++ {
		go func() {
			ready.Done()
			<-startGate
			resultCh <- app.Start()
		}()
	}

	// wait until all concurrent callers are ready, then let them call Start
	ready.Wait()
	close(startGate)

	// collect concurrent results while first Start remains blocked
	var success, startingErrors int
	for i := 0; i < concurrent; i++ {
		err := <-resultCh
		if err == nil {
			success++
		} else if errors.Is(err, ErrAppStarting) {
			startingErrors++
		} else {
			t.Fatalf("unexpected start error: %v", err)
		}
	}

	// now let the original start complete
	close(module.startRelease)

	if err := <-firstErrCh; err != nil {
		t.Fatalf("expected first start to succeed, got %v", err)
	}

	if !app.Started() {
		t.Fatal("expected app started")
	}

	if atomic.LoadInt32(&module.startCount) != 1 {
		t.Fatalf("expected exactly one module start, got %d", module.startCount)
	}

	if success != 0 {
		t.Fatalf("expected 0 additional successful starts, got %d", success)
	}

	if startingErrors != concurrent {
		t.Fatalf("expected %d ErrAppStarting, got %d", concurrent, startingErrors)
	}
}

func TestConcurrentStopDeterministic(t *testing.T) {
	app := New()
	module := newBlockingStopModule()
	app.Use(module)

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	firstErrCh := make(chan error, 1)
	go func() {
		firstErrCh <- app.Stop()
	}()

	waitForChannel(t, module.stopStarted, time.Second, "Stop did not begin")

	const concurrent = 4
	resultCh := make(chan error, concurrent)
	stopGate := make(chan struct{})
	ready := sync.WaitGroup{}
	ready.Add(concurrent)

	for i := 0; i < concurrent; i++ {
		go func() {
			ready.Done()
			<-stopGate
			resultCh <- app.Stop()
		}()
	}

	ready.Wait()
	close(stopGate)

	// collect concurrent results while first Stop remains blocked
	for i := 0; i < concurrent; i++ {
		if err := <-resultCh; err != nil {
			t.Fatalf("unexpected stop error: %v", err)
		}
	}

	// now release the first stop
	close(module.stopRelease)

	if err := <-firstErrCh; err != nil {
		t.Fatalf("expected first stop to succeed, got %v", err)
	}

	if app.Started() {
		t.Fatal("expected app stopped")
	}

	if atomic.LoadInt32(&module.stopCount) != 1 {
		t.Fatalf("expected exactly one module stop, got %d", module.stopCount)
	}
}

func TestStartStopInteractionStartWins(t *testing.T) {
	app := New()
	module := newBlockingStartModule()
	app.Use(module)

	startErrCh := make(chan error, 1)
	stopErrCh := make(chan error, 1)

	go func() {
		startErrCh <- app.Start()
	}()

	waitForChannel(t, module.startStarted, time.Second, "Start did not begin")

	go func() {
		stopErrCh <- app.Stop()
	}()

	select {
	case err := <-stopErrCh:
		if !errors.Is(err, ErrAppStarting) {
			t.Fatalf("expected ErrAppStarting, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop did not return quickly")
	}

	close(module.startRelease)

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("unexpected start error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("start did not complete")
	}

	if !app.Started() {
		t.Fatal("expected app running")
	}
}

func TestStartStopInteractionStopWins(t *testing.T) {
	app := New()
	module := newBlockingStopModule()
	app.Use(module)

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	module.stopStarted = make(chan struct{})
	module.stopRelease = make(chan struct{})
	module.stopCount = 0

	stopErrCh := make(chan error, 1)
	startErrCh := make(chan error, 1)

	go func() {
		stopErrCh <- app.Stop()
	}()

	waitForChannel(t, module.stopStarted, time.Second, "Stop did not begin")

	go func() {
		startErrCh <- app.Start()
	}()

	select {
	case err := <-startErrCh:
		if !errors.Is(err, ErrAppStopping) {
			t.Fatalf("expected ErrAppStopping, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("start did not return quickly")
	}

	close(module.stopRelease)

	select {
	case err := <-stopErrCh:
		if err != nil {
			t.Fatalf("unexpected stop error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop did not complete")
	}

	if app.Started() {
		t.Fatal("expected app stopped")
	}
}

func TestStartFailureRollsBackModules(t *testing.T) {
	app := New()

	moduleA := &RollbackModule{}
	moduleB := &FailModule{}
	moduleC := &LifecycleModule{name: "c"}

	app.Use(moduleA)
	app.Use(moduleB)
	app.Use(moduleC)

	err := app.Start()
	if err == nil {
		t.Fatal("expected start failure")
	}

	if !moduleA.started {
		t.Fatal("expected module A to have started")
	}

	if !moduleA.stopped {
		t.Fatal("expected module A to have stopped on rollback")
	}

	if moduleC.started {
		t.Fatal("expected module C not to have started")
	}

	if app.Started() {
		t.Fatal("expected app not started after failed start")
	}
}

func TestMultipleStopFailures(t *testing.T) {
	app := New()

	moduleA := &StopFailModule{name: "a"}
	moduleB := &StopFailModule{name: "b"}

	app.Use(moduleA)
	app.Use(moduleB)

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	err := app.Stop()
	if err == nil {
		t.Fatal("expected stop error")
	}

	if !errors.Is(err, moduleA.err) {
		t.Fatalf("expected error to contain %v", moduleA.err)
	}

	if !errors.Is(err, moduleB.err) {
		t.Fatalf("expected error to contain %v", moduleB.err)
	}
}

func TestContextSemanticsDuringStartAndStop(t *testing.T) {
	app := New()

	module := &ContextModule{t: t}
	app.Use(module)

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	if err := app.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestModulesSnapshotSafety(t *testing.T) {
	app := New()
	moduleA := &LifecycleModule{name: "a"}
	moduleB := &LifecycleModule{name: "b"}
	app.Use(moduleA)
	app.Use(moduleB)

	modules := app.Modules()
	modules[0] = nil

	if app.Modules()[0] == nil {
		t.Fatal("Modules() returned a live slice")
	}
}

func TestDuplicateModuleRegistration(t *testing.T) {
	app := New()
	moduleA := &LifecycleModule{name: "dup"}
	moduleB := &LifecycleModule{name: "dup"}

	app.Use(moduleA)
	app.Use(moduleB)

	if !app.HasModule("dup") {
		t.Fatal("expected module names to be present")
	}

	if len(app.Modules()) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(app.Modules()))
	}
}

type LifecycleModule struct {
	name    string
	started bool
	stopped bool
}

func (m *LifecycleModule) Name() string {
	return m.name
}

func (m *LifecycleModule) Register(app *App) error {
	return nil
}

func (m *LifecycleModule) Start(app *App) error {
	m.started = true
	return nil
}

func (m *LifecycleModule) Stop(app *App) error {
	m.stopped = true
	return nil
}

type RollbackModule struct {
	started bool
	stopped bool
}

func (m *RollbackModule) Name() string {
	return "rollback"
}

func (m *RollbackModule) Register(app *App) error {
	return nil
}

func (m *RollbackModule) Start(app *App) error {
	m.started = true
	return nil
}

func (m *RollbackModule) Stop(app *App) error {
	m.stopped = true
	return nil
}

type FailModule struct {
	started bool
}

func (m *FailModule) Name() string {
	return "fail"
}

func (m *FailModule) Register(app *App) error {
	return nil
}

func (m *FailModule) Start(app *App) error {
	m.started = true
	return errors.New("failed to start")
}

func (m *FailModule) Stop(app *App) error {
	return nil
}

type StopFailModule struct {
	name string
	err  error
}

func (m *StopFailModule) Name() string {
	return m.name
}

func (m *StopFailModule) Register(app *App) error {
	return nil
}

func (m *StopFailModule) Start(app *App) error {
	return nil
}

func (m *StopFailModule) Stop(app *App) error {
	if m.err == nil {
		m.err = errors.New("stop failure")
	}
	return m.err
}

type ContextModule struct {
	t *testing.T
}

func (m *ContextModule) Name() string {
	return "context"
}

func (m *ContextModule) Register(app *App) error {
	return nil
}

func (m *ContextModule) Start(app *App) error {
	if app.Context().Err() != nil {
		m.t.Fatal("expected context to be active during start")
	}
	return nil
}

func (m *ContextModule) Stop(app *App) error {
	select {
	case <-app.Context().Done():
		return nil
	case <-time.After(time.Second):
		m.t.Fatal("expected context to be cancelled during stop")
	}
	return nil
}

type blockingStartModule struct {
	startOnce    sync.Once
	startStarted chan struct{}
	startRelease chan struct{}
	startCount   int32
}

func newBlockingStartModule() *blockingStartModule {
	return &blockingStartModule{
		startStarted: make(chan struct{}),
		startRelease: make(chan struct{}),
	}
}

func (m *blockingStartModule) Name() string {
	return "blocking-start"
}

func (m *blockingStartModule) Register(app *App) error {
	return nil
}

func (m *blockingStartModule) Start(app *App) error {
	atomic.AddInt32(&m.startCount, 1)
	m.startOnce.Do(func() {
		close(m.startStarted)
	})
	<-m.startRelease
	return nil
}

func (m *blockingStartModule) Stop(app *App) error {
	return nil
}

type blockingStopModule struct {
	stopOnce     sync.Once
	stopStarted  chan struct{}
	stopRelease  chan struct{}
	stopCount    int32
}

func newBlockingStopModule() *blockingStopModule {
	return &blockingStopModule{
		stopStarted: make(chan struct{}),
		stopRelease: make(chan struct{}),
	}
}

func (m *blockingStopModule) Name() string {
	return "blocking-stop"
}

func (m *blockingStopModule) Register(app *App) error {
	return nil
}

func (m *blockingStopModule) Start(app *App) error {
	return nil
}

func (m *blockingStopModule) Stop(app *App) error {
	atomic.AddInt32(&m.stopCount, 1)
	m.stopOnce.Do(func() {
		close(m.stopStarted)
	})
	<-m.stopRelease
	return nil
}

func appendResult(results []error, err error, mu *sync.Mutex) []error {
	mu.Lock()
	defer mu.Unlock()
	return append(results, err)
}

func waitForChannel(t *testing.T, ch <-chan struct{}, timeout time.Duration, message string) {
	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatal(message)
	}
}
