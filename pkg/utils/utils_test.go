package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashToken_Deterministic(t *testing.T) {
	a := HashToken("refresh-me")
	b := HashToken("refresh-me")
	assert.Equal(t, a, b, "hash not stable")
	assert.NotEqual(t, a, HashToken("other"), "collision")
}
