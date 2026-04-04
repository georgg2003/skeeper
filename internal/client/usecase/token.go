package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

type AuthProvider interface {
	Refresh(ctx context.Context, refreshToken string) (*models.Session, error)
}

type TokenUseCase struct {
	store      SessionStore
	authClient AuthProvider
	l          *slog.Logger
}

func (uc *TokenUseCase) GetValidToken(ctx context.Context) (string, error) {
	session, err := uc.store.GetSession(ctx)
	if err != nil || session == nil {
		return "", errors.Wrap(err, "user not authenticated")
	}

	if time.Until(session.ExpiresAt) > 30*time.Second {
		return session.AccessToken, nil
	}

	if !session.RefreshExpiresAt.IsZero() && time.Until(session.RefreshExpiresAt) <= 0 {
		return "", errors.New("refresh token expired, please login again")
	}

	uc.l.InfoContext(ctx, "access token expired, refreshing...")
	newSession, err := uc.authClient.Refresh(ctx, session.RefreshToken)
	if err != nil {
		return "", errors.Wrap(err, "failed to refresh session")
	}

	if err := uc.store.SaveSession(ctx, *newSession); err != nil {
		uc.l.ErrorContext(ctx, "failed to save new session", "err", err)
	}

	return newSession.AccessToken, nil
}

// NewTokenUseCase constructs a TokenUseCase.
func NewTokenUseCase(s SessionStore, a AuthProvider, log *slog.Logger) *TokenUseCase {
	return &TokenUseCase{store: s, authClient: a, l: log.With("component", "token_usecase")}
}
