package delivery

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSkeeperServer_Sync(t *testing.T) {
	entryID := uuid.New()
	syncAt := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name     string
		setup    func(m *MockUseCase)
		req      *api.SyncRequest
		wantCode codes.Code
		wantOK   bool
	}{
		{
			name:  "invalid_proto_uuid",
			setup: func(m *MockUseCase) {},
			req: api.SyncRequest_builder{
				Updates: []*api.Entry{
					api.Entry_builder{
						Uuid:      "not-a-uuid",
						Type:      "T",
						UpdatedAt: timestamppb.New(time.Now()),
					}.Build(),
				},
				LastSyncAt: timestamppb.New(time.Unix(1, 0).UTC()),
			}.Build(),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "usecase_error_internal",
			setup: func(m *MockUseCase) {
				mockUC := m
				mockUC.EXPECT().
					Sync(gomock.Any(), gomock.AssignableToTypeOf(models.SyncRequest{})).
					Return(models.SyncResponse{}, errors.New("db unavailable"))
			},
			req: api.SyncRequest_builder{
				Updates: []*api.Entry{
					api.Entry_builder{
						Uuid:      entryID.String(),
						Type:      "PASSWORD",
						UpdatedAt: timestamppb.New(time.Now()),
					}.Build(),
				},
			}.Build(),
			wantCode: codes.Internal,
		},
		{
			name: "success",
			setup: func(m *MockUseCase) {
				mockUC := m
				mockUC.EXPECT().
					Sync(gomock.Any(), gomock.AssignableToTypeOf(models.SyncRequest{})).
					DoAndReturn(func(_ context.Context, got models.SyncRequest) (models.SyncResponse, error) {
						if len(got.Updates) != 1 || got.Updates[0].UUID != entryID {
							t.Fatalf("unexpected request: %+v", got)
						}
						return models.SyncResponse{
							CurrentSyncAt: syncAt,
							Updates:       []models.Entry{{UUID: entryID, Type: "TEXT", Version: 2, UpdatedAt: syncAt}},
						}, nil
					})
			},
			req: api.SyncRequest_builder{
				Updates: []*api.Entry{
					api.Entry_builder{
						Uuid:      entryID.String(),
						Type:      "TEXT",
						Version:   1,
						UpdatedAt: timestamppb.New(time.Unix(10, 0).UTC()),
					}.Build(),
				},
				LastSyncAt: timestamppb.New(time.Unix(5, 0).UTC()),
			}.Build(),
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUC := NewMockUseCase(ctrl)
			tt.setup(mockUC)
			srv := New(testLogger(), mockUC)
			resp, err := srv.Sync(context.Background(), tt.req)
			if tt.wantOK {
				if err != nil {
					t.Fatal(err)
				}
				if !resp.GetCurrentSyncAt().AsTime().Equal(syncAt) {
					t.Fatal("current_sync_at")
				}
				if len(resp.GetUpdates()) != 1 || resp.GetUpdates()[0].GetUuid() != entryID.String() {
					t.Fatalf("updates %+v", resp.GetUpdates())
				}
				return
			}
			st, ok := status.FromError(err)
			if !ok || st.Code() != tt.wantCode {
				t.Fatalf("got %v", err)
			}
		})
	}
}
