// Package usecase implements signup, login, and refresh-token rotation on top of Postgres + JWT.
package usecase

//go:generate go tool mockgen -typed -destination=mock_repository_test.go -package=usecase -source=usecase.go Repository

import (
	"context"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/postgres"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

var ErrUserNotExist = errors.New("user not exists")
var ErrInvalidToken = errors.New("refresh token is invalid")
var ErrUserExists = errors.New("user already exists")

// bcryptDummyHash is a valid bcrypt hash used only to normalize login timing when the email is unknown.
var bcryptDummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

type Repository interface {
	InsertUser(context.Context, models.DBUserCredentials) (models.UserInfo, error)
	ReplaceUserRefreshTokens(ctx context.Context, userID int64, pair jwthelper.TokenPair) error
	RotateRefreshToken(ctx context.Context, refreshPlain string, mint func(int64) (jwthelper.TokenPair, error)) (jwthelper.TokenPair, error)
	SelectUserByEmail(context.Context, string) (models.UserInfo, error)
	Close()
}

type UseCase struct {
	repository Repository
	jwtHelper  *jwthelper.JWTHelper
	l          *slog.Logger
	refreshSF  singleflight.Group
}

func (uc *UseCase) CreateUser(ctx context.Context, creds models.UserCredentials) (models.UserInfo, error) {
	if err := creds.Validate(); err != nil {
		return models.UserInfo{}, errors.Wrap(err, "user credentials are invalid")
	}
	hash, err := creds.HashPassword()
	if err != nil {
		return models.UserInfo{}, errors.Wrap(err, "failed to hash password")
	}

	info, err := uc.repository.InsertUser(ctx, models.DBUserCredentials{
		Email:        creds.Email,
		PasswordHash: hash,
	})
	if errors.Is(err, postgres.ErrUserExists) {
		return models.UserInfo{}, ErrUserExists
	}
	return info, err
}

func (uc *UseCase) LoginUser(ctx context.Context, creds models.UserCredentials) (models.LoginResponse, error) {
	if err := creds.ValidateForLogin(); err != nil {
		return models.LoginResponse{}, errors.Wrap(err, "user credentials are invalid")
	}

	user, selErr := uc.repository.SelectUserByEmail(ctx, creds.Email)
	hash := bcryptDummyHash
	if selErr == nil {
		hash = user.PasswordHash
	} else if !errors.Is(selErr, postgres.ErrUserNotExist) {
		return models.LoginResponse{}, selErr
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(creds.Password)); err != nil {
		return models.LoginResponse{}, ErrUserNotExist
	}
	if errors.Is(selErr, postgres.ErrUserNotExist) {
		return models.LoginResponse{}, ErrUserNotExist
	}

	tokenPair, err := uc.jwtHelper.NewTokenPair(user.ID)
	if err != nil {
		return models.LoginResponse{}, errors.Wrap(err, "failed to create a new token pair")
	}
	if err := uc.repository.ReplaceUserRefreshTokens(ctx, user.ID, tokenPair); err != nil {
		return models.LoginResponse{}, errors.Wrap(err, "failed to persist refresh token")
	}

	return models.LoginResponse{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

func (uc *UseCase) RotateToken(ctx context.Context, refreshToken string) (jwthelper.TokenPair, error) {
	v, err, _ := uc.refreshSF.Do(refreshToken, func() (interface{}, error) {
		pair, rotErr := uc.repository.RotateRefreshToken(ctx, refreshToken, uc.jwtHelper.NewTokenPair)
		if errors.Is(rotErr, postgres.ErrInvalidToken) {
			return nil, ErrInvalidToken
		}
		return pair, rotErr
	})
	if err != nil {
		return jwthelper.TokenPair{}, err
	}
	return v.(jwthelper.TokenPair), nil
}

func New(
	l *slog.Logger,
	repo Repository,
	jwtHelper *jwthelper.JWTHelper,
) *UseCase {
	return &UseCase{
		l:          l,
		repository: repo,
		jwtHelper:  jwtHelper,
	}
}
