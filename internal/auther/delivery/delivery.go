// Package delivery is the gRPC layer for Auther: register, login, token exchange.
package delivery

//go:generate go tool mockgen -typed -destination=mock_usecase_test.go -package=delivery -source=delivery.go UseCase

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	usecase "github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

// UseCase is the Auther business API exposed over gRPC.
type UseCase interface {
	CreateUser(context.Context, models.UserCredentials) (models.UserInfo, error)
	LoginUser(context.Context, models.UserCredentials) (models.LoginResponse, error)
	RotateToken(ctx context.Context, refreshToken string) (jwthelper.TokenPair, error)
}

type autherServer struct {
	api.UnimplementedAutherServer

	uc UseCase
	l  *slog.Logger
}

func (s autherServer) CreateUser(ctx context.Context, req *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	email := req.GetEmail()
	password := req.GetPassword()

	info, err := s.uc.CreateUser(ctx, models.UserCredentials{
		Email:    email,
		Password: password,
	})

	if valErr, ok := errors.AsType[*errors.ValidationError](err); ok {
		return nil, status.Error(codes.InvalidArgument, valErr.Error())
	}
	if errors.Is(err, usecase.ErrUserExists) {
		return nil, status.Error(codes.InvalidArgument, "unable to register with the given credentials")
	}
	if err != nil {
		s.l.ErrorContext(ctx, "failed to create user", "err", err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return api.CreateUserResponse_builder{
		User: api.User_builder{
			Email: &info.Email,
			Id:    &info.ID,
		}.Build(),
	}.Build(), nil
}

func (s autherServer) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	email := req.GetEmail()
	password := req.GetPassword()

	resp, err := s.uc.LoginUser(ctx, models.UserCredentials{
		Email:    email,
		Password: password,
	})

	if valErr, ok := errors.AsType[*errors.ValidationError](err); ok {
		return nil, status.Error(codes.InvalidArgument, valErr.Error())
	}
	if errors.Is(err, usecase.ErrUserNotExist) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}
	if err != nil {
		s.l.ErrorContext(ctx, "failed to login user", "err", err)
		return nil, status.Error(codes.Internal, "failed to login user")
	}

	return api.LoginResponse_builder{
		User: api.User_builder{
			Email: &resp.User.Email,
			Id:    &resp.User.ID,
		}.Build(),
		RefreshToken: api.Token_builder{
			Token:     &resp.TokenPair.RefreshToken.Token,
			ExpiresAt: timestamppb.New(resp.TokenPair.RefreshToken.ExpiresAt),
		}.Build(),
		AccessToken: api.Token_builder{
			Token:     &resp.TokenPair.AccessToken.Token,
			ExpiresAt: timestamppb.New(resp.TokenPair.AccessToken.ExpiresAt),
		}.Build(),
	}.Build(), nil
}

func (s autherServer) ExchangeToken(ctx context.Context, req *api.ExchangeTokenRequest) (*api.ExchangeTokenResponse, error) {
	refreshToken := req.GetRefreshToken()

	resp, err := s.uc.RotateToken(ctx, refreshToken)

	if errors.Is(err, usecase.ErrInvalidToken) {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	if err != nil {
		s.l.ErrorContext(ctx, "failed to exchange token", "err", err)
		return nil, status.Error(codes.Internal, "failed to exchange token")
	}

	return api.ExchangeTokenResponse_builder{
		RefreshToken: api.Token_builder{
			Token:     &resp.RefreshToken.Token,
			ExpiresAt: timestamppb.New(resp.RefreshToken.ExpiresAt),
		}.Build(),
		AccessToken: api.Token_builder{
			Token:     &resp.AccessToken.Token,
			ExpiresAt: timestamppb.New(resp.AccessToken.ExpiresAt),
		}.Build(),
	}.Build(), nil
}

// New returns a gRPC server implementation backed by uc.
func New(l *slog.Logger, uc UseCase) api.AutherServer {
	return &autherServer{l: l, uc: uc}
}
