package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	auth := NewMockAuthCommands(ctrl)
	secret := NewMockSecretCommands(ctrl)
	sync := NewMockSyncCommands(ctrl)
	auth.EXPECT().Login(gomock.Any(), "u@x.y", "pw").Return(nil)

	d, err := New(auth, secret, sync)
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("u@x.y\npw\n"))
	require.NoError(t, d.Login(cmd, nil))
	assert.Contains(t, out.String(), "Successfully logged in.")
}

func TestDelivery_Login_usecaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	auth := NewMockAuthCommands(ctrl)
	auth.EXPECT().Login(gomock.Any(), "u@x.y", "pw").Return(context.Canceled)
	d, err := New(auth, NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader("u@x.y\npw\n"))
	require.Error(t, d.Login(cmd, nil))
}
