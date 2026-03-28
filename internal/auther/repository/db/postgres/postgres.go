package postgres

import (
	"context"
	"fmt"

	"github.com/georgg2003/skeeper/internal/auther/pkg/config"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/db"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func (r *repository) InsertUser(ctx context.Context, creds models.DBUserCredentials) (models.UserInfo, error) {
	var userID int64
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		creds.Email, creds.PasswordHash,
	).Scan(&userID)

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return models.UserInfo{}, db.ErrUserExists
	}

	if err != nil {
		return models.UserInfo{}, errors.Wrap(err, "failed to insert user")
	}

	return models.UserInfo{
		ID:    userID,
		Email: creds.Email,
	}, nil
}

func (r *repository) DeleteRefreshTokenAndReturnUser(
	ctx context.Context,
	refreshTokenHash string,
) (int64, error) {
	var userID int64

	err := r.pool.QueryRow(
		ctx,
		`DELETE FROM tokens WHERE hash = $1 AND expires_at > now() RETURNIN user_id`,
		refreshTokenHash,
	).Scan(&userID)

	if errors.Is(err, pgx.ErrNoRows) {
		return -1, db.ErrInvalidToken
	}

	if err != nil {
		return -1, errors.Wrap(err, "failed to delete token")
	}

	return userID, nil
}

func (r *repository) InsertRefreshToken(
	ctx context.Context,
	userID int64,
	refreshToken models.RefreshTokenHashed,
) error {
	if _, err := r.pool.Exec(
		ctx, `
			INSERT INTO tokens (user_id, hash, expires_at) 
    	VALUES ($1, $2, $3)`,
		userID,
		refreshToken.Hash,
		refreshToken.Token.ExpiresAt,
	); err != nil {
		return errors.Wrap(err, "failed to insert new refresh token")
	}

	return nil
}

func (r *repository) SelectUserByEmail(ctx context.Context, email string) (info models.UserInfo, err error) {
	err = r.pool.QueryRow(
		ctx,
		"SELECT id, email, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&info.ID, &info.Email, &info.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserInfo{}, db.ErrUserNotExist
	}

	if err != nil {
		return models.UserInfo{}, errors.Wrap(err, "failed to select user")
	}

	return info, err
}

func (r *repository) Close() {
	r.pool.Close()
}

func New(ctx context.Context, cfg config.PostgresConfig) (db.Repository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	pool, err := pgxpool.New(ctx, connStr)

	if err != nil {
		return nil, err
	}

	return &repository{pool: pool}, err
}
