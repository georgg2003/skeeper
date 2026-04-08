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
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/vaulterror"
	skeeperusecase "github.com/georgg2003/skeeper/internal/skeeper/usecase"
	pkgerrors "github.com/georgg2003/skeeper/pkg/errors"
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
			name: "unauthenticated",
			setup: func(m *MockUseCase) {
				mockUC := m
				mockUC.EXPECT().
					Sync(gomock.Any(), gomock.AssignableToTypeOf(models.SyncRequest{})).
					Return(models.SyncResponse{}, skeeperusecase.ErrUnauthenticated)
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
			wantCode: codes.Unauthenticated,
		},
		{
			name: "usecase_error_internal",
			setup: func(m *MockUseCase) {
				mockUC := m
				mockUC.EXPECT().
					Sync(gomock.Any(), gomock.AssignableToTypeOf(models.SyncRequest{})).
					Return(models.SyncResponse{}, pkgerrors.New("db unavailable"))
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
							CurrentSyncAt:      syncAt,
							Updates:            []models.Entry{{UUID: entryID, Type: "TEXT", Version: 2, UpdatedAt: syncAt}},
							AppliedUpdateUUIDs: []uuid.UUID{entryID},
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
				applied := resp.GetAppliedUpdateUuids()
				if len(applied) != 1 || applied[0] != entryID.String() {
					t.Fatalf("applied_update_uuids %+v", applied)
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

func TestSkeeperServer_GetVaultCrypto(t *testing.T) {
	salt := []byte{1, 2, 3}
	ver := make([]byte, 32)

	tests := []struct {
		name     string
		setup    func(m *MockUseCase)
		wantCode codes.Code
		wantOK   bool
	}{
		{
			name: "not_found",
			setup: func(m *MockUseCase) {
				m.EXPECT().GetVaultCrypto(gomock.Any()).Return(nil, nil, vaulterror.ErrNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "unauthenticated",
			setup: func(m *MockUseCase) {
				m.EXPECT().GetVaultCrypto(gomock.Any()).Return(nil, nil, skeeperusecase.ErrUnauthenticated)
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "success",
			setup: func(m *MockUseCase) {
				m.EXPECT().GetVaultCrypto(gomock.Any()).Return(salt, ver, nil)
			},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUC := NewMockUseCase(ctrl)
			tt.setup(mockUC)
			srv := New(testLogger(), mockUC)
			resp, err := srv.GetVaultCrypto(context.Background(), api.GetVaultCryptoRequest_builder{}.Build())
			if tt.wantOK {
				if err != nil {
					t.Fatal(err)
				}
				if string(resp.GetVault().GetKdfSalt()) != string(salt) {
					t.Fatal("salt mismatch")
				}
				return
			}
			st, _ := status.FromError(err)
			if st.Code() != tt.wantCode {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestSkeeperServer_PutVaultCrypto(t *testing.T) {
	salt := make([]byte, 16)
	ver := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i)
	}
	for i := range ver {
		ver[i] = byte(i + 1)
	}

	tests := []struct {
		name     string
		setup    func(m *MockUseCase)
		req      *api.PutVaultCryptoRequest
		wantCode codes.Code
		wantOK   bool
	}{
		{
			name:     "missing_vault",
			setup:    func(m *MockUseCase) {},
			req:      api.PutVaultCryptoRequest_builder{}.Build(),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "unauthenticated",
			setup: func(m *MockUseCase) {
				m.EXPECT().PutVaultCrypto(gomock.Any(), salt, ver).Return(skeeperusecase.ErrUnauthenticated)
			},
			req: api.PutVaultCryptoRequest_builder{
				Vault: api.VaultCrypto_builder{KdfSalt: salt, MasterVerifier: ver}.Build(),
			}.Build(),
			wantCode: codes.Unauthenticated,
		},
		{
			name: "conflict",
			setup: func(m *MockUseCase) {
				m.EXPECT().PutVaultCrypto(gomock.Any(), salt, ver).Return(vaulterror.ErrConflict)
			},
			req: api.PutVaultCryptoRequest_builder{
				Vault: api.VaultCrypto_builder{KdfSalt: salt, MasterVerifier: ver}.Build(),
			}.Build(),
			wantCode: codes.AlreadyExists,
		},
		{
			name: "success",
			setup: func(m *MockUseCase) {
				m.EXPECT().PutVaultCrypto(gomock.Any(), salt, ver).Return(nil)
			},
			req: api.PutVaultCryptoRequest_builder{
				Vault: api.VaultCrypto_builder{KdfSalt: salt, MasterVerifier: ver}.Build(),
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
			_, err := srv.PutVaultCrypto(context.Background(), tt.req)
			if tt.wantOK {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			st, _ := status.FromError(err)
			if st.Code() != tt.wantCode {
				t.Fatalf("got %v", err)
			}
		})
	}
}
