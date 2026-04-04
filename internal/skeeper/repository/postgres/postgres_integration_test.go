//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/integrationtest"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository integration tests: Postgres via testcontainers-go (Docker required).

func truncateSkeeper(t *testing.T, ctx context.Context, p *pgxpool.Pool) {
	t.Helper()
	_, err := p.Exec(ctx, `TRUNCATE TABLE entries`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_UpsertAndGetUpdatedAfter(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
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
	p := integrationtest.SkeeperPostgresPool(t)
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
	p := integrationtest.SkeeperPostgresPool(t)
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
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	if err := repo.UpsertEntries(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertEntries(ctx, 1, []models.Entry{}); err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_GetUpdatedAfter_UserIsolation(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}

	const userA int64 = 101
	const userB int64 = 102
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	idA := uuid.New()
	idB := uuid.New()

	if err := repo.UpsertEntries(ctx, userA, []models.Entry{{
		UUID: idA, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, UpdatedAt: ts,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertEntries(ctx, userB, []models.Entry{{
		UUID: idB, Type: "TEXT", EncryptedDek: []byte{3}, Payload: []byte{4},
		Version: 1, UpdatedAt: ts,
	}}); err != nil {
		t.Fatal(err)
	}

	gotA, err := repo.GetUpdatedAfter(ctx, userA, ts.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotA) != 1 || gotA[0].UUID != idA {
		t.Fatalf("user A: %+v", gotA)
	}

	gotB, err := repo.GetUpdatedAfter(ctx, userB, ts.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotB) != 1 || gotB[0].UUID != idB {
		t.Fatalf("user B: %+v", gotB)
	}
}

func TestIntegration_UpsertEntries_BatchRoundtrip(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 55
	ts := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	entries := []models.Entry{
		{
			UUID: uuid.New(), Type: "PASSWORD", EncryptedDek: []byte{0xde, 0xad},
			Payload: []byte("p1"), Meta: []byte("m1"), Version: 1, UpdatedAt: ts,
		},
		{
			UUID: uuid.New(), Type: "CARD", EncryptedDek: []byte{0xbe, 0xef},
			Payload: []byte("p2"), Meta: nil, Version: 2, IsDeleted: false, UpdatedAt: ts.Add(time.Minute),
		},
		{
			UUID: uuid.New(), Type: "NOTE", EncryptedDek: []byte{1, 2, 3},
			Payload: []byte{}, Meta: []byte{}, Version: 1, IsDeleted: true, UpdatedAt: ts.Add(2 * time.Minute),
		},
	}

	if err := repo.UpsertEntries(ctx, userID, entries); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, ts.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 rows, got %d: %+v", len(got), got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].UpdatedAt.Before(got[i-1].UpdatedAt) {
			t.Fatalf("expected ORDER BY updated_at ASC, got %v before %v", got[i-1].UpdatedAt, got[i].UpdatedAt)
		}
	}

	byID := make(map[uuid.UUID]models.Entry, len(got))
	for _, e := range got {
		byID[e.UUID] = e
	}
	for _, want := range entries {
		g, ok := byID[want.UUID]
		if !ok {
			t.Fatalf("missing uuid %s", want.UUID)
		}
		if g.Type != want.Type || g.Version != want.Version || g.IsDeleted != want.IsDeleted {
			t.Fatalf("uuid %s: got %+v want %+v", want.UUID, g, want)
		}
		if string(g.EncryptedDek) != string(want.EncryptedDek) || string(g.Payload) != string(want.Payload) {
			t.Fatalf("uuid %s bytes mismatch", want.UUID)
		}
		if len(want.Meta) == 0 && len(g.Meta) == 0 {
			continue
		}
		if string(g.Meta) != string(want.Meta) {
			t.Fatalf("uuid %s meta %q vs %q", want.UUID, g.Meta, want.Meta)
		}
	}
}

func TestIntegration_GetUpdatedAfter_OrderedByUpdatedAtAsc(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 66
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Insert out of chronological order; result should still be sorted ascending.
	second := base.Add(time.Hour)
	first := base.Add(30 * time.Minute)
	third := base.Add(90 * time.Minute)

	if err := repo.UpsertEntries(ctx, userID, []models.Entry{
		{UUID: uuid.New(), Type: "A", EncryptedDek: []byte{1}, Payload: []byte{1}, Version: 1, UpdatedAt: second},
		{UUID: uuid.New(), Type: "B", EncryptedDek: []byte{1}, Payload: []byte{1}, Version: 1, UpdatedAt: first},
		{UUID: uuid.New(), Type: "C", EncryptedDek: []byte{1}, Payload: []byte{1}, Version: 1, UpdatedAt: third},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows", len(got))
	}
	if !got[0].UpdatedAt.Equal(first) || !got[1].UpdatedAt.Equal(second) || !got[2].UpdatedAt.Equal(third) {
		t.Fatalf("order: %+v", got)
	}
}

func TestIntegration_GetUpdatedAfter_ExcludesRowsAtExactlyLastSync(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 77
	t0 := time.Date(2026, 6, 1, 15, 30, 0, 0, time.UTC)
	id := uuid.New()

	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, UpdatedAt: t0,
	}}); err != nil {
		t.Fatal(err)
	}

	atBoundary, err := repo.GetUpdatedAfter(ctx, userID, t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(atBoundary) != 0 {
		t.Fatalf("updated_at == lastSync should be excluded, got %+v", atBoundary)
	}

	after, err := repo.GetUpdatedAfter(ctx, userID, t0.Add(-time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].UUID != id {
		t.Fatalf("%+v", after)
	}
}

func TestIntegration_Upsert_SameVersion_DoesNotOverwrite(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 88
	id := uuid.New()
	ts := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{0xaa},
		Version: 4, UpdatedAt: ts,
	}}); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{9}, Payload: []byte{0xbb},
		Version: 4, UpdatedAt: ts.Add(time.Hour),
	}}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, ts.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Payload[0] != 0xaa {
		t.Fatalf("payload should stay 0xaa, got %+v", got)
	}
}

func TestIntegration_Upsert_SoftDeleteRoundtrip(t *testing.T) {
	ctx := context.Background()
	p := integrationtest.SkeeperPostgresPool(t)
	truncateSkeeper(t, ctx, p)
	repo := &Repository{pool: p}
	const userID int64 = 99
	id := uuid.New()
	t1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, IsDeleted: false, UpdatedAt: t1,
	}}); err != nil {
		t.Fatal(err)
	}

	t2 := t1.Add(time.Hour)
	if err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 2, IsDeleted: true, UpdatedAt: t2,
	}}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetUpdatedAfter(ctx, userID, t1.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsDeleted || got[0].Version != 2 {
		t.Fatalf("%+v", got)
	}
}
