package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

var ErrInvalidToken = errors.New("refresh token is invalid")

type Repository interface {
	UpsertEntries(ctx context.Context, userID int64, entries []models.Entry) error
	GetUpdatedAfter(ctx context.Context, userID int64, lastSync time.Time) ([]models.Entry, error)
}

type UseCase struct {
	repo      Repository
	jwtHelper *jwthelper.JWTHelper
	l         *slog.Logger
}

func (uc *UseCase) Sync(
	ctx context.Context,
	req models.SyncRequest,
) (models.SyncResponse, error) {
	userID := contextlib.MustGetUserID(ctx)

	if len(req.Updates) > 0 {
		err := uc.repo.UpsertEntries(ctx, userID, req.Updates)
		if err != nil {
			return models.SyncResponse{}, errors.Wrap(err, "failed to upsert entries")
		}
	}

	currentSyncAt := time.Now()
	serverUpdates, err := uc.repo.GetUpdatedAfter(ctx, userID, req.LastSyncAt)
	if err != nil {
		return models.SyncResponse{}, errors.Wrap(err, "failed to get updates")
	}

	return models.SyncResponse{
		CurrentSyncAt: currentSyncAt,
		Updates:       serverUpdates,
	}, nil
}

func New(
	l *slog.Logger,
	repo Repository,
	jwtHelper *jwthelper.JWTHelper,
) *UseCase {
	return &UseCase{
		l:         l,
		repo:      repo,
		jwtHelper: jwtHelper,
	}
}
