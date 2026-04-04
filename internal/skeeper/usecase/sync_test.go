package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
)

type stubRepo struct {
	upserted  [][]models.Entry
	lastSeen  time.Time
	ret       []models.Entry
	upsertErr error
	getErr    error
}

func (s *stubRepo) GetVaultCrypto(context.Context, int64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (s *stubRepo) PutVaultCrypto(context.Context, int64, []byte, []byte) error {
	return nil
}

func (s *stubRepo) UpsertEntries(ctx context.Context, userID int64, entries []models.Entry) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, entries)
	return nil
}

func (s *stubRepo) GetUpdatedAfter(ctx context.Context, userID int64, lastSync time.Time) ([]models.Entry, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	s.lastSeen = lastSync
	return s.ret, nil
}

func testSyncLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestUseCase_Sync(t *testing.T) {
	id := uuid.New()
	baseLast := time.Unix(100, 0).UTC()

	tests := []struct {
		name    string
		repo    *stubRepo
		req     models.SyncRequest
		userID  int64
		wantErr bool
		check   func(t *testing.T, repo *stubRepo, out models.SyncResponse)
	}{
		{
			name: "upserts_then_fetches",
			repo: &stubRepo{
				ret: []models.Entry{{UUID: id, Type: "TEXT", Version: 2, UpdatedAt: time.Unix(300, 0).UTC()}},
			},
			req: models.SyncRequest{
				LastSyncAt: baseLast,
				Updates:    []models.Entry{{UUID: uuid.New(), Type: "PASSWORD", Version: 1, UpdatedAt: time.Unix(200, 0).UTC()}},
			},
			userID: 42,
			check: func(t *testing.T, repo *stubRepo, out models.SyncResponse) {
				if len(repo.upserted) != 1 || len(repo.upserted[0]) != 1 {
					t.Fatalf("upsert %+v", repo.upserted)
				}
				if !repo.lastSeen.Equal(baseLast) {
					t.Fatalf("last sync %v vs %v", repo.lastSeen, baseLast)
				}
				if len(out.Updates) != 1 || out.Updates[0].UUID != id {
					t.Fatalf("response %+v", out.Updates)
				}
			},
		},
		{
			name:   "no_client_updates_skips_upsert",
			repo:   &stubRepo{},
			req:    models.SyncRequest{LastSyncAt: time.Unix(1, 0).UTC()},
			userID: 1,
			check: func(t *testing.T, repo *stubRepo, out models.SyncResponse) {
				if len(repo.upserted) != 0 {
					t.Fatal("unexpected upsert")
				}
				if len(out.Updates) != 0 {
					t.Fatalf("expected no updates, got %d", len(out.Updates))
				}
			},
		},
		{
			name:    "upsert_error",
			repo:    &stubRepo{upsertErr: context.Canceled},
			req:     models.SyncRequest{Updates: []models.Entry{{UUID: id}}},
			userID:  1,
			wantErr: true,
		},
		{
			name:    "get_error",
			repo:    &stubRepo{getErr: context.Canceled},
			req:     models.SyncRequest{LastSyncAt: time.Unix(1, 0).UTC()},
			userID:  1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(testSyncLogger(), tt.repo, nil)
			ctx := contextlib.SetUserID(context.Background(), tt.userID)
			out, err := uc.Sync(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, tt.repo, out)
			}
		})
	}
}
