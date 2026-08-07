package container

import "testing"

func TestFreeze(t *testing.T) {

	c := New()

	c.Freeze()

	err := c.Register(&struct{}{})

	if err != ErrContainerFrozen {
		t.Fatal("expected frozen error")
	}
}

func TestFrozen(t *testing.T) {

	c := New()

	if c.Frozen() {
		t.Fatal("container should not be frozen")
	}

	c.Freeze()

	if !c.Frozen() {
		t.Fatal("container should be frozen")
	}
}
