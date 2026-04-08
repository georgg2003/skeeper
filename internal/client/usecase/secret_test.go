package usecase

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

type fakeSecretStore struct {
	salt     []byte
	verifier []byte
	entries  []models.Entry
}

func (f *fakeSecretStore) GetDirtyEntries(ctx context.Context, _ *int64) ([]models.Entry, error) {
	return nil, nil
}

func (f *fakeSecretStore) MarkAsSynced(ctx context.Context, id uuid.UUID, userID int64) error {
	_ = id
	_ = userID
	return nil
}

func (f *fakeSecretStore) SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error {
	f.entries = append(f.entries, e)
	return nil
}

func (f *fakeSecretStore) GetLastUpdate(ctx context.Context, _ *int64) (time.Time, error) {
	return time.Time{}, nil
}

func (f *fakeSecretStore) GetEntry(ctx context.Context, id uuid.UUID, _ *int64) (models.Entry, error) {
	for _, e := range f.entries {
		if e.UUID == id {
			return e, nil
		}
	}
	return models.Entry{}, context.Canceled
}

func (f *fakeSecretStore) EnsureLocalVaultCrypto(ctx context.Context, userID int64) ([]byte, []byte, error) {
	_ = userID
	if len(f.salt) == 0 {
		f.salt = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	}
	return f.salt, f.verifier, nil
}

func (f *fakeSecretStore) ReplaceLocalVaultCrypto(ctx context.Context, userID int64, salt, verifier []byte) error {
	_ = userID
	f.salt = append([]byte(nil), salt...)
	f.verifier = append([]byte(nil), verifier...)
	return nil
}

func (f *fakeSecretStore) SetLocalMasterVerifier(ctx context.Context, userID int64, verifier []byte) error {
	_ = userID
	f.verifier = append([]byte(nil), verifier...)
	return nil
}

func (f *fakeSecretStore) ListEntries(ctx context.Context, _ *int64) ([]models.Entry, error) {
	return f.entries, nil
}

func TestSecretUseCase_PasswordRoundTrip(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log)
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

func TestSecretUseCase_RejectsOtherMasterPasswordAfterFirstEntry(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log)
	ctx := context.Background()
	meta := EntryMetadata{Name: "a"}
	if err := uc.SetPassword(ctx, meta, "pw", "master-one"); err != nil {
		t.Fatal(err)
	}
	if err := uc.SetPassword(ctx, EntryMetadata{Name: "b"}, "pw2", "master-two"); err == nil {
		t.Fatal("expected wrong master password")
	} else if !errors.Is(err, ErrWrongMasterPassword) {
		t.Fatalf("want ErrWrongMasterPassword, got %v", err)
	}
}
