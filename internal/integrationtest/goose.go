// Package integrationtest provides helpers for Postgres integration tests (Goose migrations).
package integrationtest

import (
	"context"
	"embed"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// GooseMigrate applies embedded SQL migrations from the FS root (files in ".").
func GooseMigrate(ctx context.Context, pool *pgxpool.Pool, fs embed.FS) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	goose.SetBaseFS(fs)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqlDB, ".")
}

// GooseUp calls [GooseMigrate] and fails the test on error.
func GooseUp(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fs embed.FS) {
	t.Helper()
	if err := GooseMigrate(ctx, pool, fs); err != nil {
		t.Fatal(err)
	}
}
