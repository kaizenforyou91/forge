package app

import "testing"

type TestModule struct{}

func (TestModule) Name() string {

	return "test"

}

func (TestModule) Register(app *App) error {

	return nil

}

func (TestModule) Start(app *App) error {

	return nil

}

func (TestModule) Stop(app *App) error {

	return nil

}

func TestHasModule(t *testing.T) {

	app := New()

	_ = app.Use(DemoModule{})

	if !app.HasModule("demo") {
		t.Fatal("module not found")
	}

	if app.HasModule("logger") {
		t.Fatal("unexpected module")
	}
}
