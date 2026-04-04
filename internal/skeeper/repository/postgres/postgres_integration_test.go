//go:build integration

package postgres

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/integrationtest"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	skeepermigrate "github.com/georgg2003/skeeper/migrations/skeeper"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Set SKEEPER_TEST_DSN to a Postgres URL matching docker-compose (e.g.
// postgres://skeeper_user:skeeper_password@127.0.0.1:5432/skeeper_db?sslmode=disable).
// Run: docker compose up -d skeeper-db && go test -tags=integration ./internal/skeeper/repository/postgres/...

var (
	skeeperPoolOnce sync.Once
	skeeperPool     *pgxpool.Pool
	skeeperPoolErr  error
)

func skeeperTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("SKEEPER_TEST_DSN")
	if dsn == "" {
		t.Skip("SKEEPER_TEST_DSN not set (start skeeper-db from docker-compose.yaml)")
	}
	skeeperPoolOnce.Do(func() {
		ctx := context.Background()
		skeeperPool, skeeperPoolErr = pgxpool.New(ctx, dsn)
		if skeeperPoolErr != nil {
			return
		}
		skeeperPoolErr = integrationtest.GooseMigrate(ctx, skeeperPool, skeepermigrate.GooseFiles)
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

func truncateSkeeper(t *testing.T, ctx context.Context, p *pgxpool.Pool) {
	t.Helper()
	_, err := p.Exec(ctx, `TRUNCATE TABLE entries`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_UpsertAndGetUpdatedAfter(t *testing.T) {
	ctx := context.Background()
	p := skeeperTestPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 42

	id := uuid.New()
	t1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	e1 := models.Entry{
		UUID:         id,
		Type:         "PASSWORD",
		EncryptedDek: []byte{1},
		Payload:      []byte{2},
		Meta:         []byte{3},
		Version:      1,
		UpdatedAt:    t1,
	}
	if err := repo.UpsertEntries(ctx, userID, []models.Entry{e1}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, t1.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].UUID != id || got[0].Version != 1 {
		t.Fatalf("%+v", got)
	}

	empty, err := repo.GetUpdatedAfter(ctx, userID, t1.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no rows, got %+v", empty)
	}
}

func TestIntegration_Upsert_VersionIncreases(t *testing.T) {
	ctx := context.Background()
	p := skeeperTestPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 7

	id := uuid.New()
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, UpdatedAt: base,
	}}); err != nil {
		t.Fatal(err)
	}

	t2 := base.Add(time.Hour)
	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{9}, Payload: []byte{8},
		Version: 2, UpdatedAt: t2,
	}}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, base.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Version != 2 || !got[0].UpdatedAt.Equal(t2) {
		t.Fatalf("%+v", got)
	}
}

func TestIntegration_Upsert_DoesNotDowngradeVersion(t *testing.T) {
	ctx := context.Background()
	p := skeeperTestPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 8

	id := uuid.New()
	tHigh := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 5, UpdatedAt: tHigh,
	}}); err != nil {
		t.Fatal(err)
	}

	tLow := tHigh.Add(time.Hour)
	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{3}, Payload: []byte{4},
		Version: 3, UpdatedAt: tLow,
	}}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, tHigh.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Version != 5 {
		t.Fatalf("row should stay at v5, got %+v", got)
	}
	if got[0].Payload[0] != 2 {
		t.Fatal("payload should not be overwritten by lower version")
	}
}

func TestIntegration_UpsertEntries_EmptyNoOp(t *testing.T) {
	ctx := context.Background()
	p := skeeperTestPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	if err := repo.UpsertEntries(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertEntries(ctx, 1, []models.Entry{}); err != nil {
		t.Fatal(err)
	}
}
