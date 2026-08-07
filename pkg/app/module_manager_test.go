package app

import "testing"

type DemoModule struct{}

func (DemoModule) Name() string {

	return "demo"

}

func (DemoModule) Register(app *App) error {

	return nil

}

func (DemoModule) Start(app *App) error {

	return nil

}

func (DemoModule) Stop(app *App) error {

	return nil

}

func TestUseModule(t *testing.T) {

	app := New()

	returned := app.Use(DemoModule{})

	if returned != app {
		t.Fatal("expected fluent api")
	}

	if len(app.Modules()) != 1 {
		t.Fatal("expected one module")
	}

	if app.Modules()[0].Name() != "demo" {
		t.Fatal("unexpected module")
	}
}
