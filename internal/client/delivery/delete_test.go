package delivery

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_Delete(t *testing.T) {
	id := uuid.New()
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().DeleteEntry(gomock.Any(), id, "mp").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("mp\n"))
	require.NoError(t, d.Delete(cmd, []string{id.String()}))
	assert.Contains(t, out.String(), "Entry deleted locally")
}
