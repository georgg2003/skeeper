package errors

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	e := NewValidationError("email", "required")
	if e.Error() != "email is not valid: required" {
		t.Fatalf("got %q", e.Error())
	}
}

func TestWrap_Unwrap(t *testing.T) {
	base := New("root")
	w := Wrap(base, "ctx")
	if !Is(w, base) {
		t.Fatal("Is should follow unwrap chain")
	}
}

func TestTypedError_Unwrap(t *testing.T) {
	inner := New("inner")
	e := NewTypedError("auth", inner)
	if !errors.Is(e, inner) {
		t.Fatal("unwrap failed")
	}
}
