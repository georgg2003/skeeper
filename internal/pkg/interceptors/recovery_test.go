package interceptors

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUnaryRecoveryInterceptor_RecoversPanic(t *testing.T) {
	ic := NewUnaryRecoveryInterceptor(discardLog())
	_, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Ping"},
		func(context.Context, any) (any, error) {
			panic("boom")
		},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestUnaryRecoveryInterceptor_Passthrough(t *testing.T) {
	ic := NewUnaryRecoveryInterceptor(discardLog())
	out, err := ic(context.Background(), "in", &grpc.UnaryServerInfo{},
		func(context.Context, any) (any, error) {
			return "out", nil
		},
	)
	if err != nil || out != "out" {
		t.Fatalf("got %v %v", out, err)
	}
}

func TestStreamRecoveryInterceptor_RecoversPanic(t *testing.T) {
	ic := NewStreamRecoveryInterceptor(discardLog())
	ss := &fakeStream{ctx: context.Background()}
	err := ic(nil, ss, &grpc.StreamServerInfo{FullMethod: "/test/Stream"},
		func(any, grpc.ServerStream) error {
			panic("stream boom")
		})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeStream) Context() context.Context { return f.ctx }
