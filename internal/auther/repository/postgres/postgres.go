package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	ippostgres "github.com/georgg2003/skeeper/internal/pkg/postgres"
	"github.com/georgg2003/skeeper/pkg/errors"
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

func (r *Repository) DeleteRefreshTokenAndReturnUser(
	ctx context.Context,
	refreshTokenHash string,
) (int64, error) {
	var userID int64

	err := r.pool.QueryRow(
		ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1 AND expires_at > now() RETURNING user_id`,
		refreshTokenHash,
	).Scan(&userID)

	if errors.Is(err, pgx.ErrNoRows) {
		return -1, ErrInvalidToken
	}

	if err != nil {
		return -1, errors.Wrap(err, "failed to delete token")
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

// NewRepository returns a repository backed by an existing pool (e.g. integration tests outside this package).
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// PostgresConfig is the shared connection shape for YAML/env (see internal/pkg/postgres).
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
