package usecase

import (
	"context"

	"github.com/georgg2003/skeeper/internal/auther/pkg/jwthelper"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/db"
	"github.com/georgg2003/skeeper/pkg/errors"
)

var ErrUserNotExist = errors.New("user not exists")
var ErrInvalidToken = errors.New("refresh token is invalid")

//go:generate mockgen TODO
type UseCase interface {
	CreateUser(context.Context, models.UserCredentials) (models.UserInfo, error)
	LoginUser(context.Context, models.UserCredentials) (models.LoginReponse, error)
	ExchangeToken(context.Context, string) (models.TokenSet, error)
}

type useCase struct {
	repository db.Repository
	jwtHelper  jwthelper.JWTHelper
}

func (uc useCase) CreateUser(ctx context.Context, creds models.UserCredentials) (models.UserInfo, error) {
	if err := creds.Validate(); err != nil {
		return models.UserInfo{}, errors.Wrap(err, "user credentials are invalid")
	}

	creds.HashPassword()

	return uc.repository.CreateUser(ctx, creds)
}

func (uc useCase) insertTokenSet(ctx context.Context, userID int64) (models.TokenSet, error) {
	// generate pair of keys
	tokenSet := models.TokenSet{}

	if err := uc.repository.InsertTokenSet(ctx, userID, tokenSet); err != nil {
		return tokenSet, errors.Wrap(err, "failed to insert token set")
	}
}

func (uc useCase) LoginUser(ctx context.Context, creds models.UserCredentials) (models.LoginReponse, error) {
	user, err := uc.repository.SelectUserByEmail(ctx, creds.Email)
	if errors.As(err, db.ErrUserNotExist) {
		return models.LoginReponse{}, ErrUserNotExist
	}

	tokenSet, err := uc.insertTokenSet(ctx, user.ID)
	if err != nil {
		return models.LoginReponse{}, err
	}

	return models.LoginReponse{
		User:     user,
		TokenSet: tokenSet,
	}, nil
}

func (uc useCase) ExchangeToken(ctx context.Context, refreshToken string) (models.TokenSet, error) {
	accessToken, err := uc.repository.ExchangeToken(ctx, refreshToken)
	if errors.As(err, db.ErrInvalidToken) {
		return models.TokenSet{}, ErrInvalidToken
	}

	// TODO get userID from accessToken JWT or from database
	var userID int64
	newTokenSet, err := uc.insertTokenSet(ctx, userID)
	if err != nil {
		return models.TokenSet{}, err
	}

	return models.TokenSet{
		AccessToken:  accessToken,
		RefreshToken: newTokenSet.RefreshToken,
	}, err
}

func New(jwtHelper jwthelper.JWTHelper) UseCase {
	return &useCase{jwtHelper: jwtHelper}
}
