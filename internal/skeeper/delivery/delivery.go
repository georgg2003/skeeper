package delivery

//go:generate go tool mockgen -typed -destination=mock_usecase_test.go -package=delivery -source=delivery.go UseCase

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/models"
	"github.com/georgg2003/skeeper/internal/skeeper/pkg/vaulterror"
	pkgerrors "github.com/georgg2003/skeeper/pkg/errors"
)

// UseCase is implemented by the skeeper use case layer.
type UseCase interface {
	Sync(context.Context, models.SyncRequest) (models.SyncResponse, error)
	GetVaultCrypto(context.Context) ([]byte, []byte, error)
	PutVaultCrypto(context.Context, []byte, []byte) error
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
	if valErr, ok := pkgerrors.AsType[*pkgerrors.ValidationError](err); ok {
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

func (s *skeeperServer) GetVaultCrypto(
	ctx context.Context,
	_ *api.GetVaultCryptoRequest,
) (*api.GetVaultCryptoResponse, error) {
	salt, verifier, err := s.uc.GetVaultCrypto(ctx)
	if errors.Is(err, vaulterror.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "vault crypto not found")
	}
	if err != nil {
		s.l.ErrorContext(ctx, "get vault crypto", "err", err)
		return nil, status.Error(codes.Internal, "failed to get vault crypto")
	}
	return api.GetVaultCryptoResponse_builder{
		Vault: api.VaultCrypto_builder{
			KdfSalt:        salt,
			MasterVerifier: verifier,
		}.Build(),
	}.Build(), nil
}

func (s *skeeperServer) PutVaultCrypto(
	ctx context.Context,
	req *api.PutVaultCryptoRequest,
) (*api.PutVaultCryptoResponse, error) {
	v := req.GetVault()
	if v == nil {
		return nil, status.Error(codes.InvalidArgument, "missing vault")
	}
	err := s.uc.PutVaultCrypto(ctx, v.GetKdfSalt(), v.GetMasterVerifier())
	if valErr, ok := pkgerrors.AsType[*pkgerrors.ValidationError](err); ok {
		return nil, status.Error(codes.InvalidArgument, valErr.Error())
	}
	if errors.Is(err, vaulterror.ErrConflict) {
		return nil, status.Error(codes.AlreadyExists, "vault already initialized with different credentials")
	}
	if err != nil {
		s.l.ErrorContext(ctx, "put vault crypto", "err", err)
		return nil, status.Error(codes.Internal, "failed to put vault crypto")
	}
	return &api.PutVaultCryptoResponse{}, nil
}

func New(l *slog.Logger, uc UseCase) api.SkeeperServer {
	return &skeeperServer{l: l, uc: uc}
}
