package app

import "testing"

func TestBuilder(t *testing.T) {

	builder := NewBuilder()

	if builder == nil {
		t.Fatal("builder is nil")
	}

	app := builder.Build()

	if app == nil {
		t.Fatal("app is nil")
	}

	if app.Container() == nil {
		t.Fatal("container is nil")
	}
}
