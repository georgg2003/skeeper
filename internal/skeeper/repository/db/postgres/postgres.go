package postgres

import (
	"context"
	"fmt"

	"github.com/georgg2003/skeeper/internal/skeeper/repository/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func (r *repository) Close() {
	r.pool.Close()
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

func New(ctx context.Context, cfg PostgresConfig) (db.Repository, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	pool, err := pgxpool.New(ctx, connStr)

	if err != nil {
		return nil, err
	}

	return &repository{pool: pool}, err
}
