package usecase

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

// vaultMem holds mutable vault rows for [MockLocalSecretStore] expectations (in-memory double).
type vaultMem struct {
	mu       sync.Mutex
	entries  []models.Entry
	salt     []byte
	verifier []byte
}

func (v *vaultMem) wireInto(m *MockLocalSecretStore) {
	m.EXPECT().GetDirtyEntries(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	m.EXPECT().MarkAsSynced(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	m.EXPECT().GetLastUpdate(gomock.Any(), gomock.Any()).AnyTimes().Return(time.Time{}, nil)
	m.EXPECT().ReplaceLocalVaultCrypto(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	m.EXPECT().ListEntries(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, _ *int64) ([]models.Entry, error) {
		v.mu.Lock()
		defer v.mu.Unlock()
		out := make([]models.Entry, len(v.entries))
		copy(out, v.entries)
		return out, nil
	})
	m.EXPECT().EnsureLocalVaultCrypto(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, _ int64) ([]byte, []byte, error) {
		v.mu.Lock()
		defer v.mu.Unlock()
		if len(v.salt) == 0 {
			v.salt = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
		}
		return append([]byte(nil), v.salt...), append([]byte(nil), v.verifier...), nil
	})
	m.EXPECT().SaveEntry(gomock.Any(), gomock.AssignableToTypeOf(models.Entry{}), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, e models.Entry, isDirty bool) error {
			v.mu.Lock()
			defer v.mu.Unlock()
			ne := e
			ne.IsDirty = isDirty
			for i := range v.entries {
				if v.entries[i].UUID == ne.UUID {
					v.entries[i] = ne
					return nil
				}
			}
			v.entries = append(v.entries, ne)
			return nil
		})
	m.EXPECT().GetEntry(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, id uuid.UUID, _ *int64) (models.Entry, error) {
			v.mu.Lock()
			defer v.mu.Unlock()
			for _, e := range v.entries {
				if e.UUID == id {
					return e, nil
				}
			}
			return models.Entry{}, sql.ErrNoRows
		})
	m.EXPECT().SetLocalMasterVerifier(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, _ int64, ver []byte) error {
			v.mu.Lock()
			defer v.mu.Unlock()
			v.verifier = append([]byte(nil), ver...)
			return nil
		})
}

func newSecretUCTest(t *testing.T, uid int64, maxBytes int64) (*gomock.Controller, *vaultMem, *SecretUseCase) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mem := &vaultMem{}
	local := NewMockLocalSecretStore(ctrl)
	mem.wireInto(local)
	sess := NewMockSessionReader(ctrl)
	sess.EXPECT().GetSession(gomock.Any()).Return(&models.Session{UserID: &uid}, nil).AnyTimes()
	uc := NewSecretUseCase(local, sess, nil, discardClientLog(), maxBytes)
	return ctrl, mem, uc
}

func TestSecretUseCase_PasswordRoundTrip(t *testing.T) {
	ctrl, mem, uc := newSecretUCTest(t, 1, DefaultMaxFileBytes)
	defer ctrl.Finish()
	ctx := context.Background()
	meta := EntryMetadata{Name: "svc", Notes: "n"}
	require.NoError(t, uc.SetPassword(ctx, meta, "secret-pw", "master!!"))
	require.Len(t, mem.entries, 1, "not saved")
	id := mem.entries[0].UUID
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
	ctrl, mem, uc := newSecretUCTest(t, 1, 1<<20)
	defer ctrl.Finish()
	ctx := context.Background()
	meta := EntryMetadata{Name: "doc", Notes: "n"}
	content := []byte("hello skeeper")
	require.NoError(t, uc.SetFile(ctx, meta, "notes.txt", content, "master!!"))
	require.Len(t, mem.entries, 1, "not saved")
	id := mem.entries[0].UUID
	assert.Equal(t, models.EntryTypeFile, mem.entries[0].Type)
	assert.GreaterOrEqual(t, len(mem.entries[0].Payload), 16, "expected ciphertext in payload")
	payload, gotMeta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, "notes.txt", orig)
	assert.Equal(t, "notes.txt", gotMeta.OriginalFilename)
	assert.Equal(t, string(content), string(payload))
	assert.Equal(t, "doc", gotMeta.Name)
}

func TestSecretUseCase_FileTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	local := NewMockLocalSecretStore(ctrl)
	sess := NewMockSessionReader(ctrl)
	uc := NewSecretUseCase(local, sess, nil, discardClientLog(), 4)
	ctx := context.Background()
	err := uc.SetFile(ctx, EntryMetadata{Name: "x"}, "a.bin", []byte("12345"), "m")
	require.Error(t, err, "expected error")
}

func TestSecretUseCase_RejectsOtherMasterPasswordAfterFirstEntry(t *testing.T) {
	ctrl, _, uc := newSecretUCTest(t, 1, DefaultMaxFileBytes)
	defer ctrl.Finish()
	ctx := context.Background()
	meta := EntryMetadata{Name: "a"}
	require.NoError(t, uc.SetPassword(ctx, meta, "pw", "master-one"))
	err := uc.SetPassword(ctx, EntryMetadata{Name: "b"}, "pw2", "master-two")
	require.Error(t, err, "expected wrong master password")
	assert.True(t, errors.Is(err, ErrWrongMasterPassword))
}

func TestSecretUseCase_UpdatePassword_BumpsVersion(t *testing.T) {
	ctrl, mem, uc := newSecretUCTest(t, 1, DefaultMaxFileBytes)
	defer ctrl.Finish()
	ctx := context.Background()
	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "n"}, "pw1", "master!!"))
	id := mem.entries[0].UUID
	assert.Equal(t, int64(1), mem.entries[0].Version)
	require.NoError(t, uc.UpdatePassword(ctx, id, EntryMetadata{Name: "n2"}, "pw2", "master!!"))
	require.Len(t, mem.entries, 1)
	assert.Equal(t, int64(2), mem.entries[0].Version)
	payload, meta, _, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, "pw2", string(payload))
	assert.Equal(t, "n2", meta.Name)
}

func TestSecretUseCase_DeleteSoft(t *testing.T) {
	ctrl, mem, uc := newSecretUCTest(t, 1, DefaultMaxFileBytes)
	defer ctrl.Finish()
	ctx := context.Background()
	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "x"}, "pw", "master!!"))
	id := mem.entries[0].UUID
	require.NoError(t, uc.DeleteEntry(ctx, id, "master!!"))
	assert.True(t, mem.entries[0].IsDeleted)
	assert.Equal(t, int64(2), mem.entries[0].Version)
	_, _, _, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	assert.True(t, errors.Is(err, ErrEntryDeleted))
}

func TestSecretUseCase_UpdateFile_MetaOnly(t *testing.T) {
	ctrl, mem, uc := newSecretUCTest(t, 1, 1<<20)
	defer ctrl.Finish()
	ctx := context.Background()
	content := []byte("payload-bytes")
	require.NoError(t, uc.SetFile(ctx, EntryMetadata{Name: "doc"}, "a.txt", content, "master!!"))
	id := mem.entries[0].UUID
	require.NoError(t, uc.UpdateFile(ctx, id, EntryMetadata{Name: "doc2"}, "master!!", false, nil, ""))
	pl, meta, orig, err := uc.GetDecryptedEntry(ctx, id, "master!!")
	require.NoError(t, err)
	assert.Equal(t, string(content), string(pl))
	assert.Equal(t, "doc2", meta.Name)
	assert.Equal(t, "a.txt", orig)
	assert.Equal(t, int64(2), mem.entries[0].Version)
}

func TestSecretUseCase_UpdateWrongEntryType(t *testing.T) {
	ctrl, mem, uc := newSecretUCTest(t, 1, DefaultMaxFileBytes)
	defer ctrl.Finish()
	ctx := context.Background()
	require.NoError(t, uc.SetPassword(ctx, EntryMetadata{Name: "p"}, "pw", "master!!"))
	id := mem.entries[0].UUID
	err := uc.UpdateText(ctx, id, EntryMetadata{Name: "t"}, "hello", "master!!")
	assert.True(t, errors.Is(err, ErrWrongEntryType))
}
