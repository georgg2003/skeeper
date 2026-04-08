package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
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
	// If non-nil, memRemote returns this as applied_update_uuids; if nil, all sent entry UUIDs are "applied".
	appliedIDs []uuid.UUID
}

func (m *memLocal) GetDirtyEntries(ctx context.Context, _ *int64) ([]models.Entry, error) {
	if m.dirtyErr != nil {
		return nil, m.dirtyErr
	}
	return m.dirty, nil
}

func (m *memLocal) PersistSyncResult(
	ctx context.Context,
	userID int64,
	updates []models.Entry,
	dirty []models.Entry,
	applied map[uuid.UUID]struct{},
) error {
	for i := range updates {
		e := updates[i]
		u := userID
		e.UserID = &u
		if err := m.SaveEntry(ctx, e, false); err != nil {
			return err
		}
	}
	for _, e := range dirty {
		if _, ok := applied[e.UUID]; !ok {
			continue
		}
		if m.markErr != nil {
			return m.markErr
		}
		m.marked = append(m.marked, e.UUID)
	}
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

func (r *memRemote) Sync(ctx context.Context, entries []models.Entry, lastUpdate time.Time) ([]models.Entry, []uuid.UUID, error) {
	if r.local.remoteErr != nil {
		return nil, nil, r.local.remoteErr
	}
	applied := r.local.appliedIDs
	if applied == nil {
		applied = make([]uuid.UUID, 0, len(entries))
		for i := range entries {
			applied = append(applied, entries[i].UUID)
		}
	}
	return r.local.remoteResp, applied, nil
}

type noopSessionReaderForTests struct{}

func (noopSessionReaderForTests) GetSession(context.Context) (*models.Session, error) {
	return nil, nil
}

type sessionReaderWithUser struct {
	uid int64
}

func (s sessionReaderWithUser) GetSession(context.Context) (*models.Session, error) {
	u := s.uid
	return &models.Session{UserID: &u}, nil
}

func testSyncLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
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
	uc := NewSyncUseCase(loc, rem, sessionReaderWithUser{uid: 1}, testSyncLogger())

	if err := uc.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(loc.saved) != 1 {
		t.Fatalf("saved %d", len(loc.saved))
	}
	if loc.saved[0].UserID == nil || *loc.saved[0].UserID != 1 {
		t.Fatalf("remote entry should get session user id, got %+v", loc.saved[0].UserID)
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
	uc := NewSyncUseCase(loc, rem, sessionReaderWithUser{uid: 1}, testSyncLogger())
	if err := uc.Sync(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestSyncUseCase_Sync_NoSession(t *testing.T) {
	loc := &memLocal{dirty: []models.Entry{{UUID: uuid.New()}}}
	rem := &memRemote{local: loc}
	uc := NewSyncUseCase(loc, rem, noopSessionReaderForTests{}, testSyncLogger())
	if err := uc.Sync(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestSyncUseCase_Sync_UnappliedLocalStaysDirty(t *testing.T) {
	id := uuid.New()
	loc := &memLocal{
		dirty:      []models.Entry{{UUID: id, Type: "PASSWORD", Version: 1, UpdatedAt: time.Unix(1, 0)}},
		last:       time.Unix(0, 0),
		appliedIDs: []uuid.UUID{}, // server rejected all (e.g. stale version)
	}
	rem := &memRemote{local: loc}
	uc := NewSyncUseCase(loc, rem, sessionReaderWithUser{uid: 2}, testSyncLogger())
	if err := uc.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(loc.marked) != 0 {
		t.Fatalf("expected no mark-as-synced, got %+v", loc.marked)
	}
}
