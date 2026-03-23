package db

import (
	"context"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
)

//go:generate TODO
type Repository interface {
	CreateUser(context.Context, models.UserCredentials) (models.UserInfo, error)
}
