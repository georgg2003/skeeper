package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/usecase"
)

func TestDelivery_AddPassword_success(t *testing.T) {
	meta := usecase.EntryMetadata{Name: "svc", Notes: "n"}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetPassword(gomock.Any(), meta, "sec", "master").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("svc\nn\nsec\nmaster\n"))
	require.NoError(t, d.AddPassword(cmd, nil))
	assert.Contains(t, out.String(), "Encrypted entry saved locally")
}

func TestDelivery_AddPassword_usecaseError(t *testing.T) {
	meta := usecase.EntryMetadata{Name: "svc", Notes: "n"}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetPassword(gomock.Any(), meta, "sec", "master").Return(context.Canceled)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader("svc\nn\nsec\nmaster\n"))
	require.Error(t, d.AddPassword(cmd, nil))
}
