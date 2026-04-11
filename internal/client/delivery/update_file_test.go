package delivery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/usecase"
)

func TestDelivery_UpdateFile_metadataOnly(t *testing.T) {
	id := uuid.New()
	meta := usecase.EntryMetadata{Name: "nm", Notes: "nt"}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().UpdateFile(gomock.Any(), id, meta, "mp", false, nil, "").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("nm\nnt\n\nmp\n"))
	require.NoError(t, d.UpdateFile(cmd, []string{id.String()}))
	assert.Contains(t, out.String(), "Entry updated locally")
}

func TestDelivery_UpdateFile_replaceBytes(t *testing.T) {
	id := uuid.New()
	meta := usecase.EntryMetadata{Name: "f", Notes: ""}
	dir := t.TempDir()
	p := filepath.Join(dir, "x.dat")
	require.NoError(t, os.WriteFile(p, []byte{9, 9}, 0o600))
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().UpdateFile(gomock.Any(), id, meta, "mp", true, []byte{9, 9}, "x.dat").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("f\n\n" + p + "\nmp\n"))
	require.NoError(t, d.UpdateFile(cmd, []string{id.String()}))
	assert.Contains(t, out.String(), "Entry updated locally")
}
