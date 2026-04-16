package delivery

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew_requiresAllPorts(t *testing.T) {
	ctrl := gomock.NewController(t)
	a, s, y := NewMockAuthCommands(ctrl), NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl)

	_, err := New(nil, s, y)
	require.Error(t, err)

	_, err = New(a, nil, y)
	require.Error(t, err)

	_, err = New(a, s, nil)
	require.Error(t, err)

	d, err := New(a, s, y)
	require.NoError(t, err)
	require.NotNil(t, d)
}
