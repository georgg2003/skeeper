package usecase

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

type memTokenSession struct {
	s       models.Session
	saveErr error
}

func (m *memTokenSession) GetSession(context.Context) (*models.Session, error) {
	return &m.s, nil
}

func (m *memTokenSession) SaveSession(context.Context, models.Session) error {
	return m.saveErr
}

func (m *memTokenSession) ClearSession(context.Context) error {
	return nil
}

type stubRefresh struct {
	out *models.Session
	err error
}

func (s stubRefresh) Refresh(context.Context, string) (*models.Session, error) {
	return s.out, s.err
}

func TestTokenUseCase_GetValidToken_ReturnsErrorWhenSaveFails(t *testing.T) {
	ctx := context.Background()
	oldRT := "refresh-old"
	store := &memTokenSession{
		saveErr: errors.New("disk full"),
		s: models.Session{
			AccessToken:      "expired-access",
			RefreshToken:     oldRT,
			ExpiresAt:        time.Now().Add(-time.Hour),
			RefreshExpiresAt: time.Now().Add(time.Hour),
		},
	}
	newSess := models.Session{
		AccessToken:      "new-access",
		RefreshToken:     "new-refresh",
		ExpiresAt:        time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	}
	uc := NewTokenUseCase(store, stubRefresh{out: &newSess}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := uc.GetValidToken(ctx)
	require.Error(t, err, "expected error when SaveSession fails after refresh")
}
