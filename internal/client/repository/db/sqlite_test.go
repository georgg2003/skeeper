package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestRepository_KDFSaltAndEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	r, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	ctx := context.Background()
	if err := r.RunMigrations(ctx); err != nil {
		t.Fatal(err)
	}

	uid := int64(1)
	s1, _, err := r.EnsureLocalVaultCrypto(ctx, uid)
	if err != nil || len(s1) != kdfSaltSize {
		t.Fatalf("salt %+v err %v", s1, err)
	}
	s2, _, err := r.EnsureLocalVaultCrypto(ctx, uid)
	if err != nil {
		t.Fatal(err)
	}
	if string(s1) != string(s2) {
		t.Fatal("salt changed")
	}

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
	if err := r.SaveEntry(ctx, e, true); err != nil {
		t.Fatal(err)
	}
	dirty, err := r.GetDirtyEntries(ctx, &uid)
	if err != nil || len(dirty) != 1 {
		t.Fatalf("dirty %+v err %v", dirty, err)
	}
	got, err := r.GetEntry(ctx, id, &uid)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != id || !got.IsDirty {
		t.Fatalf("%+v", got)
	}
	if err := r.MarkAsSynced(ctx, id, uid); err != nil {
		t.Fatal(err)
	}
	dirty, _ = r.GetDirtyEntries(ctx, &uid)
	if len(dirty) != 0 {
		t.Fatal("expected no dirty")
	}
	list, err := r.ListEntries(ctx, &uid)
	if err != nil || len(list) != 1 {
		t.Fatalf("list %+v", list)
	}
}

func TestRepository_SessionEncryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.db")
	r, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	ctx := context.Background()
	if err := r.RunMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	uid := int64(9)
	sess := models.Session{
		AccessToken:      "eyJhbGciOiJSUzI1NiJ9.access.sig",
		RefreshToken:     "a1b2c3d4e5f6789012345678901234567890abcdef0123456789abcdef012345",
		ExpiresAt:        time.Now().Add(time.Hour).UTC().Truncate(time.Second),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second),
		UserID:           &uid,
	}
	if err := r.SaveSession(ctx, sess); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != sess.AccessToken || got.RefreshToken != sess.RefreshToken {
		t.Fatalf("tokens mismatch: %+v", got)
	}
	if got.UserID == nil || *got.UserID != uid {
		t.Fatalf("user id %+v", got.UserID)
	}
}
