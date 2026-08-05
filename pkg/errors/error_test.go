package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeInternal, "internal error")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWrap(t *testing.T) {
	base := errors.New("disk failure")

	err := Wrap(base, CodeInternal, "cannot read file")

	if err == nil {
		t.Fatal("expected wrapped error")
	}
}
