package usecase

//go:generate go tool mockgen -typed -destination=mock_repository_test.go -package=usecase -source=usecase.go Repository

import (
	"context"
	"log/slog"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/postgres"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"github.com/georgg2003/skeeper/pkg/utils"
)

var ErrUserNotExist = errors.New("user not exists")
var ErrInvalidToken = errors.New("refresh token is invalid")

type Repository interface {
	InsertUser(context.Context, models.DBUserCredentials) (models.UserInfo, error)
	DeleteRefreshTokenAndReturnUser(context.Context, string) (int64, error)
	SelectUserByEmail(context.Context, string) (models.UserInfo, error)
	InsertRefreshToken(ctx context.Context, userID int64, rt models.RefreshTokenHashed) error
	Close()
}

type UseCase struct {
	repository Repository
	jwtHelper  *jwthelper.JWTHelper
	l          *slog.Logger
}

func (uc UseCase) CreateUser(ctx context.Context, creds models.UserCredentials) (models.UserInfo, error) {
	if err := creds.Validate(); err != nil {
		return models.UserInfo{}, errors.Wrap(err, "user credentials are invalid")
	}
	hash, err := creds.HashPassword()
	if err != nil {
		return models.UserInfo{}, errors.Wrap(err, "failed to hash password")
	}

	return uc.repository.InsertUser(ctx, models.DBUserCredentials{
		Email:        creds.Email,
		PasswordHash: hash,
	})
}

func (uc UseCase) insertTokenSet(ctx context.Context, userID int64) (jwthelper.TokenPair, error) {
	tokenPair, err := uc.jwtHelper.NewTokenPair(userID)
	if err != nil {
		return tokenPair, errors.Wrap(err, "failed to create a new token pair")
	}

	rt := models.RefreshTokenHashed{
		Token: tokenPair.RefreshToken,
		Hash:  utils.HashToken(tokenPair.RefreshToken.Token),
	}

	if err := uc.repository.InsertRefreshToken(ctx, userID, rt); err != nil {
		return jwthelper.TokenPair{}, errors.Wrap(err, "failed to insert refresh token")
	}

	return tokenPair, nil
}

func (uc UseCase) LoginUser(ctx context.Context, creds models.UserCredentials) (models.LoginReponse, error) {
	if err := creds.Validate(); err != nil {
		return models.LoginReponse{}, errors.Wrap(err, "user credentials are invalid")
	}

	user, err := uc.repository.SelectUserByEmail(ctx, creds.Email)
	if errors.Is(err, postgres.ErrUserNotExist) {
		return models.LoginReponse{}, ErrUserNotExist
	}

	if err = creds.CheckPassword(user.PasswordHash); err != nil {
		return models.LoginReponse{}, ErrUserNotExist
	}

	tokenPair, err := uc.insertTokenSet(ctx, user.ID)
	if err != nil {
		return models.LoginReponse{}, err
	}

	return models.LoginReponse{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

func (uc UseCase) RotateToken(ctx context.Context, refreshToken string) (jwthelper.TokenPair, error) {
	userID, err := uc.repository.DeleteRefreshTokenAndReturnUser(ctx, utils.HashToken(refreshToken))
	if errors.Is(err, postgres.ErrInvalidToken) {
		return jwthelper.TokenPair{}, ErrInvalidToken
	}

	return uc.insertTokenSet(ctx, userID)
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
