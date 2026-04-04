package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

// SessionStore persists the authenticated session in local storage.
type SessionStore interface {
	SaveSession(ctx context.Context, s models.Session) error
	GetSession(ctx context.Context) (*models.Session, error)
	ClearSession(ctx context.Context) error
}

// RemoteAuthenticator talks to the Auther service for signup and JWT lifecycle.
type RemoteAuthenticator interface {
	CreateUser(ctx context.Context, email, password string) error
	Login(ctx context.Context, login, password string) (*models.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*models.Session, error)
}

// AuthUseCase coordinates remote authentication with the on-disk session.
type AuthUseCase struct {
	local  SessionStore
	remote RemoteAuthenticator
	l      *slog.Logger
}

// NewAuthUseCase constructs an AuthUseCase.
func NewAuthUseCase(l SessionStore, r RemoteAuthenticator, log *slog.Logger) *AuthUseCase {
	return &AuthUseCase{
		local:  l,
		remote: r,
		l:      log.With("component", "auth_usecase"),
	}
}

// Register creates a remote account and immediately establishes a local session.
func (uc *AuthUseCase) Register(ctx context.Context, email, password string) error {
	uc.l.InfoContext(ctx, "registering user", "email", email)
	if err := uc.remote.CreateUser(ctx, email, password); err != nil {
		uc.l.ErrorContext(ctx, "remote registration failed", "error", err)
		return errors.Wrap(err, "remote registration")
	}
	return uc.Login(ctx, email, password)
}

// Login authenticates against Auther and stores tokens locally.
func (uc *AuthUseCase) Login(ctx context.Context, login, password string) error {
	uc.l.InfoContext(ctx, "attempting to login", "user", login)

	session, err := uc.remote.Login(ctx, login, password)
	if err != nil {
		uc.l.ErrorContext(ctx, "failed to sign in via remote service", "error", err)
		return errors.Wrap(err, "remote sign in error")
	}

	err = uc.local.SaveSession(ctx, *session)
	if err != nil {
		uc.l.ErrorContext(ctx, "failed to save session to local db", "error", err)
		return errors.Wrap(err, "save local session error")
	}

	uc.l.InfoContext(ctx, "successfully logged in and session saved", "user", login)
	return nil
}

// GetValidToken returns a non-expired access token, refreshing when needed.
func (uc *AuthUseCase) GetValidToken(ctx context.Context) (string, error) {
	session, err := uc.local.GetSession(ctx)
	if err != nil {
		uc.l.ErrorContext(ctx, "failed to get session from local storage", "error", err)
		return "", err
	}
	if session == nil {
		return "", errors.New("no active session found, please login")
	}

	if time.Until(session.ExpiresAt) > time.Minute {
		return session.AccessToken, nil
	}

	if !session.RefreshExpiresAt.IsZero() && time.Until(session.RefreshExpiresAt) <= 0 {
		return "", errors.New("refresh token expired, please login again")
	}

	uc.l.Info("access token expired or near expiry, refreshing...")

	newSession, err := uc.remote.Refresh(ctx, session.RefreshToken)
	if err != nil {
		uc.l.ErrorContext(ctx, "failed to refresh token", "error", err)
		return "", errors.Wrap(err, "refresh token error")
	}

	if err := uc.local.SaveSession(ctx, *newSession); err != nil {
		uc.l.WarnContext(ctx, "could not save refreshed session to local db", "err", err)
	}

	uc.l.DebugContext(ctx, "token successfully refreshed")
	return newSession.AccessToken, nil
}

// Logout clears the stored session.
func (uc *AuthUseCase) Logout(ctx context.Context) error {
	uc.l.InfoContext(ctx, "logging out and clearing session")
	return uc.local.ClearSession(ctx)
}
