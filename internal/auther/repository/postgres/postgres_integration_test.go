// Repository integration tests: Postgres via testcontainers-go (Docker required).

package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/postgres"
	"github.com/georgg2003/skeeper/internal/integrationtest"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"github.com/georgg2003/skeeper/pkg/utils"
)

func newAutherRepository(t *testing.T, ctx context.Context, p *pgxpool.Pool) *postgres.Repository {
	t.Helper()
	integrationtest.TruncateAuther(t, ctx, p)
	return postgres.NewFromPool(p)
}

func TestIntegration_InsertUser_SelectByEmail(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(
		t,
		ctx,
		integrationtest.AutherTestPool(t),
	)

	email := "u-" + uuid.NewString() + "@example.com"
	info, err := repo.InsertUser(ctx, models.DBUserCredentials{
		Email:        email,
		PasswordHash: []byte("deadbeef"),
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
	if got.ID != info.ID || got.Email != email || string(got.PasswordHash) != "deadbeef" {
		t.Fatalf("%+v", got)
	}
}

func TestIntegration_InsertUser_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(
		t,
		ctx,
		integrationtest.AutherTestPool(t),
	)

	email := "dup-" + uuid.NewString() + "@example.com"
	creds := models.DBUserCredentials{Email: email, PasswordHash: []byte("a")}
	if _, err := repo.InsertUser(ctx, creds); err != nil {
		t.Fatal(err)
	}
	_, err := repo.InsertUser(ctx, creds)
	if !errors.Is(err, postgres.ErrUserExists) {
		t.Fatalf("want ErrUserExists, got %v", err)
	}
}

func TestIntegration_SelectUserByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(
		t,
		ctx,
		integrationtest.AutherTestPool(t),
	)

	_, err := repo.SelectUserByEmail(ctx, "missing-"+uuid.NewString()+"@example.com")
	if !errors.Is(err, postgres.ErrUserNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestIntegration_RefreshTokenRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(
		t,
		ctx,
		integrationtest.AutherTestPool(t),
	)
	email := "tok-" + uuid.NewString() + "@example.com"
	user, err := repo.InsertUser(ctx, models.DBUserCredentials{Email: email, PasswordHash: []byte("h")})
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
	if !errors.Is(err, postgres.ErrInvalidToken) {
		t.Fatalf("second delete: %v", err)
	}
}

func TestIntegration_InsertUser_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	canceled, cancel := context.WithCancel(setup)
	cancel()
	_, err := repo.InsertUser(canceled, models.DBUserCredentials{
		Email:        "ins-" + uuid.NewString() + "@example.com",
		PasswordHash: []byte("h"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIntegration_SelectUserByEmail_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	email := "sel-" + uuid.NewString() + "@example.com"
	if _, err := repo.InsertUser(setup, models.DBUserCredentials{Email: email, PasswordHash: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(setup)
	cancel()
	_, err := repo.SelectUserByEmail(canceled, email)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to select user") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestIntegration_InsertRefreshToken_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	email := "rt-" + uuid.NewString() + "@example.com"
	user, err := repo.InsertUser(setup, models.DBUserCredentials{Email: email, PasswordHash: []byte("h")})
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(setup)
	cancel()
	err = repo.InsertRefreshToken(canceled, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: "t", ExpiresAt: time.Now().Add(time.Hour)},
		Hash:  "deadbeef",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to insert new refresh token") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestIntegration_DeleteRefreshTokenAndReturnUser_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	canceled, cancel := context.WithCancel(setup)
	cancel()
	_, err := repo.DeleteRefreshTokenAndReturnUser(canceled, "any-hash")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to delete token") {
		t.Fatalf("unexpected: %v", err)
	}
}
