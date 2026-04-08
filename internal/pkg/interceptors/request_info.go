package interceptors

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
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

// NewRequestInfoInterceptor fills request id / peer info on the context and logs start + duration.
func NewRequestInfoInterceptor(l *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		newCtx := fillRequestInfo(ctx, info.FullMethod)
		t0 := time.Now()
		l.InfoContext(newCtx, "request started")
		res, err := handler(newCtx, req)
		l.InfoContext(
			newCtx,
			"request finished",
			"duration", time.Since(t0),
			"err", err,
		)
		return res, err
	}
}

// NewStreamRequestInfoInterceptor is the streaming variant of [NewRequestInfoInterceptor].
func NewStreamRequestInfoInterceptor(l *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		newCtx := fillRequestInfo(ss.Context(), info.FullMethod)
		t0 := time.Now()
		l.InfoContext(newCtx, "stream request started")
		wrapped := &wrappedStream{ServerStream: ss, ctx: newCtx}
		err := handler(srv, wrapped)
		l.InfoContext(
			newCtx,
			"stream request finished",
			"duration", time.Since(t0),
			"err", err,
		)
		return err
	}
}
