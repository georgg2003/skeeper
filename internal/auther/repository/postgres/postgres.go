// Package postgres persists Auther users and hashed refresh tokens.
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	ippostgres "github.com/georgg2003/skeeper/internal/pkg/postgres"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"github.com/georgg2003/skeeper/pkg/utils"
)

var ErrUserExists = errors.New("user already exists")
var ErrInvalidToken = errors.New("invalid token")
var ErrUserNotExist = errors.New("user does not exist")

type Repository struct {
	pool *pgxpool.Pool
}

func (r *Repository) InsertUser(ctx context.Context, creds models.DBUserCredentials) (models.UserInfo, error) {
	var userID int64
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		creds.Email, creds.PasswordHash,
	).Scan(&userID)

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return models.UserInfo{}, ErrUserExists
	}

	if err != nil {
		return models.UserInfo{}, errors.Wrap(err, "failed to insert user")
	}

	return models.UserInfo{
		ID:           userID,
		Email:        creds.Email,
		PasswordHash: creds.PasswordHash,
	}, nil
}

func (r *Repository) DeleteRefreshTokensForUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

// ReplaceUserRefreshTokens revokes all refresh tokens for the user and stores a new one in a single transaction.
func (r *Repository) ReplaceUserRefreshTokens(ctx context.Context, userID int64, pair jwthelper.TokenPair) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "begin replace refresh tx")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return errors.Wrap(err, "delete old refresh tokens")
	}

	rt := models.RefreshTokenHashed{
		Token: pair.RefreshToken,
		Hash:  utils.HashToken(pair.RefreshToken.Token),
	}
	if _, err := tx.Exec(ctx, `
			INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)`,
		userID,
		rt.Hash,
		rt.ExpiresAt,
	); err != nil {
		return errors.Wrap(err, "insert refresh token")
	}

	return tx.Commit(ctx)
}

// RotateRefreshToken consumes a valid refresh token, mints a new pair via mint, and inserts the new refresh hash in one transaction.
func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	refreshPlain string,
	mint func(userID int64) (jwthelper.TokenPair, error),
) (jwthelper.TokenPair, error) {
	hash := utils.HashToken(refreshPlain)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return jwthelper.TokenPair{}, errors.Wrap(err, "begin rotate refresh tx")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	var userID int64
	var expiresAt time.Time
	var usedAt sql.NullTime
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, expires_at, used_at
		FROM refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE
	`, hash).Scan(&id, &userID, &expiresAt, &usedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return jwthelper.TokenPair{}, ErrInvalidToken
	}
	if err != nil {
		return jwthelper.TokenPair{}, errors.Wrap(err, "select refresh token")
	}

	if usedAt.Valid {
		if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
			return jwthelper.TokenPair{}, errors.Wrap(err, "revoke refresh tokens after reuse")
		}
		if err := tx.Commit(ctx); err != nil {
			return jwthelper.TokenPair{}, err
		}
		return jwthelper.TokenPair{}, ErrInvalidToken
	}

	if !expiresAt.After(time.Now()) {
		return jwthelper.TokenPair{}, ErrInvalidToken
	}

	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET used_at = NOW() WHERE id = $1`, id); err != nil {
		return jwthelper.TokenPair{}, errors.Wrap(err, "mark refresh token used")
	}

	pair, err := mint(userID)
	if err != nil {
		return jwthelper.TokenPair{}, err
	}

	rt := models.RefreshTokenHashed{
		Token: pair.RefreshToken,
		Hash:  utils.HashToken(pair.RefreshToken.Token),
	}
	if _, err := tx.Exec(ctx, `
			INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)`,
		userID,
		rt.Hash,
		rt.ExpiresAt,
	); err != nil {
		return jwthelper.TokenPair{}, errors.Wrap(err, "insert rotated refresh token")
	}

	if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1 AND used_at IS NOT NULL`, userID); err != nil {
		return jwthelper.TokenPair{}, errors.Wrap(err, "prune consumed refresh tokens")
	}

	if err := tx.Commit(ctx); err != nil {
		return jwthelper.TokenPair{}, err
	}
	return pair, nil
}

func (r *Repository) DeleteRefreshTokenAndReturnUser(
	ctx context.Context,
	refreshTokenHash string,
) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return -1, errors.Wrap(err, "begin refresh token tx")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	var userID int64
	var expiresAt time.Time
	var usedAt sql.NullTime
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, expires_at, used_at
		FROM refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE
	`, refreshTokenHash).Scan(&id, &userID, &expiresAt, &usedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return -1, ErrInvalidToken
	}
	if err != nil {
		return -1, errors.Wrap(err, "failed to select refresh token")
	}

	if usedAt.Valid {
		if _, err := tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
			return -1, errors.Wrap(err, "revoke refresh tokens after reuse")
		}
		if err := tx.Commit(ctx); err != nil {
			return -1, err
		}
		return -1, ErrInvalidToken
	}

	if !expiresAt.After(time.Now()) {
		return -1, ErrInvalidToken
	}

	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET used_at = NOW() WHERE id = $1`, id); err != nil {
		return -1, errors.Wrap(err, "failed to mark refresh token used")
	}
	if err := tx.Commit(ctx); err != nil {
		return -1, err
	}
	return userID, nil
}

func (r *Repository) InsertRefreshToken(
	ctx context.Context,
	userID int64,
	refreshToken models.RefreshTokenHashed,
) error {
	if _, err := r.pool.Exec(
		ctx, `
			INSERT INTO refresh_tokens (user_id, token_hash, expires_at) 
    	VALUES ($1, $2, $3)`,
		userID,
		refreshToken.Hash,
		refreshToken.ExpiresAt,
	); err != nil {
		return errors.Wrap(err, "failed to insert new refresh token")
	}

	return nil
}

func (r *Repository) SelectUserByEmail(ctx context.Context, email string) (info models.UserInfo, err error) {
	err = r.pool.QueryRow(
		ctx,
		"SELECT id, email, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&info.ID, &info.Email, &info.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserInfo{}, ErrUserNotExist
	}

	if err != nil {
		return models.UserInfo{}, errors.Wrap(err, "failed to select user")
	}

	return info, err
}

func (r *Repository) Close() {
	r.pool.Close()
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type PostgresConfig = ippostgres.Config

func NewFromPool(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func NewFromString(ctx context.Context, connStr string) (*Repository, error) {
	pool, err := ippostgres.NewPoolFromConnString(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return NewFromPool(pool), nil
}

func New(ctx context.Context, cfg PostgresConfig) (*Repository, error) {
	pool, err := ippostgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return NewFromPool(pool), nil
}
