package usecase

import (
	"context"

	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	"github.com/georgg2003/skeeper/internal/auther/repository/db"
	"github.com/georgg2003/skeeper/pkg/errors"
)

//go:generate mockgen TODO
type UseCase interface {
	CreateUser(context.Context, models.UserCredentials) (models.UserInfo, error)
	Login(context.Context)
	Token(context.Context)
}

type useCase struct {
	repository db.Repository
}

func (uc useCase) CreateUser(ctx context.Context, creds models.UserCredentials) (models.UserInfo, error) {
	if err := creds.Validate(); err != nil {
		return models.UserInfo{}, errors.Wrap(err, "user credentials are invalid")
	}

	return uc.repository.CreateUser(ctx, creds)
}

func (uc useCase) Login(context.Context) {

}

func (uc useCase) Token(context.Context) {

}

func New() UseCase {
	return &useCase{}
}
