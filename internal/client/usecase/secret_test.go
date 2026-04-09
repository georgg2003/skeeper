package usecase

import (
	"context"
	"database/sql"
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
	ne := e
	ne.IsDirty = isDirty
	for i := range f.entries {
		if f.entries[i].UUID == ne.UUID {
			f.entries[i] = ne
			return nil
		}
	}
	f.entries = append(f.entries, ne)
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
	return models.Entry{}, sql.ErrNoRows
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
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
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
	payload, gotMeta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	if err != nil {
		t.Fatal(err)
	}
	if orig != "" {
		t.Fatalf("unexpected orig name %q", orig)
	}
	if string(payload) != "secret-pw" || gotMeta.Name != "svc" {
		t.Fatalf("%q %+v", payload, gotMeta)
	}
}

func TestSecretUseCase_FileRoundTrip(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, 1<<20)
	ctx := context.Background()
	meta := EntryMetadata{Name: "doc", Notes: "n"}
	content := []byte("hello skeeper")
	if err := uc.SetFile(ctx, meta, "notes.txt", content, "master!!"); err != nil {
		t.Fatal(err)
	}
	if len(st.entries) != 1 {
		t.Fatal("not saved")
	}
	id := st.entries[0].UUID
	if st.entries[0].Type != models.EntryTypeFile {
		t.Fatalf("type %s", st.entries[0].Type)
	}
	if len(st.entries[0].Payload) < 16 {
		t.Fatalf("expected ciphertext in payload, got len %d", len(st.entries[0].Payload))
	}
	payload, gotMeta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	if err != nil {
		t.Fatal(err)
	}
	if orig != "notes.txt" {
		t.Fatalf("orig %q", orig)
	}
	if gotMeta.OriginalFilename != "notes.txt" {
		t.Fatalf("meta original_filename %q", gotMeta.OriginalFilename)
	}
	if string(payload) != string(content) || gotMeta.Name != "doc" {
		t.Fatalf("%q %+v", payload, gotMeta)
	}
}

func TestSecretUseCase_FileTooLarge(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, 4)
	ctx := context.Background()
	err := uc.SetFile(ctx, EntryMetadata{Name: "x"}, "a.bin", []byte("12345"), "m")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSecretUseCase_RejectsOtherMasterPasswordAfterFirstEntry(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
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

func TestSecretUseCase_UpdatePassword_BumpsVersion(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	if err := uc.SetPassword(ctx, EntryMetadata{Name: "n"}, "pw1", "master!!"); err != nil {
		t.Fatal(err)
	}
	id := st.entries[0].UUID
	if st.entries[0].Version != 1 {
		t.Fatalf("version %d", st.entries[0].Version)
	}
	if err := uc.UpdatePassword(ctx, id, EntryMetadata{Name: "n2"}, "pw2", "master!!"); err != nil {
		t.Fatal(err)
	}
	if len(st.entries) != 1 {
		t.Fatalf("entries %d", len(st.entries))
	}
	if st.entries[0].Version != 2 {
		t.Fatalf("version %d", st.entries[0].Version)
	}
	payload, meta, _, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "pw2" || meta.Name != "n2" {
		t.Fatalf("%q %+v", payload, meta)
	}
}

func TestSecretUseCase_DeleteSoft(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	if err := uc.SetPassword(ctx, EntryMetadata{Name: "x"}, "pw", "master!!"); err != nil {
		t.Fatal(err)
	}
	id := st.entries[0].UUID
	if err := uc.DeleteEntry(ctx, id, "master!!"); err != nil {
		t.Fatal(err)
	}
	if !st.entries[0].IsDeleted || st.entries[0].Version != 2 {
		t.Fatalf("%+v", st.entries[0])
	}
	if _, _, _, err := uc.GetDecryptedEntry(ctx, id, "master!!"); !errors.Is(err, ErrEntryDeleted) {
		t.Fatalf("got %v", err)
	}
}

func TestSecretUseCase_UpdateFile_MetaOnly(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, 1<<20)
	ctx := context.Background()
	content := []byte("payload-bytes")
	if err := uc.SetFile(ctx, EntryMetadata{Name: "doc"}, "a.txt", content, "master!!"); err != nil {
		t.Fatal(err)
	}
	id := st.entries[0].UUID
	if err := uc.UpdateFile(ctx, id, EntryMetadata{Name: "doc2"}, "master!!", false, nil, ""); err != nil {
		t.Fatal(err)
	}
	pl, meta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	if err != nil {
		t.Fatal(err)
	}
	if string(pl) != string(content) || meta.Name != "doc2" || orig != "a.txt" {
		t.Fatalf("%q %+v %q", pl, meta, orig)
	}
	if st.entries[0].Version != 2 {
		t.Fatalf("version %d", st.entries[0].Version)
	}
}

func TestSecretUseCase_UpdateWrongEntryType(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	if err := uc.SetPassword(ctx, EntryMetadata{Name: "p"}, "pw", "master!!"); err != nil {
		t.Fatal(err)
	}
	id := st.entries[0].UUID
	err := uc.UpdateText(ctx, id, EntryMetadata{Name: "t"}, "hello", "master!!")
	if !errors.Is(err, ErrWrongEntryType) {
		t.Fatalf("got %v", err)
	}
}
