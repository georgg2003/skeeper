package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/georgg2003/skeeper/internal/integrationtest"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/vaulterror"
	"github.com/georgg2003/skeeper/internal/skeeper/repository/postgres"
)

// Repository integration tests: Postgres via testcontainers-go (Docker required).

func truncateSkeeper(t *testing.T, ctx context.Context, p *pgxpool.Pool) {
	t.Helper()
	_, err := p.Exec(ctx, `TRUNCATE TABLE entries, vault_crypto`)
	require.NoError(t, err)
}

func newSkeeperRepository(t *testing.T, ctx context.Context, p *pgxpool.Pool) *postgres.Repository {
	t.Helper()
	truncateSkeeper(t, ctx, p)
	return postgres.NewFromPool(p)
}

func TestIntegration_UpsertAndGetUpdatedAfter(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
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
	_, err := repo.UpsertEntries(ctx, userID, []models.Entry{e1})
	require.NoError(t, err)

	got, err := repo.GetUpdatedAfter(ctx, userID, t1.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0].UUID)
	assert.Equal(t, int64(1), got[0].Version)

	empty, err := repo.GetUpdatedAfter(ctx, userID, t1.Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestIntegration_Upsert_VersionIncreases(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 7

	id := uuid.New()
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, UpdatedAt: base,
	}})
	require.NoError(t, err)

	t2 := base.Add(time.Hour)
	_, err = repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{9}, Payload: []byte{8},
		Version: 2, UpdatedAt: t2,
	}})
	require.NoError(t, err)

	got, err := repo.GetUpdatedAfter(ctx, userID, base.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, int64(2), got[0].Version)
	assert.True(t, got[0].UpdatedAt.Equal(t2))
}

func TestIntegration_Upsert_DoesNotDowngradeVersion(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 8

	id := uuid.New()
	tHigh := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 5, UpdatedAt: tHigh,
	}})
	require.NoError(t, err)

	tLow := tHigh.Add(time.Hour)
	_, err = repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{3}, Payload: []byte{4},
		Version: 3, UpdatedAt: tLow,
	}})
	require.NoError(t, err)

	got, err := repo.GetUpdatedAfter(ctx, userID, tHigh.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, int64(5), got[0].Version, "row should stay at v5, got %+v", got)
	assert.Equal(t, byte(2), got[0].Payload[0], "payload should not be overwritten by lower version")
}

func TestIntegration_UpsertEntries_EmptyNoOp(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	appliedNil, err := repo.UpsertEntries(ctx, 1, nil)
	require.NoError(t, err, "nil batch")
	assert.Empty(t, appliedNil, "nil batch")
	appliedEmpty, err := repo.UpsertEntries(ctx, 1, []models.Entry{})
	require.NoError(t, err, "empty batch")
	assert.Empty(t, appliedEmpty, "empty batch")
}

func TestIntegration_GetUpdatedAfter_UserIsolation(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)

	const userA int64 = 101
	const userB int64 = 102
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	idA := uuid.New()
	idB := uuid.New()

	_, err := repo.UpsertEntries(ctx, userA, []models.Entry{{
		UUID: idA, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, UpdatedAt: ts,
	}})
	require.NoError(t, err)
	_, err = repo.UpsertEntries(ctx, userB, []models.Entry{{
		UUID: idB, Type: "TEXT", EncryptedDek: []byte{3}, Payload: []byte{4},
		Version: 1, UpdatedAt: ts,
	}})
	require.NoError(t, err)

	gotA, err := repo.GetUpdatedAfter(ctx, userA, ts.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, idA, gotA[0].UUID, "user A: %+v", gotA)

	gotB, err := repo.GetUpdatedAfter(ctx, userB, ts.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, idB, gotB[0].UUID, "user B: %+v", gotB)
}

func TestIntegration_UpsertEntries_BatchRoundtrip(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
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

	_, err := repo.UpsertEntries(ctx, userID, entries)
	require.NoError(t, err)

	got, err := repo.GetUpdatedAfter(ctx, userID, ts.Add(-time.Second))
	require.NoError(t, err)
	require.Len(t, got, 3, "want 3 rows, got %d: %+v", len(got), got)
	for i := 1; i < len(got); i++ {
		assert.False(t, got[i].UpdatedAt.Before(got[i-1].UpdatedAt),
			"expected ORDER BY updated_at ASC, got %v before %v", got[i-1].UpdatedAt, got[i].UpdatedAt)
	}

	byID := make(map[uuid.UUID]models.Entry, len(got))
	for _, e := range got {
		byID[e.UUID] = e
	}
	for _, want := range entries {
		g, ok := byID[want.UUID]
		require.True(t, ok, "missing uuid %s", want.UUID)
		assert.Equal(t, want.Type, g.Type)
		assert.Equal(t, want.Version, g.Version)
		assert.Equal(t, want.IsDeleted, g.IsDeleted)
		assert.Equal(t, string(want.EncryptedDek), string(g.EncryptedDek), "uuid %s encrypted dek mismatch", want.UUID)
		assert.Equal(t, string(want.Payload), string(g.Payload), "uuid %s payload mismatch", want.UUID)
		if len(want.Meta) == 0 && len(g.Meta) == 0 {
			continue
		}
		assert.Equal(t, string(want.Meta), string(g.Meta), "uuid %s meta %q vs %q", want.UUID, g.Meta, want.Meta)
	}
}

func TestIntegration_GetUpdatedAfter_OrderedByUpdatedAtAsc(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 66
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Insert out of chronological order; result should still be sorted ascending.
	second := base.Add(time.Hour)
	first := base.Add(30 * time.Minute)
	third := base.Add(90 * time.Minute)

	_, err := repo.UpsertEntries(ctx, userID, []models.Entry{
		{UUID: uuid.New(), Type: "A", EncryptedDek: []byte{1}, Payload: []byte{1}, Version: 1, UpdatedAt: second},
		{UUID: uuid.New(), Type: "B", EncryptedDek: []byte{1}, Payload: []byte{1}, Version: 1, UpdatedAt: first},
		{UUID: uuid.New(), Type: "C", EncryptedDek: []byte{1}, Payload: []byte{1}, Version: 1, UpdatedAt: third},
	})
	require.NoError(t, err)

	got, err := repo.GetUpdatedAfter(ctx, userID, base)
	require.NoError(t, err)
	require.Len(t, got, 3, "got %d rows", len(got))
	assert.True(t, got[0].UpdatedAt.Equal(first))
	assert.True(t, got[1].UpdatedAt.Equal(second))
	assert.True(t, got[2].UpdatedAt.Equal(third))
}

func TestIntegration_GetUpdatedAfter_ExcludesRowsAtExactlyLastSync(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 77
	t0 := time.Date(2026, 6, 1, 15, 30, 0, 0, time.UTC)
	id := uuid.New()

	_, err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, UpdatedAt: t0,
	}})
	require.NoError(t, err)

	atBoundary, err := repo.GetUpdatedAfter(ctx, userID, t0)
	require.NoError(t, err)
	assert.Empty(t, atBoundary, "updated_at == lastSync should be excluded, got %+v", atBoundary)

	after, err := repo.GetUpdatedAfter(ctx, userID, t0.Add(-time.Nanosecond))
	require.NoError(t, err)
	require.Len(t, after, 1)
	assert.Equal(t, id, after[0].UUID)
}

func TestIntegration_Upsert_SameVersion_DoesNotOverwrite(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 88
	id := uuid.New()
	ts := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	applied1, err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{0xaa},
		Version: 4, UpdatedAt: ts,
	}})
	require.NoError(t, err)
	require.Len(t, applied1, 1)
	assert.Equal(t, id, applied1[0], "first upsert applied %v want [%s]", applied1, id)

	applied2, err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{9}, Payload: []byte{0xbb},
		Version: 4, UpdatedAt: ts.Add(time.Hour),
	}})
	require.NoError(t, err)
	assert.Empty(t, applied2, "same-version upsert should apply 0 rows, got %v", applied2)

	got, err := repo.GetUpdatedAfter(ctx, userID, ts.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, byte(0xaa), got[0].Payload[0], "payload should stay 0xaa, got %+v", got)
}

func TestIntegration_Upsert_SoftDeleteRoundtrip(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 99
	id := uuid.New()
	t1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	_, err := repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 1, IsDeleted: false, UpdatedAt: t1,
	}})
	require.NoError(t, err)

	t2 := t1.Add(time.Hour)
	_, err = repo.UpsertEntries(ctx, userID, []models.Entry{{
		UUID: id, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Version: 2, IsDeleted: true, UpdatedAt: t2,
	}})
	require.NoError(t, err)

	got, err := repo.GetUpdatedAfter(ctx, userID, t1.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.True(t, got[0].IsDeleted)
	assert.Equal(t, int64(2), got[0].Version)
}

func TestIntegration_VaultCryptoPutGet(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 501
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}
	ver := make([]byte, 32)
	for i := range ver {
		ver[i] = byte(i + 1)
	}

	_, _, err := repo.GetVaultCrypto(ctx, userID)
	require.Error(t, err)
	require.True(t, errors.Is(err, vaulterror.ErrNotFound), "expected ErrNotFound, got %v", err)

	require.NoError(t, repo.PutVaultCrypto(ctx, userID, salt, ver))
	gotSalt, gotVer, err := repo.GetVaultCrypto(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, string(salt), string(gotSalt))
	assert.Equal(t, string(ver), string(gotVer))
	require.NoError(t, repo.PutVaultCrypto(ctx, userID, salt, ver), "idempotent put")
	otherSalt := append([]byte(nil), salt...)
	otherSalt[0] ^= 0xff
	err = repo.PutVaultCrypto(ctx, userID, otherSalt, ver)
	require.Error(t, err)
	require.True(t, errors.Is(err, vaulterror.ErrConflict), "expected conflict, got %v", err)
}

func TestIntegration_PutVaultCrypto_ConcurrentFirstInsert(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 602
	salt := make([]byte, 16)
	ver := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i + 7)
	}
	for i := range ver {
		ver[i] = byte(i)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var err0, err1 error
	go func() {
		defer wg.Done()
		err0 = repo.PutVaultCrypto(ctx, userID, salt, ver)
	}()
	go func() {
		defer wg.Done()
		err1 = repo.PutVaultCrypto(ctx, userID, salt, ver)
	}()
	wg.Wait()
	require.False(t, err0 != nil && err1 != nil, "both failed: %v %v", err0, err1)
	gotSalt, gotVer, err := repo.GetVaultCrypto(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, string(salt), string(gotSalt), "stored vault crypto mismatch after concurrent put")
	assert.Equal(t, string(ver), string(gotVer), "stored vault crypto mismatch after concurrent put")
}

func TestIntegration_PutVaultCrypto_InvalidPayloadSizes(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	salt16 := make([]byte, 16)
	ver32 := make([]byte, 32)

	err := repo.PutVaultCrypto(ctx, 1, make([]byte, 15), ver32)
	require.Error(t, err, "short salt")
	assert.Contains(t, err.Error(), "invalid")
	err = repo.PutVaultCrypto(ctx, 1, salt16, make([]byte, 31))
	require.Error(t, err, "short verifier")
	assert.Contains(t, err.Error(), "invalid")
}

func TestIntegration_GetVaultCrypto_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err := repo.GetVaultCrypto(canceled, 1)
	require.Error(t, err)
	require.False(t, errors.Is(err, vaulterror.ErrNotFound), "expected non-NotFound error, got %v", err)
}

func TestIntegration_PutVaultCrypto_ContextCanceled(t *testing.T) {
	base := context.Background()
	repo := newSkeeperRepository(
		t,
		base,
		integrationtest.SkeeperPostgresPool(t),
	)
	salt := make([]byte, 16)
	ver := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i)
	}
	canceled, cancel := context.WithCancel(base)
	cancel()
	err := repo.PutVaultCrypto(canceled, 88801, salt, ver)
	require.Error(t, err, "expected error on canceled insert path")
}

func TestIntegration_PutVaultCrypto_QueryCanceledOnIdempotentCheck(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userID int64 = 88802
	salt := make([]byte, 16)
	ver := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i + 3)
	}
	for i := range ver {
		ver[i] = byte(i)
	}
	require.NoError(t, repo.PutVaultCrypto(ctx, userID, salt, ver))
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	err := repo.PutVaultCrypto(canceled, userID, salt, ver)
	require.Error(t, err, "expected error when SELECT is canceled")
}

// TestIntegration_Upsert_SameUUID_DifferentUsersCannotOverwrite verifies (user_id, uuid) PK isolation:
// user B cannot clobber user A's row by reusing the same UUID with a higher version.
func TestIntegration_Upsert_SameUUID_DifferentUsersCannotOverwrite(t *testing.T) {
	ctx := context.Background()
	repo := newSkeeperRepository(
		t,
		ctx,
		integrationtest.SkeeperPostgresPool(t),
	)
	const userA int64 = 201
	const userB int64 = 202
	ts := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	sharedID := uuid.New()

	eA := models.Entry{
		UUID: sharedID, Type: "TEXT", EncryptedDek: []byte{1}, Payload: []byte{2},
		Meta: []byte{3}, Version: 1, UpdatedAt: ts, IsDeleted: false,
	}
	_, err := repo.UpsertEntries(ctx, userA, []models.Entry{eA})
	require.NoError(t, err)

	eB := models.Entry{
		UUID: sharedID, Type: "TEXT", EncryptedDek: []byte{9}, Payload: []byte{9},
		Meta: []byte{9}, Version: 99, UpdatedAt: ts.Add(time.Hour), IsDeleted: false,
	}
	_, err = repo.UpsertEntries(ctx, userB, []models.Entry{eB})
	require.NoError(t, err)

	gotA, err := repo.GetUpdatedAfter(ctx, userA, ts.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, int64(1), gotA[0].Version, "user A row corrupted: %+v", gotA)
	assert.Equal(t, byte(2), gotA[0].Payload[0], "user A row corrupted: %+v", gotA)

	gotB, err := repo.GetUpdatedAfter(ctx, userB, ts.Add(-time.Minute))
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, int64(99), gotB[0].Version, "user B row: %+v", gotB)
	assert.Equal(t, byte(9), gotB[0].Payload[0], "user B row: %+v", gotB)
}
