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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)
	require.NotZero(t, info.ID)
	require.Equal(t, email, info.Email)

	got, err := repo.SelectUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, info.ID, got.ID)
	assert.Equal(t, email, got.Email)
	assert.Equal(t, "deadbeef", string(got.PasswordHash))
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
	_, err := repo.InsertUser(ctx, creds)
	require.NoError(t, err)
	_, err = repo.InsertUser(ctx, creds)
	assert.True(t, errors.Is(err, postgres.ErrUserExists))
}

func TestIntegration_SelectUserByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(
		t,
		ctx,
		integrationtest.AutherTestPool(t),
	)

	_, err := repo.SelectUserByEmail(ctx, "missing-"+uuid.NewString()+"@example.com")
	assert.True(t, errors.Is(err, postgres.ErrUserNotExist))
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
	require.NoError(t, err)

	raw := "refresh-secret-" + uuid.NewString()
	hash := utils.HashToken(raw)
	exp := time.Now().Add(time.Hour).UTC()
	require.NoError(t, repo.InsertRefreshToken(ctx, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: "opaque-refresh", ExpiresAt: exp},
		Hash:  hash,
	}))

	uid, err := repo.DeleteRefreshTokenAndReturnUser(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, user.ID, uid)

	_, err = repo.DeleteRefreshTokenAndReturnUser(ctx, hash)
	assert.True(t, errors.Is(err, postgres.ErrInvalidToken), "second delete: %v", err)
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
	require.Error(t, err, "expected error")
}

func TestIntegration_SelectUserByEmail_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	email := "sel-" + uuid.NewString() + "@example.com"
	_, err := repo.InsertUser(setup, models.DBUserCredentials{Email: email, PasswordHash: []byte("x")})
	require.NoError(t, err)
	canceled, cancel := context.WithCancel(setup)
	cancel()
	_, err = repo.SelectUserByEmail(canceled, email)
	require.Error(t, err, "expected error")
	assert.Contains(t, err.Error(), "failed to select user")
}

func TestIntegration_InsertRefreshToken_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	email := "rt-" + uuid.NewString() + "@example.com"
	user, err := repo.InsertUser(setup, models.DBUserCredentials{Email: email, PasswordHash: []byte("h")})
	require.NoError(t, err)
	canceled, cancel := context.WithCancel(setup)
	cancel()
	err = repo.InsertRefreshToken(canceled, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: "t", ExpiresAt: time.Now().Add(time.Hour)},
		Hash:  "deadbeef",
	})
	require.Error(t, err, "expected error")
	assert.Contains(t, err.Error(), "failed to insert")
}

func TestIntegration_DeleteRefreshTokenAndReturnUser_ContextCanceled(t *testing.T) {
	setup := context.Background()
	repo := newAutherRepository(t, setup, integrationtest.AutherTestPool(t))
	canceled, cancel := context.WithCancel(setup)
	cancel()
	_, err := repo.DeleteRefreshTokenAndReturnUser(canceled, "any-hash")
	require.Error(t, err, "expected error")
	assert.True(t, strings.Contains(err.Error(), "refresh token"))
}

func TestIntegration_RefreshTokenReuseRevokesActiveTokens(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(
		t,
		ctx,
		integrationtest.AutherTestPool(t),
	)
	email := "reuse-" + uuid.NewString() + "@example.com"
	user, err := repo.InsertUser(ctx, models.DBUserCredentials{Email: email, PasswordHash: []byte("h")})
	require.NoError(t, err)

	raw1 := "refresh-one-" + uuid.NewString()
	hash1 := utils.HashToken(raw1)
	raw2 := "refresh-two-" + uuid.NewString()
	hash2 := utils.HashToken(raw2)
	exp := time.Now().Add(time.Hour).UTC()
	require.NoError(t, repo.InsertRefreshToken(ctx, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: "t1", ExpiresAt: exp},
		Hash:  hash1,
	}))
	uid, err := repo.DeleteRefreshTokenAndReturnUser(ctx, hash1)
	require.NoError(t, err)
	assert.Equal(t, user.ID, uid)
	require.NoError(t, repo.InsertRefreshToken(ctx, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: "t2", ExpiresAt: exp},
		Hash:  hash2,
	}))
	_, err = repo.DeleteRefreshTokenAndReturnUser(ctx, hash1)
	assert.True(t, errors.Is(err, postgres.ErrInvalidToken), "reuse stale token: %v", err)
	_, err = repo.DeleteRefreshTokenAndReturnUser(ctx, hash2)
	assert.True(t, errors.Is(err, postgres.ErrInvalidToken), "valid token should be revoked after reuse; got %v", err)
}

func TestIntegration_RotateRefreshToken_RollsBackWhenMintFails(t *testing.T) {
	ctx := context.Background()
	repo := newAutherRepository(t, ctx, integrationtest.AutherTestPool(t))
	email := "rot-" + uuid.NewString() + "@example.com"
	user, err := repo.InsertUser(ctx, models.DBUserCredentials{Email: email, PasswordHash: []byte("h")})
	require.NoError(t, err)
	raw := "plain-refresh-" + uuid.NewString()
	hash := utils.HashToken(raw)
	exp := time.Now().Add(time.Hour).UTC()
	require.NoError(t, repo.InsertRefreshToken(ctx, user.ID, models.RefreshTokenHashed{
		Token: jwthelper.Token{Token: raw, ExpiresAt: exp},
		Hash:  hash,
	}))
	_, err = repo.RotateRefreshToken(ctx, raw, func(int64) (jwthelper.TokenPair, error) {
		return jwthelper.TokenPair{}, errors.New("mint fail")
	})
	require.Error(t, err, "expected mint error")
	newRT := "new-rt-" + uuid.NewString()
	pair, err := repo.RotateRefreshToken(ctx, raw, func(int64) (jwthelper.TokenPair, error) {
		return jwthelper.TokenPair{
			AccessToken:  jwthelper.Token{Token: "at", ExpiresAt: exp},
			RefreshToken: jwthelper.Token{Token: newRT, ExpiresAt: exp},
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, newRT, pair.RefreshToken.Token)
	_, err = repo.RotateRefreshToken(ctx, newRT, func(int64) (jwthelper.TokenPair, error) {
		return jwthelper.TokenPair{
			AccessToken:  jwthelper.Token{Token: "at2", ExpiresAt: exp},
			RefreshToken: jwthelper.Token{Token: "rt3", ExpiresAt: exp},
		}, nil
	})
	require.NoError(t, err, "second rotation with new refresh: %v", err)
	// Replaying the original refresh after it was marked used revokes the whole family (reuse detection).
	_, err = repo.RotateRefreshToken(ctx, raw, func(int64) (jwthelper.TokenPair, error) {
		return jwthelper.TokenPair{}, errors.New("should not run")
	})
	assert.True(t, errors.Is(err, postgres.ErrInvalidToken), "stale refresh reuse: %v", err)
}
