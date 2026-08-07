package app

import "testing"

func TestBootstrap(t *testing.T) {

	app := New().
		Use(&DemoModule{})

	if err := app.Run(); err != nil {
		t.Fatal(err)
	}

	if !app.Started() {
		t.Fatal("application not started")
	}
}
