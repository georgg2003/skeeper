package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestSyncUseCase_Sync_MergesRemote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	id := uuid.New()
	dirty := []models.Entry{{UUID: id, Type: models.EntryTypePassword, Version: 1, UpdatedAt: time.Unix(100, 0)}}
	last := time.Unix(50, 0)
	remoteRow := models.Entry{UUID: uuid.New(), Type: models.EntryTypeText, Version: 1, UpdatedAt: time.Unix(200, 0)}

	sess := NewMockSessionReader(ctrl)
	sess.EXPECT().GetSession(gomock.Any()).Return(&models.Session{UserID: &uid}, nil)

	local := NewMockLocalSyncRepo(ctrl)
	rem := NewMockRemoteSyncRepo(ctrl)

	gomock.InOrder(
		local.EXPECT().GetDirtyEntries(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *int64) ([]models.Entry, error) {
			require.NotNil(t, p)
			require.Equal(t, uid, *p)
			return dirty, nil
		}),
		local.EXPECT().GetLastUpdate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *int64) (time.Time, error) {
			require.NotNil(t, p)
			require.Equal(t, uid, *p)
			return last, nil
		}),
		rem.EXPECT().Sync(gomock.Any(), dirty, last).Return([]models.Entry{remoteRow}, []uuid.UUID{id}, nil),
		local.EXPECT().PersistSyncResult(gomock.Any(), uid, []models.Entry{remoteRow}, dirty, gomock.AssignableToTypeOf(map[uuid.UUID]struct{}{})).
			DoAndReturn(func(_ context.Context, gotUID int64, updates []models.Entry, gotDirty []models.Entry, applied map[uuid.UUID]struct{}) error {
				require.Equal(t, uid, gotUID)
				require.Len(t, updates, 1)
				assert.Equal(t, remoteRow.UUID, updates[0].UUID)
				require.Len(t, gotDirty, 1)
				assert.Equal(t, id, gotDirty[0].UUID)
				_, ok := applied[id]
				require.True(t, ok, "expected dirty UUID in applied set")
				return nil
			}),
	)

	uc := NewSyncUseCase(local, rem, sess, discardClientLog())
	require.NoError(t, uc.Sync(context.Background()))
}

func TestSyncUseCase_Sync_RemoteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(1)
	dirty := []models.Entry{{UUID: uuid.New()}}

	sess := NewMockSessionReader(ctrl)
	sess.EXPECT().GetSession(gomock.Any()).Return(&models.Session{UserID: &uid}, nil)

	local := NewMockLocalSyncRepo(ctrl)
	rem := NewMockRemoteSyncRepo(ctrl)

	gomock.InOrder(
		local.EXPECT().GetDirtyEntries(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *int64) ([]models.Entry, error) {
			require.NotNil(t, p)
			require.Equal(t, uid, *p)
			return dirty, nil
		}),
		local.EXPECT().GetLastUpdate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *int64) (time.Time, error) {
			require.NotNil(t, p)
			require.Equal(t, uid, *p)
			return time.Time{}, nil
		}),
		rem.EXPECT().Sync(gomock.Any(), dirty, time.Time{}).Return(nil, nil, context.Canceled),
	)

	uc := NewSyncUseCase(local, rem, sess, discardClientLog())
	require.Error(t, uc.Sync(context.Background()), "expected error")
}

func TestSyncUseCase_Sync_NoSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sess := NewMockSessionReader(ctrl)
	sess.EXPECT().GetSession(gomock.Any()).Return(nil, nil)

	local := NewMockLocalSyncRepo(ctrl)
	rem := NewMockRemoteSyncRepo(ctrl)

	uc := NewSyncUseCase(local, rem, sess, discardClientLog())
	require.Error(t, uc.Sync(context.Background()), "expected error")
}

func TestSyncUseCase_Sync_UnappliedLocalStaysDirty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uid := int64(2)
	id := uuid.New()
	dirty := []models.Entry{{UUID: id, Type: models.EntryTypePassword, Version: 1, UpdatedAt: time.Unix(1, 0)}}
	last := time.Unix(0, 0)

	sess := NewMockSessionReader(ctrl)
	sess.EXPECT().GetSession(gomock.Any()).Return(&models.Session{UserID: &uid}, nil)

	local := NewMockLocalSyncRepo(ctrl)
	rem := NewMockRemoteSyncRepo(ctrl)

	applied := []uuid.UUID{}
	gomock.InOrder(
		local.EXPECT().GetDirtyEntries(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *int64) ([]models.Entry, error) {
			require.NotNil(t, p)
			require.Equal(t, uid, *p)
			return dirty, nil
		}),
		local.EXPECT().GetLastUpdate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *int64) (time.Time, error) {
			require.NotNil(t, p)
			require.Equal(t, uid, *p)
			return last, nil
		}),
		rem.EXPECT().Sync(gomock.Any(), dirty, last).Return(nil, applied, nil),
		local.EXPECT().PersistSyncResult(gomock.Any(), uid, gomock.Nil(), dirty, gomock.AssignableToTypeOf(map[uuid.UUID]struct{}{})).
			DoAndReturn(func(_ context.Context, _ int64, _ []models.Entry, _ []models.Entry, appliedMap map[uuid.UUID]struct{}) error {
				_, ok := appliedMap[id]
				require.False(t, ok, "server did not confirm local row; applied set must omit dirty UUID")
				return nil
			}),
	)

	uc := NewSyncUseCase(local, rem, sess, discardClientLog())
	require.NoError(t, uc.Sync(context.Background()))
}
