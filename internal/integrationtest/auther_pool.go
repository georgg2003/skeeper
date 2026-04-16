package integrationtest

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	authermigrate "github.com/georgg2003/skeeper/migrations/auther"
)

var (
	autherPoolOnce sync.Once
	autherPool     *pgxpool.Pool
	autherPoolErr  error
)

// AutherTestPool returns a migrated Postgres pool for Auther repository integration tests
// (Postgres via testcontainers-go; Docker must be available).
func AutherTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	autherPoolOnce.Do(func() {
		ctx := context.Background()
		c, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("auther_db"),
			postgres.WithUsername("auther_user"),
			postgres.WithPassword("auther_password"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			autherPoolErr = err
			return
		}
		connStr, err := c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			autherPoolErr = err
			return
		}
		autherPool, err = pgxpool.New(ctx, connStr)
		if err != nil {
			autherPoolErr = err
			return
		}
		autherPoolErr = GooseMigrate(ctx, autherPool, authermigrate.GooseFiles)
		if autherPoolErr != nil {
			autherPool.Close()
			autherPool = nil
		}
	})
	if autherPoolErr != nil {
		t.Fatal(autherPoolErr)
	}
	return autherPool
}

// TruncateAuther clears Auther tables (users and dependent refresh_tokens).
func TruncateAuther(t *testing.T, ctx context.Context, p *pgxpool.Pool) {
	t.Helper()
	_, err := p.Exec(ctx, `TRUNCATE TABLE users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal(err)
	}
}
