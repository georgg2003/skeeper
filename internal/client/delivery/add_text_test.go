package delivery

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/usecase"
)

func TestDelivery_AddText(t *testing.T) {
	meta := usecase.EntryMetadata{Name: "t", Notes: ""}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetText(gomock.Any(), meta, "hello", "mp").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	stdin := "t\n\nhello\n\nmp\n"
	cmd, out := testCmd(strings.NewReader(stdin))
	require.NoError(t, d.AddText(cmd, nil))
	assert.Contains(t, out.String(), "Encrypted text saved locally")
}
