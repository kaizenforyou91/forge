package app

import "testing"

func TestNewApp(t *testing.T) {

	app := New()

	if app == nil {
		t.Fatal("app is nil")
	}

	if app.Container() == nil {
		t.Fatal("container is nil")
	}

	if app.Started() {
		t.Fatal("expected app not started")
	}
}
