package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

// LocalSyncRepo reads dirty entries and applies post-sync merges from the server.
type LocalSyncRepo interface {
	GetDirtyEntries(ctx context.Context, forUserID *int64) ([]models.Entry, error)
	PersistSyncResult(ctx context.Context, userID int64, updates []models.Entry, dirty []models.Entry, applied map[uuid.UUID]struct{}) error
	SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error
	GetLastUpdate(ctx context.Context, forUserID *int64) (time.Time, error)
}

// RemoteSyncRepo calls the Skeeper Sync RPC with ciphertext entries only.
type RemoteSyncRepo interface {
	Sync(ctx context.Context, entries []models.Entry, lastUpdate time.Time) ([]models.Entry, []uuid.UUID, error)
}

// SyncUseCase pushes local changes and merges server updates into the vault DB.
type SyncUseCase struct {
	local   LocalSyncRepo
	remote  RemoteSyncRepo
	session SessionReader
	log     *slog.Logger
}

// NewSyncUseCase wires the local vault store to the Skeeper gRPC client.
func NewSyncUseCase(local LocalSyncRepo, remote RemoteSyncRepo, session SessionReader, log *slog.Logger) *SyncUseCase {
	return &SyncUseCase{
		local:   local,
		remote:  remote,
		session: session,
		log:     log.With("component", "sync_usecase"),
	}
}

func (uc *SyncUseCase) requireAutherUserID(ctx context.Context) (int64, error) {
	s, err := uc.session.GetSession(ctx)
	if err != nil || s == nil || s.UserID == nil {
		return 0, errors.New("sync requires an active session; log in first")
	}
	return *s.UserID, nil
}

// Sync uploads dirty entries, downloads newer remote rows, and updates local state.
func (uc *SyncUseCase) Sync(ctx context.Context) error {
	uid, err := uc.requireAutherUserID(ctx)
	if err != nil {
		return err
	}

	dirty, err := uc.local.GetDirtyEntries(ctx, &uid)
	if err != nil {
		return fmt.Errorf("read local dirty: %w", err)
	}

	lastUpdate, err := uc.local.GetLastUpdate(ctx, &uid)
	if err != nil {
		return err
	}

	uc.log.InfoContext(ctx, "syncing", "dirty_count", len(dirty), "last_update", lastUpdate)

	updates, applied, err := uc.remote.Sync(ctx, dirty, lastUpdate)
	if err != nil {
		return fmt.Errorf("grpc sync call: %w", err)
	}

	appliedSet := make(map[uuid.UUID]struct{}, len(applied))
	for _, id := range applied {
		appliedSet[id] = struct{}{}
	}

	if err := uc.local.PersistSyncResult(ctx, uid, updates, dirty, appliedSet); err != nil {
		return fmt.Errorf("persist sync: %w", err)
	}

	uc.log.InfoContext(ctx, "sync completed", "remote_updates", len(updates), "applied_local", len(applied))
	return nil
}
