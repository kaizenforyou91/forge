package app

import "testing"

func TestRuntimeContext(t *testing.T) {

	app := New()

	if app.Context() == nil {

		t.Fatal("context is nil")

	}
}
