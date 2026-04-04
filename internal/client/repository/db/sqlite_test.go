package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
)

func TestRepository_KDFSaltAndEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	r, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	ctx := context.Background()
	if err := r.RunMigrations(ctx); err != nil {
		t.Fatal(err)
	}

	s1, err := r.GetOrCreateKDFSalt(ctx)
	if err != nil || len(s1) != kdfSaltSize {
		t.Fatalf("salt %+v err %v", s1, err)
	}
	s2, err := r.GetOrCreateKDFSalt(ctx)
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
	}
	if err := r.SaveEntry(ctx, e, true); err != nil {
		t.Fatal(err)
	}
	dirty, err := r.GetDirtyEntries(ctx, nil)
	if err != nil || len(dirty) != 1 {
		t.Fatalf("dirty %+v err %v", dirty, err)
	}
	got, err := r.GetEntry(ctx, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != id || !got.IsDirty {
		t.Fatalf("%+v", got)
	}
	if err := r.MarkAsSynced(ctx, id); err != nil {
		t.Fatal(err)
	}
	dirty, _ = r.GetDirtyEntries(ctx, nil)
	if len(dirty) != 0 {
		t.Fatal("expected no dirty")
	}
	list, err := r.ListEntries(ctx, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("list %+v", list)
	}
}
