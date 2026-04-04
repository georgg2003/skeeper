package db

import (
	"context"
	"database/sql"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func (r *Repository) SaveSession(ctx context.Context, s models.Session) error {
	query := `
		INSERT INTO session (id, access_token, refresh_token, expires_at, refresh_expires_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at,
			refresh_expires_at = excluded.refresh_expires_at;
	`
	_, err := r.db.ExecContext(ctx, query, s.AccessToken, s.RefreshToken, s.ExpiresAt, s.RefreshExpiresAt)
	return err
}

func (r *Repository) GetSession(ctx context.Context) (*models.Session, error) {
	query := `SELECT access_token, refresh_token, expires_at, refresh_expires_at FROM session WHERE id = 1`

	var s models.Session
	var refreshExp sql.NullTime
	err := r.db.QueryRowContext(ctx, query).Scan(
		&s.AccessToken, &s.RefreshToken, &s.ExpiresAt, &refreshExp,
	)
	if refreshExp.Valid {
		s.RefreshExpiresAt = refreshExp.Time
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ClearSession(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM session")
	return err
}
