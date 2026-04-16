package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
)

func discardSyncLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestUseCase_Sync(t *testing.T) {
	id := uuid.New()
	baseLast := time.Unix(100, 0).UTC()

	t.Run("upserts_then_fetches", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := NewMockRepository(ctrl)
		clientUUID := uuid.New()
		updates := []models.Entry{{UUID: clientUUID, Type: "PASSWORD", Version: 1, UpdatedAt: time.Unix(200, 0).UTC()}}
		req := models.SyncRequest{LastSyncAt: baseLast, Updates: updates}
		serverRows := []models.Entry{{UUID: id, Type: "TEXT", Version: 2, UpdatedAt: time.Unix(300, 0).UTC()}}

		gomock.InOrder(
			repo.EXPECT().UpsertEntries(gomock.Any(), int64(42), updates).Return([]uuid.UUID{clientUUID}, nil),
			repo.EXPECT().GetUpdatedAfter(gomock.Any(), int64(42), baseLast).Return(serverRows, nil),
		)

		uc := New(discardSyncLog(), repo)
		ctx := contextlib.SetUserID(context.Background(), 42)
		out, err := uc.Sync(ctx, req)
		require.NoError(t, err)
		require.Len(t, out.Updates, 1)
		assert.Equal(t, id, out.Updates[0].UUID)
		require.Len(t, out.AppliedUpdateUUIDs, 1)
		assert.Equal(t, clientUUID, out.AppliedUpdateUUIDs[0])
	})

	t.Run("no_client_updates_skips_upsert", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := NewMockRepository(ctrl)
		last := time.Unix(1, 0).UTC()
		req := models.SyncRequest{LastSyncAt: last}
		repo.EXPECT().GetUpdatedAfter(gomock.Any(), int64(1), last).Return(nil, nil)

		uc := New(discardSyncLog(), repo)
		ctx := contextlib.SetUserID(context.Background(), 1)
		out, err := uc.Sync(ctx, req)
		require.NoError(t, err)
		assert.Empty(t, out.Updates, "expected no updates")
		assert.Empty(t, out.AppliedUpdateUUIDs)
	})

	t.Run("upsert_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := NewMockRepository(ctrl)
		updates := []models.Entry{{UUID: id}}
		req := models.SyncRequest{Updates: updates}
		repo.EXPECT().UpsertEntries(gomock.Any(), int64(1), updates).Return(nil, context.Canceled)

		uc := New(discardSyncLog(), repo)
		ctx := contextlib.SetUserID(context.Background(), 1)
		_, err := uc.Sync(ctx, req)
		require.Error(t, err, "expected error")
	})

	t.Run("get_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := NewMockRepository(ctrl)
		last := time.Unix(1, 0).UTC()
		req := models.SyncRequest{LastSyncAt: last}
		repo.EXPECT().GetUpdatedAfter(gomock.Any(), int64(1), last).Return(nil, context.Canceled)

		uc := New(discardSyncLog(), repo)
		ctx := contextlib.SetUserID(context.Background(), 1)
		_, err := uc.Sync(ctx, req)
		require.Error(t, err, "expected error")
	})

	t.Run("missing_user_in_context", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := NewMockRepository(ctrl)
		req := models.SyncRequest{Updates: []models.Entry{{UUID: id, Version: 1}}}

		uc := New(discardSyncLog(), repo)
		out, err := uc.Sync(context.Background(), req)
		require.Error(t, err, "expected error")
		assert.True(t, errors.Is(err, ErrUnauthenticated), "expected ErrUnauthenticated, got %v", err)
		assert.Empty(t, out.Updates)
	})
}
