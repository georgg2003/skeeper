package delivery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelivery_AddFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob.bin")
	require.NoError(t, os.WriteFile(path, []byte{1, 2, 3}, 0o600))
	meta := models.EntryMetadata{Name: "bin", Notes: ""}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetFile(gomock.Any(), meta, "blob.bin", []byte{1, 2, 3}, "m").Return(nil)
	d, err := New(NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl))
	require.NoError(t, err)
	cmd, out := testCmd(strings.NewReader("bin\n\nm\n"))
	require.NoError(t, d.AddFile(cmd, []string{path}))
	assert.Contains(t, out.String(), "Encrypted file saved")
}
