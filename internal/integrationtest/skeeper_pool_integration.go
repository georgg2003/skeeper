package integrationtest

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	skeepermigrate "github.com/georgg2003/skeeper/migrations/skeeper"
)

var (
	skeeperPoolOnce sync.Once
	skeeperPool     *pgxpool.Pool
	skeeperPoolErr  error
)

// SkeeperPostgresPool returns a migrated pool for Skeeper repository integration tests
// (Postgres via testcontainers-go; Docker must be available).
func SkeeperPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	skeeperPoolOnce.Do(func() {
		ctx := context.Background()
		c, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("skeeper_db"),
			postgres.WithUsername("skeeper_user"),
			postgres.WithPassword("skeeper_password"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			skeeperPoolErr = err
			return
		}
		connStr, err := c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			skeeperPoolErr = err
			return
		}
		skeeperPool, err = pgxpool.New(ctx, connStr)
		if err != nil {
			skeeperPoolErr = err
			return
		}
		skeeperPoolErr = GooseMigrate(ctx, skeeperPool, skeepermigrate.GooseFiles)
		if skeeperPoolErr != nil {
			skeeperPool.Close()
			skeeperPool = nil
		}
	})
	if skeeperPoolErr != nil {
		t.Fatal(skeeperPoolErr)
	}
	return skeeperPool
}
