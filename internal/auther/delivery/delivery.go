package delivery

import (
	"context"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	usecase "github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type autherServer struct {
	api.UnimplementedAutherServer

	uc usecase.UseCase
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
	if err != nil {
		// TODO add logs
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

	if err != nil {
		// TODO add logs
		return nil, status.Error(codes.Internal, "failed to login user")
	}

	return api.LoginResponse_builder{
		User: api.User_builder{
			Email: &resp.User.Email,
			Id:    &resp.User.ID,
		}.Build(),
		RefreshToken: api.Token_builder{
			Data:      &resp.RefreshToken.Data,
			ExpiresAt: timestamppb.New(resp.RefreshToken.ExpiresAt),
		}.Build(),
		AccessToken: api.Token_builder{
			Data:      &resp.AccessToken.Data,
			ExpiresAt: timestamppb.New(resp.AccessToken.ExpiresAt),
		}.Build(),
	}.Build(), nil
}

func (s autherServer) ExchangeToken(ctx context.Context, req *api.ExchangeTokenRequest) (*api.ExchangeTokenResponse, error) {
	refreshToken := req.GetRefreshToken()

	resp, err := s.uc.ExchangeToken(ctx, refreshToken)

	if err != nil {
		// TODO add logs
		return nil, status.Error(codes.Internal, "failed to login user")
	}

	return api.ExchangeTokenResponse_builder{
		RefreshToken: api.Token_builder{
			Data:      &resp.RefreshToken.Data,
			ExpiresAt: timestamppb.New(resp.RefreshToken.ExpiresAt),
		}.Build(),
		AccessToken: api.Token_builder{
			Data:      &resp.AccessToken.Data,
			ExpiresAt: timestamppb.New(resp.AccessToken.ExpiresAt),
		}.Build(),
	}.Build(), nil
}

func New(uc usecase.UseCase) api.AutherServer {
	return &autherServer{uc: uc}
}
