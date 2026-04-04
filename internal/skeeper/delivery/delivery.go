package delivery

//go:generate go tool mockgen -typed -destination=mock_usecase_test.go -package=delivery -source=delivery.go UseCase

import (
	"context"
	"log/slog"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UseCase is implemented by the skeeper use case layer.
type UseCase interface {
	Sync(context.Context, models.SyncRequest) (models.SyncResponse, error)
}

type skeeperServer struct {
	api.UnimplementedSkeeperServer

	uc UseCase
	l  *slog.Logger
}

func (s *skeeperServer) Sync(
	ctx context.Context,
	req *api.SyncRequest,
) (resp *api.SyncResponse, err error) {
	syncReq, err := models.NewSyncRequestFromProto(req)
	if valErr, ok := errors.AsType[*errors.ValidationError](err); ok {
		return nil, status.Error(codes.InvalidArgument, valErr.Error())
	}
	if err != nil {
		s.l.ErrorContext(ctx, "failed to create new sync request from proto", "err", err)
		return nil, status.Error(codes.Internal, "failed to sync")
	}
	res, err := s.uc.Sync(ctx, syncReq)
	if err != nil {
		s.l.ErrorContext(ctx, "failed to sync", "err", err)
		return nil, status.Error(codes.Internal, "failed to sync")
	}
	return res.ToProto(), nil
}

func New(l *slog.Logger, uc UseCase) api.SkeeperServer {
	return &skeeperServer{l: l, uc: uc}
}
