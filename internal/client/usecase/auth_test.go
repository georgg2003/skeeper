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

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

type memSession struct {
	s       *models.Session
	saveErr error
}

func (m *memSession) SaveSession(ctx context.Context, s models.Session) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	cp := s
	m.s = &cp
	return nil
}

func (m *memSession) GetSession(ctx context.Context) (*models.Session, error) {
	return m.s, nil
}

func (m *memSession) ClearSession(ctx context.Context) error {
	m.s = nil
	return nil
}

type stubRemote struct {
	createErr  error
	loginOut   *models.Session
	loginErr   error
	refreshOut *models.Session
	refreshErr error
}

func (s *stubRemote) CreateUser(ctx context.Context, email, password string) error {
	return s.createErr
}

func (s *stubRemote) Login(ctx context.Context, login, password string) (*models.Session, error) {
	if s.loginErr != nil {
		return nil, s.loginErr
	}
	return s.loginOut, nil
}

func (s *stubRemote) Refresh(ctx context.Context, refreshToken string) (*models.Session, error) {
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}
	return s.refreshOut, nil
}

func clientTestJWTHelper(t *testing.T, userID int64) (access string, refresh string) {
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
	pair, err := h.NewTokenPair(userID)
	if err != nil {
		t.Fatal(err)
	}
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
		local   *memSession
		remote  *stubRemote
		wantErr bool
	}{
		{
			name:    "create_user_fails",
			local:   &memSession{},
			remote:  &stubRemote{createErr: context.Canceled},
			wantErr: true,
		},
		{
			name:   "success",
			local:  &memSession{},
			remote: &stubRemote{loginOut: sess},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAuthUseCase(tt.local, tt.remote, discardClientLog())
			err := uc.Register(ctx, "a@b.c", "password")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.local.s == nil || tt.local.s.AccessToken != at {
				t.Fatal("session not saved")
			}
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
		local   *memSession
		remote  *stubRemote
		wantErr bool
	}{
		{
			name:    "remote_error",
			local:   &memSession{},
			remote:  &stubRemote{loginErr: context.Canceled},
			wantErr: true,
		},
		{
			name:   "success",
			local:  &memSession{},
			remote: &stubRemote{loginOut: sess},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAuthUseCase(tt.local, tt.remote, discardClientLog())
			err := uc.Login(ctx, "u@x.y", "pw")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.local.s == nil || tt.local.s.UserID == nil || *tt.local.s.UserID != uid {
				t.Fatalf("session %+v", tt.local.s)
			}
		})
	}
}

func TestAuthUseCase_GetValidToken(t *testing.T) {
	ctx := context.Background()
	uid := int64(3)
	at, rt := clientTestJWTHelper(t, uid)

	tests := []struct {
		name    string
		local   *memSession
		remote  *stubRemote
		wantErr bool
	}{
		{
			name:    "no_session",
			local:   &memSession{s: nil},
			remote:  &stubRemote{},
			wantErr: true,
		},
		{
			name: "valid_access_returns_token",
			local: &memSession{s: &models.Session{
				AccessToken:      at,
				RefreshToken:     rt,
				ExpiresAt:        time.Now().Add(10 * time.Minute),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				UserID:           &uid,
			}},
			remote: &stubRemote{},
		},
		{
			name: "refresh_path",
			local: &memSession{s: &models.Session{
				AccessToken:      "old",
				RefreshToken:     rt,
				ExpiresAt:        time.Now().Add(30 * time.Second),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				UserID:           &uid,
			}},
			remote: &stubRemote{
				refreshOut: func() *models.Session {
					at2, rt2 := clientTestJWTHelper(t, uid)
					return &models.Session{
						AccessToken:      at2,
						RefreshToken:     rt2,
						ExpiresAt:        time.Now().Add(time.Hour),
						RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					}
				}(),
			},
		},
		{
			name: "refresh_token_expired",
			local: &memSession{s: &models.Session{
				AccessToken:      "x",
				RefreshToken:     rt,
				ExpiresAt:        time.Now().Add(-time.Hour),
				RefreshExpiresAt: time.Now().Add(-time.Minute),
				UserID:           &uid,
			}},
			remote:  &stubRemote{},
			wantErr: true,
		},
		{
			name: "refresh_remote_error",
			local: &memSession{s: &models.Session{
				AccessToken:      "x",
				RefreshToken:     rt,
				ExpiresAt:        time.Now().Add(30 * time.Second),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				UserID:           &uid,
			}},
			remote:  &stubRemote{refreshErr: context.Canceled},
			wantErr: true,
		},
		{
			name: "refresh_save_session_error",
			local: &memSession{
				saveErr: context.Canceled,
				s: &models.Session{
					AccessToken:      "old",
					RefreshToken:     rt,
					ExpiresAt:        time.Now().Add(30 * time.Second),
					RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					UserID:           &uid,
				},
			},
			remote: &stubRemote{
				refreshOut: func() *models.Session {
					at2, rt2 := clientTestJWTHelper(t, uid)
					return &models.Session{
						AccessToken:      at2,
						RefreshToken:     rt2,
						ExpiresAt:        time.Now().Add(time.Hour),
						RefreshExpiresAt: time.Now().Add(24 * time.Hour),
					}
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAuthUseCase(tt.local, tt.remote, discardClientLog())
			tok, err := uc.GetValidToken(ctx)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || tok == "" {
				t.Fatalf("token %q err %v", tok, err)
			}
		})
	}
}

func TestAuthUseCase_Logout(t *testing.T) {
	ctx := context.Background()
	local := &memSession{s: &models.Session{AccessToken: "x"}}
	uc := NewAuthUseCase(local, &stubRemote{}, discardClientLog())
	if err := uc.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	if local.s != nil {
		t.Fatal("expected cleared session")
	}
}

// Ensures jwtuser can read claims from tokens produced by jwthelper (regression for Login/Register).
func TestJWTUserClaimRoundTrip(t *testing.T) {
	uid := int64(42)
	at, _ := clientTestJWTHelper(t, uid)
	p := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err := p.ParseUnverified(at, claims)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := claims["UserID"].(float64); !ok || int64(v) != uid {
		t.Fatalf("claims %+v", claims)
	}
}
