package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func runCLITest(t *testing.T, h Handlers, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runCLITestIn(t, h, strings.NewReader(stdin), args...)
}

func runCLITestIn(t *testing.T, h Handlers, stdin io.Reader, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	app := newTestApp(t, h)
	var out, errBuf bytes.Buffer
	err = app.Run(args, stdin, &out, &errBuf)
	return out.String(), errBuf.String(), err
}

func TestCLI_Register(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockHandlers)
		stdin   string
		wantSub string
		wantErr bool
	}{
		{
			name: "register_error",
			setup: func(m *MockHandlers) {
				m.EXPECT().Register(gomock.Any(), gomock.Any()).Return(context.Canceled)
			},
			stdin:   "a@b.c\npw\npw\n",
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *MockHandlers) {
				m.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Registered and logged in.")
					return nil
				})
			},
			stdin:   "a@b.c\npw\npw\n",
			wantSub: "Registered and logged in.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockHandlers(ctrl)
			tt.setup(m)
			out, _, err := runCLITest(t, m, tt.stdin, "register")
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
		setup   func(m *MockHandlers)
		wantSub string
		wantErr bool
	}{
		{
			name: "empty",
			setup: func(m *MockHandlers) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No entries.")
					return nil
				})
			},
			wantSub: "No entries.",
		},
		{
			name: "rows",
			setup: func(m *MockHandlers) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %-8s  %s%s\n", id.String(), "password", ts.Format("2006-01-02 15:04"), " (dirty)")
					return nil
				})
			},
			wantSub: id.String(),
		},
		{
			name: "error",
			setup: func(m *MockHandlers) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).Return(context.Canceled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockHandlers(ctrl)
			tt.setup(m)
			out, _, err := runCLITest(t, m, "", "list")
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
		setup   func(m *MockHandlers)
		wantSub string
		wantErr bool
	}{
		{
			name: "ok",
			setup: func(m *MockHandlers) {
				m.EXPECT().Sync(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Sync completed.")
					return nil
				})
			},
			wantSub: "Sync completed.",
		},
		{
			name: "err",
			setup: func(m *MockHandlers) {
				m.EXPECT().Sync(gomock.Any(), gomock.Any()).Return(context.Canceled)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockHandlers(ctrl)
			tt.setup(m)
			out, _, err := runCLITest(t, m, "", "sync")
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
		m := NewMockHandlers(ctrl)
		m.EXPECT().Login(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged in.")
			return nil
		})
		out, _, err := runCLITest(t, m, "u@x.y\npw\n", "login")
		require.NoError(t, err)
		assert.Contains(t, out, "Successfully logged in.")
	})
	t.Run("login_err", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := NewMockHandlers(ctrl)
		m.EXPECT().Login(gomock.Any(), gomock.Any()).Return(context.Canceled)
		_, _, err := runCLITest(t, m, "u@x.y\npw\n", "login")
		require.Error(t, err)
	})
	t.Run("logout_ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := NewMockHandlers(ctrl)
		m.EXPECT().Logout(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		})
		out, _, err := runCLITest(t, m, "", "logout")
		require.NoError(t, err)
		assert.Contains(t, out, "Logged out.")
	})
}

func TestCLI_Get(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name    string
		setup   func(m *MockHandlers)
		args    []string
		stdin   string
		wantSub string
		wantErr bool
	}{
		{
			name: "bad_uuid",
			setup: func(m *MockHandlers) {
				m.EXPECT().Get(gomock.Any(), gomock.Eq([]string{"not-a-uuid"})).Return(fmt.Errorf("invalid uuid"))
			},
			args:    []string{"get", "not-a-uuid"},
			wantErr: true,
		},
		{
			name: "password_entry",
			setup: func(m *MockHandlers) {
				m.EXPECT().Get(gomock.Any(), gomock.Eq([]string{id.String()})).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password: secret")
					return nil
				})
			},
			args:    []string{"get", id.String()},
			stdin:   "master\n",
			wantSub: "Password: secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockHandlers(ctrl)
			tt.setup(m)
			out, _, err := runCLITest(t, m, tt.stdin, tt.args...)
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
	tests := []struct {
		name    string
		setup   func(m *MockHandlers)
		stdin   string
		wantSub string
		wantErr bool
	}{
		{
			name: "success",
			setup: func(m *MockHandlers) {
				m.EXPECT().AddPassword(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted entry saved locally (run `sync` to upload).")
					return nil
				})
			},
			stdin:   "svc\nn\nsec\nmaster\n",
			wantSub: "Encrypted entry saved locally",
		},
		{
			name: "handler_error",
			setup: func(m *MockHandlers) {
				m.EXPECT().AddPassword(gomock.Any(), gomock.Any()).Return(context.Canceled)
			},
			stdin:   "svc\nn\nsec\nmaster\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockHandlers(ctrl)
			tt.setup(m)
			out, _, err := runCLITest(t, m, tt.stdin, "add", "password")
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
	ctrl := gomock.NewController(t)
	m := NewMockHandlers(ctrl)
	m.EXPECT().AddFile(gomock.Any(), gomock.Eq([]string{path})).DoAndReturn(func(cmd *cobra.Command, args []string) error {
		assert.Equal(t, path, args[0])
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted file saved locally (ciphertext in vault payload; run `sync` to upload).")
		return nil
	})
	out, _, err := runCLITest(t, m, "bin\n\nm\n", "add", "file", path)
	require.NoError(t, err)
	assert.Contains(t, out, "Encrypted file saved")
}

func TestCLI_AddCard(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockHandlers(ctrl)
	m.EXPECT().AddCard(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted card saved locally (run `sync` to upload).")
		return nil
	})
	stdin := "c\n\nH\n4111\n01/99\n999\nmaster\n"
	out, _, err := runCLITest(t, m, stdin, "add", "card")
	require.NoError(t, err)
	assert.Contains(t, out, "Encrypted card saved locally")
}

func TestCLI_AddText(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockHandlers(ctrl)
	m.EXPECT().AddText(gomock.Any(), gomock.Any()).DoAndReturn(func(cmd *cobra.Command, _ []string) error {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted text saved locally (run `sync` to upload).")
		return nil
	})
	stdin := "t\n\nhello\n\nmp\n"
	out, _, err := runCLITest(t, m, stdin, "add", "text")
	require.NoError(t, err)
	assert.Contains(t, out, "Encrypted text saved locally")
}
