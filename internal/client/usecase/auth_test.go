package usecase

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

func clientTestJWTHelper(t *testing.T, userID int64) (access string, refresh string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	h, err := jwthelper.New(privPEM, pubPEM, time.Minute, 24*time.Hour, "")
	require.NoError(t, err)
	pair, err := h.NewTokenPair(userID)
	require.NoError(t, err)
	return pair.AccessToken.Token, pair.RefreshToken.Token
}

func discardClientLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAuthUseCase_Register(t *testing.T) {
	ctx := context.Background()
	uid := int64(55)
	at, rt := clientTestJWTHelper(t, uid)
	sess := &models.Session{
		AccessToken:      at,
		RefreshToken:     rt,
		ExpiresAt:        time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name    string
		setup   func(local *MockSessionStore, remote *MockRemoteAuthenticator)
		wantErr bool
	}{
		{
			name: "create_user_fails",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				remote.EXPECT().CreateUser(gomock.Any(), "a@b.c", "password123456").Return(context.Canceled)
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				gomock.InOrder(
					remote.EXPECT().CreateUser(gomock.Any(), "a@b.c", "password123456").Return(nil),
					remote.EXPECT().Login(gomock.Any(), "a@b.c", "password123456").Return(sess, nil),
					local.EXPECT().SaveSession(gomock.Any(), gomock.AssignableToTypeOf(models.Session{})).DoAndReturn(
						func(_ context.Context, got models.Session) error {
							assert.Equal(t, at, got.AccessToken)
							assert.NotNil(t, got.UserID)
							assert.Equal(t, uid, *got.UserID)
							return nil
						}),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			local := NewMockSessionStore(ctrl)
			remote := NewMockRemoteAuthenticator(ctrl)
			tt.setup(local, remote)
			uc := NewAuthUseCase(local, remote, discardClientLog())
			err := uc.Register(ctx, "a@b.c", "password123456")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthUseCase_Login(t *testing.T) {
	ctx := context.Background()
	uid := int64(9)
	at, rt := clientTestJWTHelper(t, uid)
	sess := &models.Session{
		AccessToken:      at,
		RefreshToken:     rt,
		ExpiresAt:        time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name    string
		setup   func(local *MockSessionStore, remote *MockRemoteAuthenticator)
		wantErr bool
	}{
		{
			name: "remote_error",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				remote.EXPECT().Login(gomock.Any(), "u@x.y", "pw").Return(nil, context.Canceled)
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				gomock.InOrder(
					remote.EXPECT().Login(gomock.Any(), "u@x.y", "pw").Return(sess, nil),
					local.EXPECT().SaveSession(gomock.Any(), gomock.AssignableToTypeOf(models.Session{})).DoAndReturn(
						func(_ context.Context, got models.Session) error {
							assert.Equal(t, uid, *got.UserID)
							return nil
						}),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			local := NewMockSessionStore(ctrl)
			remote := NewMockRemoteAuthenticator(ctrl)
			tt.setup(local, remote)
			uc := NewAuthUseCase(local, remote, discardClientLog())
			err := uc.Login(ctx, "u@x.y", "pw")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthUseCase_GetValidToken(t *testing.T) {
	ctx := context.Background()
	uid := int64(3)
	at, rt := clientTestJWTHelper(t, uid)

	tests := []struct {
		name    string
		setup   func(local *MockSessionStore, remote *MockRemoteAuthenticator)
		wantErr bool
	}{
		{
			name: "no_session",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				local.EXPECT().GetSession(gomock.Any()).Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name: "valid_access_returns_token",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				local.EXPECT().GetSession(gomock.Any()).Return(&models.Session{
					AccessToken:      at,
					RefreshToken:     rt,
					ExpiresAt:        time.Now().Add(10 * time.Minute),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					UserID:           &uid,
				}, nil)
			},
		},
		{
			name: "refresh_path",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				at2, rt2 := clientTestJWTHelper(t, uid)
				refreshed := &models.Session{
					AccessToken:      at2,
					RefreshToken:     rt2,
					ExpiresAt:        time.Now().Add(time.Hour),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				}
				gomock.InOrder(
					local.EXPECT().GetSession(gomock.Any()).Return(&models.Session{
						AccessToken:      "old",
						RefreshToken:     rt,
						ExpiresAt:        time.Now().Add(30 * time.Second),
						RefreshExpiresAt: time.Now().Add(24 * time.Hour),
						UserID:           &uid,
					}, nil),
					remote.EXPECT().Refresh(gomock.Any(), rt).Return(refreshed, nil),
					local.EXPECT().SaveSession(gomock.Any(), gomock.AssignableToTypeOf(models.Session{})).DoAndReturn(
						func(_ context.Context, got models.Session) error {
							assert.Equal(t, at2, got.AccessToken)
							assert.NotNil(t, got.UserID)
							assert.Equal(t, uid, *got.UserID)
							return nil
						}),
				)
			},
		},
		{
			name: "refresh_token_expired",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				local.EXPECT().GetSession(gomock.Any()).Return(&models.Session{
					AccessToken:      "x",
					RefreshToken:     rt,
					ExpiresAt:        time.Now().Add(-time.Hour),
					RefreshExpiresAt: time.Now().Add(-time.Minute),
					UserID:           &uid,
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "refresh_remote_error",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				gomock.InOrder(
					local.EXPECT().GetSession(gomock.Any()).Return(&models.Session{
						AccessToken:      "x",
						RefreshToken:     rt,
						ExpiresAt:        time.Now().Add(30 * time.Second),
						RefreshExpiresAt: time.Now().Add(24 * time.Hour),
						UserID:           &uid,
					}, nil),
					remote.EXPECT().Refresh(gomock.Any(), rt).Return(nil, context.Canceled),
				)
			},
			wantErr: true,
		},
		{
			name: "refresh_save_session_error",
			setup: func(local *MockSessionStore, remote *MockRemoteAuthenticator) {
				at2, rt2 := clientTestJWTHelper(t, uid)
				refreshed := &models.Session{
					AccessToken:      at2,
					RefreshToken:     rt2,
					ExpiresAt:        time.Now().Add(time.Hour),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				}
				gomock.InOrder(
					local.EXPECT().GetSession(gomock.Any()).Return(&models.Session{
						AccessToken:      "old",
						RefreshToken:     rt,
						ExpiresAt:        time.Now().Add(30 * time.Second),
						RefreshExpiresAt: time.Now().Add(24 * time.Hour),
						UserID:           &uid,
					}, nil),
					remote.EXPECT().Refresh(gomock.Any(), rt).Return(refreshed, nil),
					local.EXPECT().SaveSession(gomock.Any(), gomock.AssignableToTypeOf(models.Session{})).Return(context.Canceled),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			local := NewMockSessionStore(ctrl)
			remote := NewMockRemoteAuthenticator(ctrl)
			tt.setup(local, remote)
			uc := NewAuthUseCase(local, remote, discardClientLog())
			tok, err := uc.GetValidToken(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, tok)
		})
	}
}

func TestAuthUseCase_Logout(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	local := NewMockSessionStore(ctrl)
	local.EXPECT().ClearSession(gomock.Any()).Return(nil)
	uc := NewAuthUseCase(local, NewMockRemoteAuthenticator(ctrl), discardClientLog())
	require.NoError(t, uc.Logout(ctx))
}

// Ensures jwtuser can read claims from tokens produced by jwthelper (regression for Login/Register).
func TestJWTUserClaimRoundTrip(t *testing.T) {
	uid := int64(42)
	at, _ := clientTestJWTHelper(t, uid)
	p := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err := p.ParseUnverified(at, claims)
	require.NoError(t, err)
	v, ok := claims["UserID"].(float64)
	require.True(t, ok)
	assert.Equal(t, uid, int64(v))
}
