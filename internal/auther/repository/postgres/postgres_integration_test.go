//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/integrationtest"
	authermigrate "github.com/georgg2003/skeeper/migrations/auther"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"github.com/georgg2003/skeeper/pkg/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	autherPoolOnce sync.Once
	autherPool     *pgxpool.Pool
	autherPoolErr  error
)

func autherTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("AUTHER_TEST_DSN")
	if dsn == "" {
		t.Fatal("AUTHER_TEST_DSN not set (start auther-db from docker-compose.yaml)")
	}
	autherPoolOnce.Do(func() {
		ctx := context.Background()
		autherPool, autherPoolErr = pgxpool.New(ctx, dsn)
		if autherPoolErr != nil {
			return
		}
		autherPoolErr = integrationtest.GooseMigrate(ctx, autherPool, authermigrate.GooseFiles)
		if autherPoolErr != nil {
			autherPool.Close()
			autherPool = nil
		}
	})
	if autherPoolErr != nil {
		t.Fatal(autherPoolErr)
	}
	return autherPool
}

func truncateAuther(t *testing.T, ctx context.Context, p *pgxpool.Pool) {
	t.Helper()
	_, err := p.Exec(ctx, `TRUNCATE TABLE users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_InsertUser_SelectByEmail(t *testing.T) {
	ctx := context.Background()
	p := autherTestPool(t)
	truncateAuther(t, ctx, p)
	repo := &Repository{pool: p}

	email := "u-" + uuid.NewString() + "@example.com"
	info, err := repo.InsertUser(ctx, models.DBUserCredentials{
		Email:        email,
		PasswordHash: "deadbeef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.ID == 0 || info.Email != email {
		t.Fatalf("%+v", info)
	}

	got, err := repo.SelectUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != info.ID || got.Email != email || got.PasswordHash != "deadbeef" {
		t.Fatalf("%+v", got)
	}
}

func TestIntegration_InsertUser_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	p := autherTestPool(t)
	truncateAuther(t, ctx, p)
	repo := &Repository{pool: p}

	email := "dup-" + uuid.NewString() + "@example.com"
	creds := models.DBUserCredentials{Email: email, PasswordHash: "a"}
	if _, err := repo.InsertUser(ctx, creds); err != nil {
		t.Fatal(err)
	}
	_, err := repo.InsertUser(ctx, creds)
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("want ErrUserExists, got %v", err)
	}
}

func TestIntegration_SelectUserByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	p := autherTestPool(t)
	truncateAuther(t, ctx, p)
	repo := &Repository{pool: p}

	_, err := repo.SelectUserByEmail(ctx, "missing-"+uuid.NewString()+"@example.com")
	if !errors.Is(err, ErrUserNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestIntegration_RefreshTokenRoundTrip(t *testing.T) {
	ctx := context.Background()
	p := autherTestPool(t)
	truncateAuther(t, ctx, p)
	repo := &Repository{pool: p}

	email := "tok-" + uuid.NewString() + "@example.com"
	user, err := repo.InsertUser(ctx, models.DBUserCredentials{Email: email, PasswordHash: "h"})
	if err != nil {
		t.Fatal(err)
	}

	raw := "refresh-secret-" + uuid.NewString()
	hash := utils.HashToken(raw)
	exp := time.Now().Add(time.Hour).UTC()
	if err := repo.InsertRefreshToken(ctx, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: "opaque-refresh", ExpiresAt: exp},
		Hash:  hash,
	}); err != nil {
		t.Fatal(err)
	}

	uid, err := repo.DeleteRefreshTokenAndReturnUser(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	if uid != user.ID {
		t.Fatalf("user id %d want %d", uid, user.ID)
	}

	_, err = repo.DeleteRefreshTokenAndReturnUser(ctx, hash)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("second delete: %v", err)
	}
}
