package delivery

import (
	"log/slog"

	"github.com/georgg2003/skeeper/api"
	usecase "github.com/georgg2003/skeeper/internal/skeeper/usecase"
)

type skeeperServer struct {
	api.UnimplementedSkeeperServer

	uc usecase.UseCase
	l  *slog.Logger
}

func New(l *slog.Logger, uc usecase.UseCase) api.SkeeperServer {
	return &skeeperServer{l: l, uc: uc}
}
