// Package auther is the gRPC client for register, login, and token refresh.
package auther

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

type AutherClient struct {
	conn *grpc.ClientConn
	api  api.AutherClient
}

func (c *AutherClient) CreateUser(ctx context.Context, email, password string) error {
	_, err := c.api.CreateUser(ctx, api.CreateUserRequest_builder{
		Email:    &email,
		Password: &password,
	}.Build())
	if err != nil {
		return errors.Wrap(err, "create user")
	}
	return nil
}

func (c *AutherClient) Login(ctx context.Context, email, password string) (*models.Session, error) {
	resp, err := c.api.Login(ctx, api.LoginRequest_builder{
		Email:    &email,
		Password: &password,
	}.Build())
	if err != nil {
		return nil, errors.Wrap(err, "login")
	}
	at := resp.GetAccessToken()
	rt := resp.GetRefreshToken()
	if at == nil || rt == nil || at.GetExpiresAt() == nil || rt.GetExpiresAt() == nil {
		return nil, errors.New("invalid login response: missing tokens")
	}
	return &models.Session{
		AccessToken:      at.GetToken(),
		RefreshToken:     rt.GetToken(),
		ExpiresAt:        at.GetExpiresAt().AsTime(),
		RefreshExpiresAt: rt.GetExpiresAt().AsTime(),
	}, nil
}

func (c *AutherClient) Refresh(ctx context.Context, refreshToken string) (*models.Session, error) {
	resp, err := c.api.ExchangeToken(ctx, api.ExchangeTokenRequest_builder{
		RefreshToken: &refreshToken,
	}.Build())
	if err != nil {
		return nil, errors.Wrap(err, "exchange token")
	}
	at := resp.GetAccessToken()
	rt := resp.GetRefreshToken()
	if at == nil || rt == nil || at.GetExpiresAt() == nil || rt.GetExpiresAt() == nil {
		return nil, errors.New("invalid exchange response: missing tokens")
	}
	return &models.Session{
		AccessToken:      at.GetToken(),
		RefreshToken:     rt.GetToken(),
		ExpiresAt:        at.GetExpiresAt().AsTime(),
		RefreshExpiresAt: rt.GetExpiresAt().AsTime(),
	}, nil
}

// NewAutherClient connects to addr (host:port). dialOpts must include grpcclient.DialOptions output.
func NewAutherClient(addr string, dialOpts ...grpc.DialOption) (*AutherClient, error) {
	if addr == "" {
		return nil, fmt.Errorf("auther address is required")
	}
	if len(dialOpts) == 0 {
		return nil, fmt.Errorf("dial options are required (use grpcclient.DialOptions)")
	}
	opts := append([]grpc.DialOption(nil), dialOpts...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &AutherClient{
		conn: conn,
		api:  api.NewAutherClient(conn),
	}, nil
}
