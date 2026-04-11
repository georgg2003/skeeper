package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
)

func runCLITest(t *testing.T, a AuthCommands, s SecretCommands, y SyncCommands, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCLITestIn(t, a, s, y, strings.NewReader(stdin), args...)
}

func runCLITestIn(t *testing.T, a AuthCommands, s SecretCommands, y SyncCommands, stdin io.Reader, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetCLIForTest(t)
	SetUseCases(a, s, y)
	full := append([]string{"--data-dir", t.TempDir()}, args...)
	var out, errBuf bytes.Buffer
	err = Run(full, stdin, &out, &errBuf)
	return out.String(), errBuf.String(), err
}

func TestCLI_Register(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(auth *MockAuthCommands)
		stdin   string
		wantSub string
		wantErr bool
	}{
		{
			name:    "password_mismatch",
			setup:   func(auth *MockAuthCommands) {},
			stdin:   "a@b.c\np1\np2\n",
			wantErr: true,
		},
		{
			name: "register_error",
			setup: func(auth *MockAuthCommands) {
				auth.EXPECT().Register(gomock.Any(), "a@b.c", "pw").Return(context.Canceled)
			},
			stdin:   "a@b.c\npw\npw\n",
			wantErr: true,
		},
		{
			name: "success",
			setup: func(auth *MockAuthCommands) {
				auth.EXPECT().Register(gomock.Any(), "a@b.c", "pw").Return(nil)
			},
			stdin:   "a@b.c\npw\npw\n",
			wantSub: "Registered and logged in.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			auth := NewMockAuthCommands(ctrl)
			tt.setup(auth)
			secret := NewMockSecretCommands(ctrl)
			sync := NewMockSyncCommands(ctrl)
			out, _, err := runCLITest(t, auth, secret, sync, tt.stdin, "register")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantSub)
		})
	}
}

func TestCLI_List(t *testing.T) {
	id := uuid.New()
	ts := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		setup   func(secret *MockSecretCommands)
		wantSub string
		wantErr bool
	}{
		{
			name: "empty",
			setup: func(secret *MockSecretCommands) {
				secret.EXPECT().ListLocal(gomock.Any()).Return([]models.Entry{}, nil)
			},
			wantSub: "No entries.",
		},
		{
			name: "rows",
			setup: func(secret *MockSecretCommands) {
				secret.EXPECT().ListLocal(gomock.Any()).Return([]models.Entry{{
					UUID: id, Type: models.EntryTypePassword, UpdatedAt: ts, IsDirty: true,
				}}, nil)
			},
			wantSub: id.String(),
		},
		{
			name: "error",
			setup: func(secret *MockSecretCommands) {
				secret.EXPECT().ListLocal(gomock.Any()).Return(nil, context.Canceled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			auth := NewMockAuthCommands(ctrl)
			secret := NewMockSecretCommands(ctrl)
			sync := NewMockSyncCommands(ctrl)
			tt.setup(secret)
			out, _, err := runCLITest(t, auth, secret, sync, "", "list")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantSub)
		})
	}
}

func TestCLI_Sync(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(sync *MockSyncCommands)
		wantSub string
		wantErr bool
	}{
		{
			name: "ok",
			setup: func(sync *MockSyncCommands) {
				sync.EXPECT().Sync(gomock.Any()).Return(nil)
			},
			wantSub: "Sync completed.",
		},
		{
			name: "err",
			setup: func(sync *MockSyncCommands) {
				sync.EXPECT().Sync(gomock.Any()).Return(context.Canceled)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			auth := NewMockAuthCommands(ctrl)
			secret := NewMockSecretCommands(ctrl)
			sync := NewMockSyncCommands(ctrl)
			tt.setup(sync)
			out, _, err := runCLITest(t, auth, secret, sync, "", "sync")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantSub)
		})
	}
}

func TestCLI_LoginLogout(t *testing.T) {
	t.Run("login_ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		auth := NewMockAuthCommands(ctrl)
		auth.EXPECT().Login(gomock.Any(), "u@x.y", "pw").Return(nil)
		secret := NewMockSecretCommands(ctrl)
		sync := NewMockSyncCommands(ctrl)
		out, _, err := runCLITest(t, auth, secret, sync, "u@x.y\npw\n", "login")
		require.NoError(t, err)
		assert.Contains(t, out, "Successfully logged in.")
	})
	t.Run("login_err", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		auth := NewMockAuthCommands(ctrl)
		auth.EXPECT().Login(gomock.Any(), "u@x.y", "pw").Return(context.Canceled)
		secret := NewMockSecretCommands(ctrl)
		sync := NewMockSyncCommands(ctrl)
		_, _, err := runCLITest(t, auth, secret, sync, "u@x.y\npw\n", "login")
		require.Error(t, err)
	})
	t.Run("logout_ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		auth := NewMockAuthCommands(ctrl)
		auth.EXPECT().Logout(gomock.Any()).Return(nil)
		secret := NewMockSecretCommands(ctrl)
		sync := NewMockSyncCommands(ctrl)
		out, _, err := runCLITest(t, auth, secret, sync, "", "logout")
		require.NoError(t, err)
		assert.Contains(t, out, "Logged out.")
	})
}

func TestCLI_Get(t *testing.T) {
	id := uuid.New()
	meta := &usecase.EntryMetadata{Name: "svc", Notes: "n"}

	tests := []struct {
		name    string
		setup   func(secret *MockSecretCommands)
		args    []string
		stdin   string
		wantSub string
		wantErr bool
	}{
		{
			name:    "bad_uuid",
			setup:   func(secret *MockSecretCommands) {},
			args:    []string{"get", "not-a-uuid"},
			wantErr: true,
		},
		{
			name: "password_entry",
			setup: func(secret *MockSecretCommands) {
				secret.EXPECT().GetLocalEntry(gomock.Any(), id).Return(models.Entry{Type: models.EntryTypePassword}, nil)
				secret.EXPECT().GetDecryptedEntry(gomock.Any(), id, "master").Return([]byte("secret"), meta, "", nil)
			},
			args:    []string{"get", id.String()},
			stdin:   "master\n",
			wantSub: "Password: secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			auth := NewMockAuthCommands(ctrl)
			secret := NewMockSecretCommands(ctrl)
			sync := NewMockSyncCommands(ctrl)
			tt.setup(secret)
			out, _, err := runCLITest(t, auth, secret, sync, tt.stdin, tt.args...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantSub)
		})
	}
}

func TestCLI_AddPassword(t *testing.T) {
	meta := usecase.EntryMetadata{Name: "svc", Notes: "n"}
	tests := []struct {
		name    string
		setup   func(secret *MockSecretCommands)
		stdin   string
		wantSub string
		wantErr bool
	}{
		{
			name: "success",
			setup: func(secret *MockSecretCommands) {
				secret.EXPECT().SetPassword(gomock.Any(), meta, "sec", "master").Return(nil)
			},
			stdin:   "svc\nn\nsec\nmaster\n",
			wantSub: "Encrypted entry saved locally",
		},
		{
			name: "usecase_error",
			setup: func(secret *MockSecretCommands) {
				secret.EXPECT().SetPassword(gomock.Any(), meta, "sec", "master").Return(context.Canceled)
			},
			stdin:   "svc\nn\nsec\nmaster\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			auth := NewMockAuthCommands(ctrl)
			secret := NewMockSecretCommands(ctrl)
			sync := NewMockSyncCommands(ctrl)
			tt.setup(secret)
			out, _, err := runCLITest(t, auth, secret, sync, tt.stdin, "add-password")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantSub)
		})
	}
}

func TestCLI_AddBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob.bin")
	require.NoError(t, os.WriteFile(path, []byte{1, 2, 3}, 0o600))
	meta := usecase.EntryMetadata{Name: "bin", Notes: ""}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetFile(gomock.Any(), meta, "blob.bin", []byte{1, 2, 3}, "m").Return(nil)
	out, _, err := runCLITest(t, NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl), "bin\n\nm\n", "add-file", path)
	require.NoError(t, err)
	assert.Contains(t, out, "Encrypted file saved")
}

func TestCLI_AddCard(t *testing.T) {
	card := models.CardPayload{Holder: "H", Number: "4111", Expiry: "01/99", CVC: "999"}
	meta := usecase.EntryMetadata{Name: "c", Notes: ""}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetCard(gomock.Any(), meta, card, "master").Return(nil)
	stdin := "c\n\nH\n4111\n01/99\n999\nmaster\n"
	out, _, err := runCLITest(t, NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl), stdin, "add-card")
	require.NoError(t, err)
	assert.Contains(t, out, "Encrypted card saved locally")
}

func TestCLI_AddText(t *testing.T) {
	meta := usecase.EntryMetadata{Name: "t", Notes: ""}
	ctrl := gomock.NewController(t)
	secret := NewMockSecretCommands(ctrl)
	secret.EXPECT().SetText(gomock.Any(), meta, "hello", "mp").Return(nil)
	// Empty line ends the text body, then master password.
	stdin := "t\n\nhello\n\nmp\n"
	out, _, err := runCLITest(t, NewMockAuthCommands(ctrl), secret, NewMockSyncCommands(ctrl), stdin, "add-text")
	require.NoError(t, err)
	assert.Contains(t, out, "Encrypted text saved locally")
}
