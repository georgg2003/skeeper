package usecase

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	stderrors "errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/postgres"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

func testJWTHelper(t *testing.T) *jwthelper.JWTHelper {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	h, err := jwthelper.New(privPEM, pubPEM, time.Minute, 24*time.Hour, "")
	require.NoError(t, err)
	return h
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUseCase_CreateUser(t *testing.T) {
	ctx := context.Background()
	good := models.UserCredentials{Email: "u@example.com", Password: "valid-pass-12chars"}

	tests := []struct {
		name    string
		creds   models.UserCredentials
		setup   func(m *MockRepository)
		wantErr bool
	}{
		{
			name:    "invalid_email",
			creds:   models.UserCredentials{Email: "nope", Password: "long-password-ok"},
			setup:   func(m *MockRepository) {},
			wantErr: true,
		},
		{
			name:    "empty_password",
			creds:   models.UserCredentials{Email: "a@b.c", Password: ""},
			setup:   func(m *MockRepository) {},
			wantErr: true,
		},
		{
			name:  "user_exists",
			creds: good,
			setup: func(m *MockRepository) {
				m.EXPECT().InsertUser(gomock.Any(), gomock.AssignableToTypeOf(models.DBUserCredentials{})).
					Return(models.UserInfo{}, postgres.ErrUserExists)
			},
			wantErr: true,
		},
		{
			name:  "repo_error",
			creds: good,
			setup: func(m *MockRepository) {
				m.EXPECT().InsertUser(gomock.Any(), gomock.AssignableToTypeOf(models.DBUserCredentials{})).
					Return(models.UserInfo{}, errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:  "success",
			creds: good,
			setup: func(m *MockRepository) {
				m.EXPECT().InsertUser(gomock.Any(), gomock.AssignableToTypeOf(models.DBUserCredentials{})).
					DoAndReturn(func(_ context.Context, db models.DBUserCredentials) (models.UserInfo, error) {
						require.Equal(t, good.Email, db.Email)
						require.NotEmpty(t, db.PasswordHash)
						return models.UserInfo{ID: 3, Email: good.Email}, nil
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockRepository(ctrl)
			tt.setup(repo)
			uc, err := New(discardLogger(), repo, testJWTHelper(t))
			require.NoError(t, err)
			info, err := uc.CreateUser(ctx, tt.creds)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				if tt.name == "user_exists" {
					assert.True(t, stderrors.Is(err, ErrUserExists))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(3), info.ID)
		})
	}
}

func TestUseCase_LoginUser(t *testing.T) {
	ctx := context.Background()
	jh := testJWTHelper(t)
	email := "login@example.com"
	pass := "correct-password"
	hash, err := (&models.UserCredentials{Email: email, Password: pass}).HashPassword()
	require.NoError(t, err)
	userRow := models.UserInfo{ID: 77, Email: email, PasswordHash: hash}

	tests := []struct {
		name    string
		creds   models.UserCredentials
		setup   func(m *MockRepository)
		wantErr bool
	}{
		{
			name:    "invalid_email",
			creds:   models.UserCredentials{Email: "not-email", Password: "x"},
			setup:   func(m *MockRepository) {},
			wantErr: true,
		},
		{
			name:    "empty_password",
			creds:   models.UserCredentials{Email: "a@b.c", Password: ""},
			setup:   func(m *MockRepository) {},
			wantErr: true,
		},
		{
			name:  "user_not_found",
			creds: models.UserCredentials{Email: "missing@example.com", Password: "x"},
			setup: func(m *MockRepository) {
				m.EXPECT().SelectUserByEmail(gomock.Any(), "missing@example.com").
					Return(models.UserInfo{}, postgres.ErrUserNotExist)
			},
			wantErr: true,
		},
		{
			name:  "wrong_password",
			creds: models.UserCredentials{Email: email, Password: "wrong"},
			setup: func(m *MockRepository) {
				m.EXPECT().SelectUserByEmail(gomock.Any(), email).Return(userRow, nil)
			},
			wantErr: true,
		},
		{
			name:  "replace_refresh_fails",
			creds: models.UserCredentials{Email: email, Password: pass},
			setup: func(m *MockRepository) {
				m.EXPECT().SelectUserByEmail(gomock.Any(), email).Return(userRow, nil)
				m.EXPECT().ReplaceUserRefreshTokens(gomock.Any(), userRow.ID, gomock.Any()).Return(errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:  "success",
			creds: models.UserCredentials{Email: email, Password: pass},
			setup: func(m *MockRepository) {
				m.EXPECT().SelectUserByEmail(gomock.Any(), email).Return(userRow, nil)
				m.EXPECT().ReplaceUserRefreshTokens(gomock.Any(), userRow.ID, gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockRepository(ctrl)
			tt.setup(repo)
			uc, err := New(discardLogger(), repo, jh)
			require.NoError(t, err)
			out, err := uc.LoginUser(ctx, tt.creds)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, userRow.ID, out.User.ID)
			assert.NotEmpty(t, out.TokenPair.AccessToken.Token)
		})
	}
}

func TestUseCase_RotateToken(t *testing.T) {
	ctx := context.Background()
	jh := testJWTHelper(t)
	rawRefresh := "opaque-refresh-token"
	userID := int64(100)

	tests := []struct {
		name    string
		setup   func(m *MockRepository)
		wantErr bool
	}{
		{
			name: "invalid_token",
			setup: func(m *MockRepository) {
				m.EXPECT().RotateRefreshToken(gomock.Any(), rawRefresh, gomock.Any()).
					Return(jwthelper.TokenPair{}, postgres.ErrInvalidToken)
			},
			wantErr: true,
		},
		{
			name: "rotate_db_error",
			setup: func(m *MockRepository) {
				m.EXPECT().RotateRefreshToken(gomock.Any(), rawRefresh, gomock.Any()).
					Return(jwthelper.TokenPair{}, errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "mint_fails_inside_rotate",
			setup: func(m *MockRepository) {
				m.EXPECT().RotateRefreshToken(gomock.Any(), rawRefresh, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ func(int64) (jwthelper.TokenPair, error)) (jwthelper.TokenPair, error) {
						return jwthelper.TokenPair{}, errors.New("mint failed")
					})
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *MockRepository) {
				m.EXPECT().RotateRefreshToken(gomock.Any(), rawRefresh, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, mint func(int64) (jwthelper.TokenPair, error)) (jwthelper.TokenPair, error) {
						return mint(userID)
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockRepository(ctrl)
			tt.setup(repo)
			uc, err := New(discardLogger(), repo, jh)
			require.NoError(t, err)
			pair, err := uc.RotateToken(ctx, rawRefresh)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, pair.AccessToken.Token)
			assert.NotEmpty(t, pair.RefreshToken.Token)
			assert.NotEqual(t, rawRefresh, pair.RefreshToken.Token, "expected new refresh token")
		})
	}
}

// singleFlightRepoStub counts RotateRefreshToken calls for TestUseCase_RotateToken_SingleFlight.
// Other repository methods panic so the test cannot accidentally use them.
type singleFlightRepoStub struct {
	rotateCalls atomic.Int64
}

func (s *singleFlightRepoStub) InsertUser(context.Context, models.DBUserCredentials) (models.UserInfo, error) {
	panic("unexpected")
}

func (s *singleFlightRepoStub) ReplaceUserRefreshTokens(context.Context, int64, jwthelper.TokenPair) error {
	panic("unexpected")
}

func (s *singleFlightRepoStub) RotateRefreshToken(_ context.Context, _ string, mint func(int64) (jwthelper.TokenPair, error)) (jwthelper.TokenPair, error) {
	s.rotateCalls.Add(1)
	time.Sleep(25 * time.Millisecond)
	return mint(42)
}

func (s *singleFlightRepoStub) SelectUserByEmail(context.Context, string) (models.UserInfo, error) {
	panic("unexpected")
}

func (*singleFlightRepoStub) Close() {}

func TestUseCase_RotateToken_SingleFlight(t *testing.T) {
	ctx := context.Background()
	jh := testJWTHelper(t)
	rawRefresh := "shared-refresh-token"
	repo := &singleFlightRepoStub{}
	uc, err := New(discardLogger(), repo, jh)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)
	var err0, err1 error
	go func() {
		defer wg.Done()
		_, err0 = uc.RotateToken(ctx, rawRefresh)
	}()
	go func() {
		defer wg.Done()
		_, err1 = uc.RotateToken(ctx, rawRefresh)
	}()
	wg.Wait()
	require.NoError(t, err0)
	require.NoError(t, err1)
	assert.Equal(t, int64(1), repo.rotateCalls.Load(), "RotateRefreshToken calls")
}
