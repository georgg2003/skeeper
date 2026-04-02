package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

type LocalRepo interface {
	GetDirtyEntries(ctx context.Context) ([]models.Entry, error)
	MarkAsSynced(ctx context.Context, uuid string) error
	SaveEntry(ctx context.Context, e models.Entry, isDirty bool) error
	GetLastUpdate(ctx context.Context) (time.Time, error)
}

type RemoteRepo interface {
	Sync(ctx context.Context, entries []models.Entry, lastUpdate time.Time) ([]models.Entry, error)
}

type SyncUseCase struct {
	local  LocalRepo
	remote RemoteRepo
}

func (uc *SyncUseCase) Sync(ctx context.Context) error {
	dirty, err := uc.local.GetDirtyEntries(ctx)
	if err != nil {
		return fmt.Errorf("read local dirty: %w", err)
	}

	lastUpdate, err := uc.local.GetLastUpdate(ctx)
	if err != nil {
		return err
	}

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
		if err := uc.local.MarkAsSynced(ctx, e.UUID.String()); err != nil {
			return err
		}
	}

	return nil
}
