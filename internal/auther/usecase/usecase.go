package usecase

import (
	"context"
	"log/slog"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/db"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"github.com/georgg2003/skeeper/pkg/utils"
)

var ErrUserNotExist = errors.New("user not exists")
var ErrInvalidToken = errors.New("refresh token is invalid")

//go:generate mockgen TODO
type UseCase interface {
	CreateUser(context.Context, models.UserCredentials) (models.UserInfo, error)
	LoginUser(context.Context, models.UserCredentials) (models.LoginReponse, error)
	RotateToken(ctx context.Context, refreshToken string) (jwthelper.TokenPair, error)
}

type useCase struct {
	repository db.Repository
	jwtHelper  jwthelper.JWTHelper
	l          *slog.Logger
}

func (uc useCase) CreateUser(ctx context.Context, creds models.UserCredentials) (models.UserInfo, error) {
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

func (uc useCase) insertTokenSet(ctx context.Context, userID int64) (jwthelper.TokenPair, error) {
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

	return tokenPair, err
}

func (uc useCase) LoginUser(ctx context.Context, creds models.UserCredentials) (models.LoginReponse, error) {
	user, err := uc.repository.SelectUserByEmail(ctx, creds.Email)
	if errors.As(err, db.ErrUserNotExist) {
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

func (uc useCase) RotateToken(ctx context.Context, refreshToken string) (jwthelper.TokenPair, error) {
	userID, err := uc.repository.DeleteRefreshTokenAndReturnUser(ctx, utils.HashToken(refreshToken))
	if errors.As(err, db.ErrInvalidToken) {
		return jwthelper.TokenPair{}, ErrInvalidToken
	}

	return uc.insertTokenSet(ctx, userID)
}

func New(l *slog.Logger, jwtHelper jwthelper.JWTHelper) UseCase {
	return &useCase{l: l, jwtHelper: jwtHelper}
}
