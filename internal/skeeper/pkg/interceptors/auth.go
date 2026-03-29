package interceptors

import (
	"context"
	"strings"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func authorize(ctx context.Context, jwt *jwthelper.JWTHelper) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	parts := strings.Split(values[0], " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	token := parts[1]
	claims, err := jwt.ValidateToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return contextlib.SetUserID(ctx, claims.UserID), nil
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func NewAuthInterceptor(jwt *jwthelper.JWTHelper) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		newCtx, err := authorize(ctx, jwt)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

func NewStreamAuthInterceptor(jwt *jwthelper.JWTHelper) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		newCtx, err := authorize(ss.Context(), jwt)
		if err != nil {
			return err
		}

		wrapped := &wrappedStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}
