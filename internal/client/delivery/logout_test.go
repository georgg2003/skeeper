package delivery

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	auth := NewMockAuthCommands(ctrl)
	auth.EXPECT().Logout(gomock.Any()).Return(nil)
	d, err := New(auth, NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader(""))
	require.NoError(t, d.Logout(cmd, nil))
	assert.Contains(t, out.String(), "Logged out.")
}
