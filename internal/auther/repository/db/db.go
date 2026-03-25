package db

import (
	"context"
	"time"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

var ErrUserExists = errors.New("user already exists")
var ErrInvalidToken = errors.New("invalid token")
var ErrUserNotExist = errors.New("user does not exist")

//go:generate TODO
type Repository interface {
	CreateUser(context.Context, models.UserCredentials) (models.UserInfo, error)
	DeleteRefreshToken(context.Context, string) (time.Time, error)
	SelectUserByEmail(context.Context, string) (models.UserInfo, error)
	InsertRefreshToken(context.Context, int64, models.Token) error
}
