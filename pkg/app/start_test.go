package app

import "testing"

type StartModule struct {
	started bool
}

func (m *StartModule) Name() string {
	return "start"
}

func (m *StartModule) Register(app *App) error {
	return nil
}

func (m *StartModule) Start(app *App) error {
	m.started = true
	return nil
}

func (m *StartModule) Stop(app *App) error {
	return nil
}

func TestStart(t *testing.T) {

	app := New()

	module := &StartModule{}

	returned := app.Use(module)

	if returned != app {
		t.Fatal("expected fluent api")
	}

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	if !app.Started() {
		t.Fatal("application not started")
	}

	if !module.started {
		t.Fatal("module not started")
	}
}

func TestStartTwice(t *testing.T) {

	app := New()

	module := &StartModule{}

	_ = app.Use(module)

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}

	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
}
