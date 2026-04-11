package interceptors

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Error(t, err, "expected error")
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestUnaryRecoveryInterceptor_Passthrough(t *testing.T) {
	ic := NewUnaryRecoveryInterceptor(discardLog())
	out, err := ic(context.Background(), "in", &grpc.UnaryServerInfo{},
		func(context.Context, any) (any, error) {
			return "out", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "out", out)
}

func TestStreamRecoveryInterceptor_RecoversPanic(t *testing.T) {
	ic := NewStreamRecoveryInterceptor(discardLog())
	ss := &fakeStream{ctx: context.Background()}
	err := ic(nil, ss, &grpc.StreamServerInfo{FullMethod: "/test/Stream"},
		func(any, grpc.ServerStream) error {
			panic("stream boom")
		})
	require.Error(t, err, "expected error")
	assert.Equal(t, codes.Internal, status.Code(err))
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeStream) Context() context.Context { return f.ctx }
