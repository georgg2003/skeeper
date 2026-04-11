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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, uc.SetPassword(ctx, meta, "secret-pw", "master!!"))
	require.Len(t, st.entries, 1, "not saved")
	id := st.entries[0].UUID
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
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, 1<<20)
	ctx := context.Background()
	meta := EntryMetadata{Name: "doc", Notes: "n"}
	content := []byte("hello skeeper")
	require.NoError(t, uc.SetFile(ctx, meta, "notes.txt", content, "master!!"))
	require.Len(t, st.entries, 1, "not saved")
	id := st.entries[0].UUID
	assert.Equal(t, models.EntryTypeFile, st.entries[0].Type)
	assert.GreaterOrEqual(t, len(st.entries[0].Payload), 16, "expected ciphertext in payload")
	payload, gotMeta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, "notes.txt", orig)
	assert.Equal(t, "notes.txt", gotMeta.OriginalFilename)
	assert.Equal(t, string(content), string(payload))
	assert.Equal(t, "doc", gotMeta.Name)
}

func TestSecretUseCase_FileTooLarge(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, 4)
	ctx := context.Background()
	err := uc.SetFile(ctx, EntryMetadata{Name: "x"}, "a.bin", []byte("12345"), "m")
	require.Error(t, err, "expected error")
}

func TestSecretUseCase_RejectsOtherMasterPasswordAfterFirstEntry(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	meta := EntryMetadata{Name: "a"}
	require.NoError(t, uc.SetPassword(ctx, meta, "pw", "master-one"))
	err := uc.SetPassword(ctx, EntryMetadata{Name: "b"}, "pw2", "master-two")
	require.Error(t, err, "expected wrong master password")
	assert.True(t, errors.Is(err, ErrWrongMasterPassword))
}

func TestSecretUseCase_UpdatePassword_BumpsVersion(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "n"}, "pw1", "master!!"))
	id := st.entries[0].UUID
	assert.Equal(t, int64(1), st.entries[0].Version)
	require.NoError(t, uc.UpdatePassword(ctx, id, EntryMetadata{Name: "n2"}, "pw2", "master!!"))
	require.Len(t, st.entries, 1)
	assert.Equal(t, int64(2), st.entries[0].Version)
	payload, meta, _, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, "pw2", string(payload))
	assert.Equal(t, "n2", meta.Name)
}

func TestSecretUseCase_DeleteSoft(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "x"}, "pw", "master!!"))
	id := st.entries[0].UUID
	require.NoError(t, uc.DeleteEntry(ctx, id, "master!!"))
	assert.True(t, st.entries[0].IsDeleted)
	assert.Equal(t, int64(2), st.entries[0].Version)
	_, _, _, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	assert.True(t, errors.Is(err, ErrEntryDeleted))
}

func TestSecretUseCase_UpdateFile_MetaOnly(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, 1<<20)
	ctx := context.Background()
	content := []byte("payload-bytes")
	require.NoError(t, uc.SetFile(ctx, EntryMetadata{Name: "doc"}, "a.txt", content, "master!!"))
	id := st.entries[0].UUID
	require.NoError(t, uc.UpdateFile(ctx, id, EntryMetadata{Name: "doc2"}, "master!!", false, nil, ""))
	pl, meta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, string(content), string(pl))
	assert.Equal(t, "doc2", meta.Name)
	assert.Equal(t, "a.txt", orig)
	assert.Equal(t, int64(2), st.entries[0].Version)
}

func TestSecretUseCase_UpdateWrongEntryType(t *testing.T) {
	st := &fakeSecretStore{}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := NewSecretUseCase(st, sessionReaderWithUser{uid: 1}, nil, log, DefaultMaxFileBytes)
	ctx := context.Background()
	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "p"}, "pw", "master!!"))
	id := st.entries[0].UUID
	err := uc.UpdateText(ctx, id, EntryMetadata{Name: "t"}, "hello", "master!!")
	assert.True(t, errors.Is(err, ErrWrongEntryType))
}
