package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestRepository_KDFSaltAndEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	r, err := New(path)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	ctx := context.Background()
	require.NoError(t, r.RunMigrations(ctx))

	uid := int64(1)
	s1, _, err := r.EnsureLocalVaultCrypto(ctx, uid)
	require.NoError(t, err)
	require.Len(t, s1, kdfSaltSize, "salt %+v", s1)
	s2, _, err := r.EnsureLocalVaultCrypto(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, string(s1), string(s2), "salt changed")

	id := uuid.New()
	e := models.Entry{
		UUID:         id,
		Type:         models.EntryTypePassword,
		EncryptedDek: []byte{1},
		Payload:      []byte{2},
		Meta:         []byte{3},
		Version:      1,
		UpdatedAt:    time.Now().Truncate(time.Second),
		UserID:       &uid,
	}
	require.NoError(t, r.SaveEntry(ctx, e, true))
	dirty, err := r.GetDirtyEntries(ctx, &uid)
	require.NoError(t, err)
	require.Len(t, dirty, 1)
	got, err := r.GetEntry(ctx, id, &uid)
	require.NoError(t, err)
	assert.Equal(t, id, got.UUID)
	assert.True(t, got.IsDirty)
	require.NoError(t, r.MarkAsSynced(ctx, id, uid))
	dirty, _ = r.GetDirtyEntries(ctx, &uid)
	assert.Empty(t, dirty, "expected no dirty")
	list, err := r.ListEntries(ctx, &uid)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestRepository_SessionEncryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.db")
	r, err := New(path)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	ctx := context.Background()
	require.NoError(t, r.RunMigrations(ctx))
	uid := int64(9)
	sess := models.Session{
		AccessToken:      "eyJhbGciOiJSUzI1NiJ9.access.sig",
		RefreshToken:     "a1b2c3d4e5f6789012345678901234567890abcdef0123456789abcdef012345",
		ExpiresAt:        time.Now().Add(time.Hour).UTC().Truncate(time.Second),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second),
		UserID:           &uid,
	}
	require.NoError(t, r.SaveSession(ctx, sess))
	got, err := r.GetSession(ctx)
	require.NoError(t, err)
	assert.Equal(t, sess.AccessToken, got.AccessToken)
	assert.Equal(t, sess.RefreshToken, got.RefreshToken)
	require.NotNil(t, got.UserID)
	assert.Equal(t, uid, *got.UserID)
}
