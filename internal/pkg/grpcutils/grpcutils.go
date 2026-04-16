package grpcutils

import (
	"context"

	"google.golang.org/grpc"
)

type WrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *WrappedStream) Context() context.Context {
	return w.ctx
}

func NewWrappedStream(ctx context.Context, ss grpc.ServerStream) *WrappedStream {
	return &WrappedStream{ctx: ctx, ServerStream: ss}
}
