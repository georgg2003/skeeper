package delivery

import (
	"context"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/auther/pkg/models"
	usecase "github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	//s.uc.CheckUser()

	return nil, nil
}

func (s autherServer) Token(ctx context.Context, req *api.TokenRequest) (*api.TokenResponse, error) {
	//s.uc.GetToken()

	return nil, nil
}

func New(uc usecase.UseCase) api.AutherServer {
	return &autherServer{uc: uc}
}
