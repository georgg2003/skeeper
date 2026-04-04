// Package postgres provides shared pgxpool connection configuration for service repositories.
package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds libpq-style parameters for building a pgx pool DSN.
type Config struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// PoolConfig returns a pgx pool config built via url.URL (safe escaping for user/password/database).
func (c Config) PoolConfig() (*pgxpool.Config, error) {
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

// NewPool opens a pool from Config.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	pc, err := cfg.PoolConfig()
	if err != nil {
		return nil, fmt.Errorf("postgres pool config: %w", err)
	}
	return pgxpool.NewWithConfig(ctx, pc)
}

// NewPoolFromConnString opens a pool from a libpq connection string (e.g. testcontainers output).
func NewPoolFromConnString(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, pc)
}
