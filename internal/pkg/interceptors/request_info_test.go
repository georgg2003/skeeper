package interceptors

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
)

func TestRequestInfoInterceptor_SetsRequestIDFromMetadata(t *testing.T) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	ic := NewRequestInfoInterceptor(l)
	md := metadata.Pairs("x-request-id", "req-123", "user-agent", "test-agent")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: &net.TCPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 443}})
	_, err := ic(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/M"},
		func(c context.Context, _ any) (any, error) {
			ri, ok := contextlib.GetRequestInfo(c)
			if !ok || ri.RequestID != "req-123" || ri.UserAgent != "test-agent" {
				t.Fatalf("request info %+v", ri)
			}
			return nil, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamRequestInfoInterceptor(t *testing.T) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	ic := NewStreamRequestInfoInterceptor(l)
	md := metadata.Pairs("host", "example.com")
	base := metadata.NewIncomingContext(context.Background(), md)
	ss := &fakeStream{ctx: base}
	err := ic(nil, ss, &grpc.StreamServerInfo{FullMethod: "/svc/S"},
		func(_ any, stream grpc.ServerStream) error {
			ri, ok := contextlib.GetRequestInfo(stream.Context())
			if !ok || ri.Host != "example.com" {
				t.Fatalf("request info %+v", ri)
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
}
