package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
)

type memLocal struct {
	dirty      []models.Entry
	last       time.Time
	saved      []models.Entry
	marked     []uuid.UUID
	saveErr    error
	markErr    error
	dirtyErr   error
	lastErr    error
	remoteResp []models.Entry
	remoteErr  error
}

func (m *memLocal) GetDirtyEntries(ctx context.Context, _ *int64) ([]models.Entry, error) {
	if m.dirtyErr != nil {
		return nil, m.dirtyErr
	}
	return m.dirty, nil
}

func (m *memLocal) MarkAsSynced(ctx context.Context, id uuid.UUID) error {
	if m.markErr != nil {
		return m.markErr
	}
	m.marked = append(m.marked, id)
	return nil
}

func (m *memLocal) SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, e)
	return nil
}

func (m *memLocal) GetLastUpdate(ctx context.Context, _ *int64) (time.Time, error) {
	if m.lastErr != nil {
		return time.Time{}, m.lastErr
	}
	return m.last, nil
}

type memRemote struct {
	local *memLocal
}

func (r *memRemote) Sync(ctx context.Context, entries []models.Entry, lastUpdate time.Time) ([]models.Entry, error) {
	if r.local.remoteErr != nil {
		return nil, r.local.remoteErr
	}
	return r.local.remoteResp, nil
}

type noopSessionReaderForTests struct{}

func (noopSessionReaderForTests) GetSession(context.Context) (*models.Session, error) {
	return nil, nil
}

func TestSyncUseCase_Sync_MergesRemote(t *testing.T) {
	id := uuid.New()
	loc := &memLocal{
		dirty: []models.Entry{{UUID: id, Type: "PASSWORD", Version: 1, UpdatedAt: time.Unix(100, 0)}},
		last:  time.Unix(50, 0),
		remoteResp: []models.Entry{
			{UUID: uuid.New(), Type: "TEXT", Version: 1, UpdatedAt: time.Unix(200, 0)},
		},
	}
	rem := &memRemote{local: loc}
	uc := NewSyncUseCase(loc, rem, noopSessionReaderForTests{}, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))

	if err := uc.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(loc.saved) != 1 {
		t.Fatalf("saved %d", len(loc.saved))
	}
	if len(loc.marked) != 1 || loc.marked[0] != id {
		t.Fatalf("marked %+v", loc.marked)
	}
}

func TestSyncUseCase_Sync_RemoteError(t *testing.T) {
	loc := &memLocal{
		dirty:     []models.Entry{{UUID: uuid.New()}},
		remoteErr: context.Canceled,
	}
	rem := &memRemote{local: loc}
	uc := NewSyncUseCase(loc, rem, noopSessionReaderForTests{}, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))
	if err := uc.Sync(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
