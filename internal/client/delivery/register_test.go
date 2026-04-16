package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_Register_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	auth := NewMockAuthCommands(ctrl)
	auth.EXPECT().Register(gomock.Any(), "a@b.c", "pw").Return(nil)
	d, err := New(auth, NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("a@b.c\npw\npw\n"))
	require.NoError(t, d.Register(cmd, nil))
	assert.Contains(t, out.String(), "Registered and logged in.")
}

func TestDelivery_Register_passwordMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	d, err := New(NewMockAuthCommands(ctrl), NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader("a@b.c\np1\np2\n"))
	require.Error(t, d.Register(cmd, nil))
}

func TestDelivery_Register_usecaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	auth := NewMockAuthCommands(ctrl)
	auth.EXPECT().Register(gomock.Any(), "a@b.c", "pw").Return(context.Canceled)
	d, err := New(auth, NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader("a@b.c\npw\npw\n"))
	require.Error(t, d.Register(cmd, nil))
}
