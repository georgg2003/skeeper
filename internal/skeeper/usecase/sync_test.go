package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/google/uuid"
)

type stubRepo struct {
	upserted  [][]models.Entry
	lastSeen  time.Time
	ret       []models.Entry
	upsertErr error
	getErr    error
}

func (s *stubRepo) UpsertEntries(ctx context.Context, userID int64, entries []models.Entry) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, entries)
	return nil
}

func (s *stubRepo) GetUpdatedAfter(ctx context.Context, userID int64, lastSync time.Time) ([]models.Entry, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	s.lastSeen = lastSync
	return s.ret, nil
}

func TestUseCase_Sync_UpsertAndFetch(t *testing.T) {
	id := uuid.New()
	repo := &stubRepo{
		ret: []models.Entry{
			{UUID: id, Type: "TEXT", Version: 2, UpdatedAt: time.Unix(300, 0).UTC()},
		},
	}
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := New(l, repo, nil)

	in := models.SyncRequest{
		LastSyncAt: time.Unix(100, 0).UTC(),
		Updates: []models.Entry{
			{UUID: uuid.New(), Type: "PASSWORD", Version: 1, UpdatedAt: time.Unix(200, 0).UTC()},
		},
	}
	ctx := contextlib.SetUserID(context.Background(), 42)
	out, err := uc.Sync(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.upserted) != 1 || len(repo.upserted[0]) != 1 {
		t.Fatalf("upsert %+v", repo.upserted)
	}
	if !repo.lastSeen.Equal(in.LastSyncAt) {
		t.Fatalf("last sync %v vs %v", repo.lastSeen, in.LastSyncAt)
	}
	if len(out.Updates) != 1 || out.Updates[0].UUID != id {
		t.Fatalf("response %+v", out.Updates)
	}
}

func TestUseCase_Sync_NoUpdates(t *testing.T) {
	repo := &stubRepo{}
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	uc := New(l, repo, nil)
	ctx := contextlib.SetUserID(context.Background(), 1)
	out, err := uc.Sync(ctx, models.SyncRequest{LastSyncAt: time.Unix(1, 0).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.upserted) != 0 {
		t.Fatal("unexpected upsert")
	}
	if len(out.Updates) != 0 {
		t.Fatalf("expected no updates, got %d", len(out.Updates))
	}
}
