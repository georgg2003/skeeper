package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_Sync_ok(t *testing.T) {
	ctrl := gomock.NewController(t)
	sync := NewMockSyncCommands(ctrl)
	sync.EXPECT().Sync(gomock.Any()).Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), NewMockSecretCommands(ctrl), sync)
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader(""))
	require.NoError(t, d.Sync(cmd, nil))
	assert.Contains(t, out.String(), "Sync completed.")
}

func TestDelivery_Sync_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	sync := NewMockSyncCommands(ctrl)
	sync.EXPECT().Sync(gomock.Any()).Return(context.Canceled)
	d, err := New(NewMockAuthCommands(ctrl), NewMockSecretCommands(ctrl), sync)
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader(""))
	require.Error(t, d.Sync(cmd, nil))
}
