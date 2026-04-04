package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
)

type fakeSecretStore struct {
	salt    []byte
	entries []models.Entry
}

func (f *fakeSecretStore) GetDirtyEntries(ctx context.Context) ([]models.Entry, error) {
	return nil, nil
}

func (f *fakeSecretStore) MarkAsSynced(ctx context.Context, id uuid.UUID) error { return nil }

func (f *fakeSecretStore) SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error {
	f.entries = append(f.entries, e)
	return nil
}

func (f *fakeSecretStore) GetLastUpdate(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}

func (f *fakeSecretStore) GetEntry(ctx context.Context, id uuid.UUID) (models.Entry, error) {
	for _, e := range f.entries {
		if e.UUID == id {
			return e, nil
		}
	}
	return models.Entry{}, context.Canceled
}

func (f *fakeSecretStore) GetOrCreateKDFSalt(ctx context.Context) ([]byte, error) {
	if len(f.salt) == 0 {
		f.salt = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	}
	return f.salt, nil
}

func (f *fakeSecretStore) ListEntries(ctx context.Context) ([]models.Entry, error) {
	return f.entries, nil
}

func TestSecretUseCase_PasswordRoundTrip(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, log)
	ctx := context.Background()
	meta := EntryMetadata{Name: "svc", Notes: "n"}
	if err := uc.SetPassword(ctx, meta, "secret-pw", "master!!"); err != nil {
		t.Fatal(err)
	}
	if len(st.entries) != 1 {
		t.Fatal("not saved")
	}
	id := st.entries[0].UUID
	raw, err := uc.GetLocalEntry(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if raw.Type != models.EntryTypePassword {
		t.Fatal(raw.Type)
	}
	payload, gotMeta, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "secret-pw" || gotMeta.Name != "svc" {
		t.Fatalf("%q %+v", payload, gotMeta)
	}
}
