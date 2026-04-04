package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
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

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

func (c PostgresConfig) poolConfig() (*pgxpool.Config, error) {
	ssl := c.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.FormatUint(uint64(c.Port), 10)),
		Path:   "/" + url.PathEscape(c.Database),
	}
	q := url.Values{}
	q.Set("sslmode", ssl)
	u.RawQuery = q.Encode()
	return pgxpool.ParseConfig(u.String())
}

func NewFromPool(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func NewFromString(ctx context.Context, connStr string) (*Repository, error) {
	pc, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, err
	}
	return NewFromPool(pool), nil
}

func New(ctx context.Context, cfg PostgresConfig) (*Repository, error) {
	pc, err := cfg.poolConfig()
	if err != nil {
		return nil, fmt.Errorf("postgres pool config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, err
	}
	return NewFromPool(pool), nil
}
