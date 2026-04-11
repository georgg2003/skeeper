package contextlib

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserIDRoundTrip(t *testing.T) {
	ctx := SetUserID(context.Background(), 42)
	id, ok := GetUserID(ctx)
	require.True(t, ok)
	require.Equal(t, int64(42), id)
}

func TestGetUserID_Missing(t *testing.T) {
	_, ok := GetUserID(context.Background())
	assert.False(t, ok, "expected no user id")
}
