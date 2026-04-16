package delivery

import (
	"strings"
	"testing"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_Get_badUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	d, err := New(NewMockAuthCommands(ctrl), NewMockSecretCommands(ctrl), NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, _ := testCmd(strings.NewReader(""))
	require.Error(t, d.Get(cmd, []string{"not-a-uuid"}))
}

func TestDelivery_Get_passwordEntry(t *testing.T) {
	id := uuid.New()
	meta := models.EntryMetadata{Name: "svc", Notes: "n"}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().GetDecryptedEntry(gomock.Any(), id, "master").
		Return(models.DecryptedEntry{Payload: []byte("secret"), Meta: meta, Type: models.EntryTypePassword}, nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("master\n"))
	require.NoError(t, d.Get(cmd, []string{id.String()}))
	assert.Contains(t, out.String(), "Password: secret")
}
