package delivery

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestDelivery_AddCard(t *testing.T) {
	card := models.CardPayload{Holder: "H", Number: "4111", Expiry: "01/99", CVC: "999"}
	meta := models.EntryMetadata{Name: "c", Notes: ""}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetCard(gomock.Any(), meta, card, "master").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	stdin := "c\n\nH\n4111\n01/99\n999\nmaster\n"
	cmd, out := testCmd(strings.NewReader(stdin))
	require.NoError(t, d.AddCard(cmd, nil))
	assert.Contains(t, out.String(), "Encrypted card saved locally")
}
