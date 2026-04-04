// Package auther implements a gRPC client for the Auther authentication service.
package auther

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/pkg/errors"
)

// AutherClient wraps the generated gRPC client with domain-oriented helpers.
type AutherClient struct {
	conn *grpc.ClientConn
	api  api.AutherClient
}

// CreateUser registers a new account on the Auther service.
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

// Login returns JWT session material after successful password verification.
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

// Refresh exchanges a refresh token for a new access (and refresh) token pair.
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

// NewAutherClient dials the Auther gRPC endpoint (address like "host:port").
func NewAutherClient(addr string) (*AutherClient, error) {
	if addr == "" {
		return nil, fmt.Errorf("auther address is required")
	}
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &AutherClient{
		conn: conn,
		api:  api.NewAutherClient(conn),
	}, nil
}
