// Package postgres builds a pgx connection pool from host, user, password, and the usual knobs.
package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is the connection block you’d put in YAML (host, port, user, password, database, ssl).
type Config struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// PoolConfig turns Config into something pgx can open (URL-escaped user/password/db name).
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

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	pc, err := cfg.PoolConfig()
	if err != nil {
		return nil, fmt.Errorf("postgres pool config: %w", err)
	}
	return pgxpool.NewWithConfig(ctx, pc)
}

// NewPoolFromConnString is handy for tests (e.g. a libpq string from testcontainers).
func NewPoolFromConnString(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, pc)
}
