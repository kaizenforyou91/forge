package app

import "testing"

type StopModule struct {
	stopped bool
}

func (m *StopModule) Name() string {
	return "stop"
}

func (m *StopModule) Register(app *App) error {
	return nil
}

func (m *StopModule) Start(app *App) error {
	return nil
}

func (m *StopModule) Stop(app *App) error {
	m.stopped = true
	return nil
}

func TestStop(t *testing.T) {

	app := New()

	module := &StopModule{}

	returned := app.Use(module)

	if returned != app {
		t.Fatal("expected fluent api")
	}

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	if err := app.Stop(); err != nil {
		t.Fatal(err)
	}

	if app.Started() {
		t.Fatal("application still started")
	}

	if !module.stopped {
		t.Fatal("module not stopped")
	}
}

func TestStopTwice(t *testing.T) {

	app := New()

	module := &StopModule{}

	_ = app.Use(module)

	_ = app.Start()

	if err := app.Stop(); err != nil {
		t.Fatal(err)
	}

	if err := app.Stop(); err != nil {
		t.Fatal(err)
	}
}
