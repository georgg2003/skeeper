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

func TestDelivery_UpdatePassword(t *testing.T) {
	id := uuid.New()
	meta := models.EntryMetadata{Name: "nm", Notes: "nt"}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().UpdatePassword(gomock.Any(), id, meta, "newpw", "mp").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("nm\nnt\nnewpw\nmp\n"))
	require.NoError(t, d.UpdatePassword(cmd, []string{id.String()}))
	assert.Contains(t, out.String(), "Entry updated locally")
}
