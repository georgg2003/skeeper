package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError_Error(t *testing.T) {
	e := NewValidationError("email", "required")
	assert.Equal(t, "email is not valid: required", e.Error())
}

func TestWrap_Unwrap(t *testing.T) {
	base := New("root")
	w := Wrap(base, "ctx")
	assert.True(t, Is(w, base), "Is should follow unwrap chain")
}

func TestTypedError_Unwrap(t *testing.T) {
	inner := New("inner")
	e := NewTypedError("auth", inner)
	assert.True(t, errors.Is(e, inner), "unwrap failed")
}
