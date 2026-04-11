package delivery

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
)

func TestDelivery_UpdateCard(t *testing.T) {
	id := uuid.New()
	meta := usecase.EntryMetadata{Name: "nm", Notes: "nt"}
	card := models.CardPayload{Holder: "H", Number: "4111", Expiry: "01/99", CVC: "999"}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().UpdateCard(gomock.Any(), id, meta, card, "mp").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("nm\nnt\nH\n4111\n01/99\n999\nmp\n"))
	require.NoError(t, d.UpdateCard(cmd, []string{id.String()}))
	assert.Contains(t, out.String(), "Entry updated locally")
}
