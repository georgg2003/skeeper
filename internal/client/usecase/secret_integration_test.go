package usecase

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/repository/db"
)

func openIntegrationVault(t *testing.T) (context.Context, *db.Repository, func()) {
	t.Helper()
	ctx := context.Background()
	r, err := db.New(filepath.Join(t.TempDir(), "vault.db"))
	require.NoError(t, err)
	require.NoError(t, r.RunMigrations(ctx))
	return ctx, r, func() { _ = r.Close() }
}

func sessionReaderForUser(ctrl *gomock.Controller, uid int64) *MockSessionReader {
	s := NewMockSessionReader(ctrl)
	s.EXPECT().GetSession(gomock.Any()).Return(&models.Session{UserID: &uid}, nil).AnyTimes()
	return s
}

func TestSecretUseCase_PasswordRoundTrip(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), DefaultMaxFileBytes)

	meta := EntryMetadata{Name: "svc", Notes: "n"}
	require.NoError(t, uc.SetPassword(ctx, meta, "secret-pw", "master!!"))

	entries, err := uc.ListLocal(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 1, "not saved")
	id := entries[0].UUID

	raw, err := uc.GetLocalEntry(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, models.EntryTypePassword, raw.Type)

	payload, gotMeta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Empty(t, orig, "unexpected orig name")
	assert.Equal(t, "secret-pw", string(payload))
	assert.Equal(t, "svc", gotMeta.Name)
}

func TestSecretUseCase_FileRoundTrip(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), 1<<20)

	meta := EntryMetadata{Name: "doc", Notes: "n"}
	content := []byte("hello skeeper")
	require.NoError(t, uc.SetFile(ctx, meta, "notes.txt", content, "master!!"))

	entries, err := uc.ListLocal(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 1, "not saved")
	id := entries[0].UUID
	assert.Equal(t, models.EntryTypeFile, entries[0].Type)
	assert.GreaterOrEqual(t, len(entries[0].Payload), 16, "expected ciphertext in payload")

	payload, gotMeta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, "notes.txt", orig)
	assert.Equal(t, "notes.txt", gotMeta.OriginalFilename)
	assert.Equal(t, string(content), string(payload))
	assert.Equal(t, "doc", gotMeta.Name)
}

func TestSecretUseCase_RejectsOtherMasterPasswordAfterFirstEntry(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), DefaultMaxFileBytes)

	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "a"}, "pw", "master-one"))
	err := uc.SetPassword(ctx, EntryMetadata{Name: "b"}, "pw2", "master-two")
	require.Error(t, err, "expected wrong master password")
	assert.True(t, errors.Is(err, ErrWrongMasterPassword))
}

func TestSecretUseCase_UpdatePassword_BumpsVersion(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), DefaultMaxFileBytes)

	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "n"}, "pw1", "master!!"))
	entries, err := uc.ListLocal(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	id := entries[0].UUID
	assert.Equal(t, int64(1), entries[0].Version)

	require.NoError(t, uc.UpdatePassword(ctx, id, EntryMetadata{Name: "n2"}, "pw2", "master!!"))

	entries, err = uc.ListLocal(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int64(2), entries[0].Version)

	payload, meta, _, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, "pw2", string(payload))
	assert.Equal(t, "n2", meta.Name)
}

func TestSecretUseCase_DeleteSoft(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), DefaultMaxFileBytes)

	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "x"}, "pw", "master!!"))
	entries, err := uc.ListLocal(ctx)
	require.NoError(t, err)
	id := entries[0].UUID

	require.NoError(t, uc.DeleteEntry(ctx, id, "master!!"))

	stored, err := uc.GetLocalEntry(ctx, id)
	require.NoError(t, err)
	assert.True(t, stored.IsDeleted)
	assert.Equal(t, int64(2), stored.Version)

	_, _, _, err = uc.GetDecryptedEntry(ctx, id, "master!!")
	assert.True(t, errors.Is(err, ErrEntryDeleted))
}

func TestSecretUseCase_UpdateFile_MetaOnly(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), 1<<20)

	content := []byte("payload-bytes")
	require.NoError(t, uc.SetFile(ctx, EntryMetadata{Name: "doc"}, "a.txt", content, "master!!"))
	entries, err := uc.ListLocal(ctx)
	require.NoError(t, err)
	id := entries[0].UUID

	require.NoError(t, uc.UpdateFile(ctx, id, EntryMetadata{Name: "doc2"}, "master!!", false, nil, ""))

	pl, meta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, string(content), string(pl))
	assert.Equal(t, "doc2", meta.Name)
	assert.Equal(t, "a.txt", orig)

	entries, err = uc.ListLocal(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int64(2), entries[0].Version)
}

func TestSecretUseCase_UpdateWrongEntryType(t *testing.T) {
	ctx, r, cleanup := openIntegrationVault(t)
	defer cleanup()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	uc := NewSecretUseCase(r, sessionReaderForUser(ctrl, uid), nil, discardClientLog(), DefaultMaxFileBytes)

	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "p"}, "pw", "master!!"))
	entries, err := uc.ListLocal(ctx)
	require.NoError(t, err)
	id := entries[0].UUID

	err = uc.UpdateText(ctx, id, EntryMetadata{Name: "t"}, "hello", "master!!")
	assert.True(t, errors.Is(err, ErrWrongEntryType))
}
