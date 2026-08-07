package app

import "testing"

func TestRun(t *testing.T) {

	app := New()

	if err := app.Run(); err != nil {
		t.Fatal(err)
	}

	if !app.Started() {
		t.Fatal("application not started")
	}
}

func TestRunTwice(t *testing.T) {

	app := New()

	if err := app.Run(); err != nil {
		t.Fatal(err)
	}

	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
}
