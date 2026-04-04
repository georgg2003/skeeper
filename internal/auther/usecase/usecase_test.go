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

	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/postgres"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"github.com/georgg2003/skeeper/pkg/utils"
)

func testJWTHelper(t *testing.T) *jwthelper.JWTHelper {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	h, err := jwthelper.New(privPEM, pubPEM, time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUseCase_CreateUser(t *testing.T) {
	ctx := context.Background()
	good := models.UserCredentials{Email: "u@example.com", Password: "valid-pass"}

	tests := []struct {
		name    string
		creds   models.UserCredentials
		setup   func(m *MockRepository)
		wantErr bool
	}{
		{
			name:    "invalid_email",
			creds:   models.UserCredentials{Email: "nope", Password: "x"},
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
						if db.Email != good.Email || len(db.PasswordHash) == 0 {
							t.Fatalf("bad insert payload %+v", db)
						}
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
			uc := New(discardLogger(), repo, testJWTHelper(t))
			info, err := uc.CreateUser(ctx, tt.creds)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || info.ID != 3 {
				t.Fatalf("got %+v %v", info, err)
			}
		})
	}
}

func TestUseCase_LoginUser(t *testing.T) {
	ctx := context.Background()
	jh := testJWTHelper(t)
	email := "login@example.com"
	pass := "correct-password"
	hash, err := (&models.UserCredentials{Email: email, Password: pass}).HashPassword()
	if err != nil {
		t.Fatal(err)
	}
	userRow := models.UserInfo{ID: 77, Email: email, PasswordHash: hash}

	tests := []struct {
		name    string
		creds   models.UserCredentials
		setup   func(m *MockRepository)
		wantErr bool
	}{
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
			name:  "insert_refresh_fails",
			creds: models.UserCredentials{Email: email, Password: pass},
			setup: func(m *MockRepository) {
				m.EXPECT().SelectUserByEmail(gomock.Any(), email).Return(userRow, nil)
				m.EXPECT().InsertRefreshToken(gomock.Any(), userRow.ID, gomock.Any()).Return(errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:  "success",
			creds: models.UserCredentials{Email: email, Password: pass},
			setup: func(m *MockRepository) {
				m.EXPECT().SelectUserByEmail(gomock.Any(), email).Return(userRow, nil)
				m.EXPECT().InsertRefreshToken(gomock.Any(), userRow.ID, gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockRepository(ctrl)
			tt.setup(repo)
			uc := New(discardLogger(), repo, jh)
			out, err := uc.LoginUser(ctx, tt.creds)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || out.User.ID != userRow.ID || out.TokenPair.AccessToken.Token == "" {
				t.Fatalf("%+v %v", out, err)
			}
		})
	}
}

func TestUseCase_RotateToken(t *testing.T) {
	ctx := context.Background()
	jh := testJWTHelper(t)
	rawRefresh := "opaque-refresh-token"
	hash := utils.HashToken(rawRefresh)
	userID := int64(100)

	tests := []struct {
		name    string
		setup   func(m *MockRepository)
		wantErr bool
	}{
		{
			name: "invalid_token",
			setup: func(m *MockRepository) {
				m.EXPECT().DeleteRefreshTokenAndReturnUser(gomock.Any(), gomock.Any()).
					Return(int64(0), postgres.ErrInvalidToken)
			},
			wantErr: true,
		},
		{
			name: "insert_new_refresh_fails",
			setup: func(m *MockRepository) {
				m.EXPECT().DeleteRefreshTokenAndReturnUser(gomock.Any(), hash).Return(userID, nil)
				m.EXPECT().InsertRefreshToken(gomock.Any(), userID, gomock.Any()).Return(errors.New("db"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *MockRepository) {
				m.EXPECT().DeleteRefreshTokenAndReturnUser(gomock.Any(), hash).Return(userID, nil)
				m.EXPECT().InsertRefreshToken(gomock.Any(), userID, gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockRepository(ctrl)
			tt.setup(repo)
			uc := New(discardLogger(), repo, jh)
			pair, err := uc.RotateToken(ctx, rawRefresh)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || pair.AccessToken.Token == "" || pair.RefreshToken.Token == "" {
				t.Fatalf("%+v %v", pair, err)
			}
			if pair.RefreshToken.Token == rawRefresh {
				t.Fatal("expected new refresh token")
			}
		})
	}
}
