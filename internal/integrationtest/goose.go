// Package integrationtest provides helpers for Postgres integration tests (Goose migrations).
package integrationtest

import (
	"context"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// GooseMigrate applies embedded SQL migrations from the FS root (files in ".").
func GooseMigrate(ctx context.Context, pool *pgxpool.Pool, fs embed.FS) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	goose.SetBaseFS(fs)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqlDB, ".")
}
