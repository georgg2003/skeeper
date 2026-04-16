package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func (r *Repository) SaveSession(ctx context.Context, s models.Session) error {
	at, err := encryptSessionToken(s.AccessToken, r.sessionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}
	rt, err := encryptSessionToken(s.RefreshToken, r.sessionKey)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}
	query := `
		INSERT INTO session (id, access_token, refresh_token, expires_at, refresh_expires_at, user_id)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at,
			refresh_expires_at = excluded.refresh_expires_at,
			user_id = excluded.user_id;
	`
	var uid any
	if s.UserID != nil {
		uid = *s.UserID
	}
	_, err = r.db.ExecContext(ctx, query, at, rt, s.ExpiresAt, s.RefreshExpiresAt, uid)
	return err
}

func (r *Repository) GetSession(ctx context.Context) (*models.Session, error) {
	query := `SELECT access_token, refresh_token, expires_at, refresh_expires_at, user_id FROM session WHERE id = 1`

	var s models.Session
	var atRaw, rtRaw string
	var refreshExp sql.NullTime
	var userID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query).Scan(
		&atRaw, &rtRaw, &s.ExpiresAt, &refreshExp, &userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if refreshExp.Valid {
		s.RefreshExpiresAt = refreshExp.Time
	}
	if userID.Valid {
		v := userID.Int64
		s.UserID = &v
	}
	s.AccessToken, err = decryptSessionToken(atRaw, r.sessionKey)
	if err != nil {
		return nil, err
	}
	s.RefreshToken, err = decryptSessionToken(rtRaw, r.sessionKey)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ClearSession(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM session")
	return err
}
