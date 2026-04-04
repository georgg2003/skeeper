package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/google/uuid"
)

// LocalSyncRepo reads and writes encrypted entries in the local SQLite cache.
type LocalSyncRepo interface {
	GetDirtyEntries(ctx context.Context) ([]models.Entry, error)
	MarkAsSynced(ctx context.Context, id uuid.UUID) error
	SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error
	GetLastUpdate(ctx context.Context) (time.Time, error)
}

// RemoteSyncRepo performs bidirectional sync with the Skeeper server.
type RemoteSyncRepo interface {
	Sync(ctx context.Context, entries []models.Entry, lastUpdate time.Time) ([]models.Entry, error)
}

// SyncUseCase pushes dirty local rows and merges server updates.
type SyncUseCase struct {
	local  LocalSyncRepo
	remote RemoteSyncRepo
	log    *slog.Logger
}

// NewSyncUseCase constructs a SyncUseCase.
func NewSyncUseCase(local LocalSyncRepo, remote RemoteSyncRepo, log *slog.Logger) *SyncUseCase {
	return &SyncUseCase{
		local:  local,
		remote: remote,
		log:    log.With("component", "sync_usecase"),
	}
}

// Sync runs one full sync cycle (requires a valid auth token on the remote client).
func (uc *SyncUseCase) Sync(ctx context.Context) error {
	dirty, err := uc.local.GetDirtyEntries(ctx)
	if err != nil {
		return fmt.Errorf("read local dirty: %w", err)
	}

	lastUpdate, err := uc.local.GetLastUpdate(ctx)
	if err != nil {
		return err
	}

	uc.log.InfoContext(ctx, "syncing", "dirty_count", len(dirty), "last_update", lastUpdate)

	updates, err := uc.remote.Sync(ctx, dirty, lastUpdate)
	if err != nil {
		return fmt.Errorf("grpc sync call: %w", err)
	}

	for _, remoteEntry := range updates {
		if err := uc.local.SaveEntry(ctx, remoteEntry, false); err != nil {
			return fmt.Errorf("save remote update: %w", err)
		}
	}

	for _, e := range dirty {
		if err := uc.local.MarkAsSynced(ctx, e.UUID); err != nil {
			return err
		}
	}

	uc.log.InfoContext(ctx, "sync completed", "remote_updates", len(updates))
	return nil
}
