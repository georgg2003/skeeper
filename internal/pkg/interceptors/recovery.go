package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewUnaryRecoveryInterceptor turns panics into Internal and logs the stack.
func NewUnaryRecoveryInterceptor(l *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				l.ErrorContext(ctx, "grpc unary panic recovered",
					"method", info.FullMethod,
					"recover", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// NewStreamRecoveryInterceptor is the streaming version of [NewUnaryRecoveryInterceptor].
func NewStreamRecoveryInterceptor(l *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				l.ErrorContext(ss.Context(), "grpc stream panic recovered",
					"method", info.FullMethod,
					"recover", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(srv, ss)
	}
}
