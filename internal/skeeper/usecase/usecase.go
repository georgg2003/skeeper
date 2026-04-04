package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

var ErrInvalidToken = errors.New("refresh token is invalid")

const vaultKDFSaltSize = 16

type Repository interface {
	UpsertEntries(ctx context.Context, userID int64, entries []models.Entry) error
	GetUpdatedAfter(ctx context.Context, userID int64, lastSync time.Time) ([]models.Entry, error)
	GetVaultCrypto(ctx context.Context, userID int64) (kdfSalt, masterVerifier []byte, err error)
	PutVaultCrypto(ctx context.Context, userID int64, kdfSalt, masterVerifier []byte) error
}

type UseCase struct {
	repo Repository
	l    *slog.Logger
}

func (uc *UseCase) Sync(
	ctx context.Context,
	req models.SyncRequest,
) (models.SyncResponse, error) {
	userID, ok := contextlib.GetUserID(ctx)
	if !ok {
		return models.SyncResponse{}, ErrUnauthenticated
	}

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

// GetVaultCrypto returns the Argon2 salt and master-key verifier for the authenticated user.
func (uc *UseCase) GetVaultCrypto(ctx context.Context) (kdfSalt, masterVerifier []byte, err error) {
	userID, ok := contextlib.GetUserID(ctx)
	if !ok {
		return nil, nil, ErrUnauthenticated
	}
	return uc.repo.GetVaultCrypto(ctx, userID)
}

// PutVaultCrypto stores salt and verifier for the authenticated user (idempotent if unchanged).
func (uc *UseCase) PutVaultCrypto(ctx context.Context, kdfSalt, masterVerifier []byte) error {
	if len(kdfSalt) != vaultKDFSaltSize {
		return errors.NewValidationError("kdf_salt", fmt.Sprintf("must be exactly %d bytes", vaultKDFSaltSize))
	}
	if len(masterVerifier) != 32 {
		return errors.NewValidationError("master_verifier", "must be exactly 32 bytes (SHA-256 of derived master key)")
	}
	userID, ok := contextlib.GetUserID(ctx)
	if !ok {
		return ErrUnauthenticated
	}
	return uc.repo.PutVaultCrypto(ctx, userID, kdfSalt, masterVerifier)
}

func New(
	l *slog.Logger,
	repo Repository,
) *UseCase {
	return &UseCase{
		l:    l,
		repo: repo,
	}
}
