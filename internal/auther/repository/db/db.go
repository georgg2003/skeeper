package db

import (
	"context"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

var ErrUserExists = errors.New("user already exists")
var ErrInvalidToken = errors.New("invalid token")
var ErrUserNotExist = errors.New("user does not exist")

//go:generate TODO
type Repository interface {
	InsertUser(context.Context, models.DBUserCredentials) (models.UserInfo, error)
	DeleteRefreshTokenAndReturnUser(context.Context, string) (int64, error)
	SelectUserByEmail(context.Context, string) (models.UserInfo, error)
	InsertRefreshToken(ctx context.Context, userID int64, rt models.RefreshTokenHashed) error
}
