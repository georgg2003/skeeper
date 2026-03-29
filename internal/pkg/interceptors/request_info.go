package interceptors

import (
	"context"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func getMetadataValue(md metadata.MD, keys ...string) string {
	for _, key := range keys {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func fillRequestInfo(ctx context.Context, fullMethod string) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	p, _ := peer.FromContext(ctx)

	info := contextlib.RequestInfo{
		Method: fullMethod,
		Path:   fullMethod,
	}

	if id := getMetadataValue(md, "x-request-id", "request-id"); id != "" {
		info.RequestID = id
	} else {
		info.RequestID = uuid.New().String()
	}

	if p != nil {
		info.RemoteIP = p.Addr.String()
	}

	info.UserAgent = getMetadataValue(md, "user-agent")
	info.Host = getMetadataValue(md, ":authority", "host")

	return contextlib.SetRequestInfo(ctx, info)
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func NewRequestInfoInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		newCtx := fillRequestInfo(ctx, info.FullMethod)
		return handler(newCtx, req)
	}
}

func NewStreamRequestInfoInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		newCtx := fillRequestInfo(ss.Context(), info.FullMethod)
		wrapped := &wrappedStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}
