package usecase

import (
	"log/slog"

	"github.com/georgg2003/skeeper/internal/skeeper/repository/db"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

var ErrInvalidToken = errors.New("refresh token is invalid")

//go:generate mockgen TODO
type UseCase interface{}

type useCase struct {
	repository db.Repository
	jwtHelper  jwthelper.JWTHelper
	l          *slog.Logger
}

func New(
	l *slog.Logger,
	repo db.Repository,
	jwtHelper jwthelper.JWTHelper,
) UseCase {
	return &useCase{
		l:          l,
		repository: repo,
		jwtHelper:  jwtHelper,
	}
}
