package app

import (
	"errors"
	"testing"
)

type addTestModule struct {
	name        string
	registerErr error
	registered  bool
	started     bool
	stopped     bool
}

func (m *addTestModule) Name() string {
	return m.name
}

func (m *addTestModule) Register(*App) error {
	m.registered = true
	return m.registerErr
}

func (m *addTestModule) Start(*App) error {
	m.started = true
	return nil
}

func (m *addTestModule) Stop(*App) error {
	m.stopped = true
	return nil
}

func TestAddRegistersModule(t *testing.T) {
	a := New()
	m := &addTestModule{name: "test"}

	if err := a.Add(m); err != nil {
		t.Fatalf("unexpected Add error: %v", err)
	}

	if !m.registered {
		t.Fatal("expected module Register to be called")
	}

	if !a.HasModule("test") {
		t.Fatal("expected module to be registered")
	}

	if len(a.Modules()) != 1 {
		t.Fatalf("expected 1 module, got %d", len(a.Modules()))
	}
}

func TestAddReturnsRegistrationError(t *testing.T) {
	a := New()
	want := errors.New("registration failed")

	m := &addTestModule{
		name:        "broken",
		registerErr: want,
	}

	err := a.Add(m)
	if !errors.Is(err, want) {
		t.Fatalf("expected registration error %v, got %v", want, err)
	}

	if !m.registered {
		t.Fatal("expected module Register to be called")
	}

	if a.HasModule("broken") {
		t.Fatal("module with failed registration must not be stored")
	}
}

func TestAddRejectsNilModule(t *testing.T) {
	a := New()

	err := a.Add(nil)
	if !errors.Is(err, ErrNilModule) {
		t.Fatalf("expected ErrNilModule, got %v", err)
	}
}
