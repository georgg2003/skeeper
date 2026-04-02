package skeeper

import (
	"context"

	"github.com/georgg2003/skeeper/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type TokenProvider interface {
	GetAccessToken(ctx context.Context) (string, error)
}

type SkeeperClient struct {
	conn *grpc.ClientConn
	api  api.SkeeperClient
}

func newAuthInterceptor(provider TokenProvider) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		token, err := provider.GetAccessToken(ctx)
		if err != nil {
			return err
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func NewSkeeperClient(addr string, provider TokenProvider) (*SkeeperClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(newAuthInterceptor(provider)),
	)
	if err != nil {
		return nil, err
	}

	return &SkeeperClient{
		conn: conn,
		api:  api.NewSkeeperClient(conn),
	}, nil
}
