package delivery

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSync_InvalidProtoEntry_UUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	// No EXPECT: Sync must not be called when proto validation fails.

	srv := New(testLogger(), mockUC)
	badID := "not-a-uuid"
	req := api.SyncRequest_builder{
		Updates: []*api.Entry{
			api.Entry_builder{
				Uuid:      badID,
				Type:      "T",
				UpdatedAt: timestamppb.New(time.Now()),
			}.Build(),
		},
		LastSyncAt: timestamppb.New(time.Unix(1, 0).UTC()),
	}.Build()
	_, err := srv.Sync(context.Background(), req)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestSync_UsecaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	entryID := uuid.New()
	mockUC.EXPECT().
		Sync(gomock.Any(), gomock.AssignableToTypeOf(models.SyncRequest{})).
		DoAndReturn(func(_ context.Context, got models.SyncRequest) (models.SyncResponse, error) {
			if len(got.Updates) != 1 || got.Updates[0].UUID.String() != entryID.String() {
				t.Fatalf("unexpected request: %+v", got)
			}
			return models.SyncResponse{}, errors.New("db unavailable")
		})

	srv := New(testLogger(), mockUC)
	id := entryID.String()
	req := api.SyncRequest_builder{
		Updates: []*api.Entry{
			api.Entry_builder{
				Uuid:      id,
				Type:      "PASSWORD",
				UpdatedAt: timestamppb.New(time.Now()),
			}.Build(),
		},
	}.Build()
	_, err := srv.Sync(context.Background(), req)
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestSync_Success(t *testing.T) {
	syncAt := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	entryID := uuid.New()
	ctrl := gomock.NewController(t)
	mockUC := NewMockUseCase(ctrl)
	mockUC.EXPECT().
		Sync(gomock.Any(), gomock.AssignableToTypeOf(models.SyncRequest{})).
		DoAndReturn(func(_ context.Context, got models.SyncRequest) (models.SyncResponse, error) {
			if len(got.Updates) != 1 || got.Updates[0].UUID != entryID {
				t.Fatalf("unexpected request: %+v", got)
			}
			return models.SyncResponse{
				CurrentSyncAt: syncAt,
				Updates: []models.Entry{
					{UUID: entryID, Type: "TEXT", Version: 2, UpdatedAt: syncAt},
				},
			}, nil
		})

	srv := New(testLogger(), mockUC)
	req := api.SyncRequest_builder{
		Updates: []*api.Entry{
			api.Entry_builder{
				Uuid:      entryID.String(),
				Type:      "TEXT",
				Version:   1,
				UpdatedAt: timestamppb.New(time.Unix(10, 0).UTC()),
			}.Build(),
		},
		LastSyncAt: timestamppb.New(time.Unix(5, 0).UTC()),
	}.Build()
	resp, err := srv.Sync(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetCurrentSyncAt().AsTime().Equal(syncAt) {
		t.Fatal("current_sync_at")
	}
	if len(resp.GetUpdates()) != 1 || resp.GetUpdates()[0].GetUuid() != entryID.String() {
		t.Fatalf("updates %+v", resp.GetUpdates())
	}
}
