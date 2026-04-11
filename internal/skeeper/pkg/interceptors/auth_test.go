package interceptors

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/georgg2003/skeeper/internal/pkg/contextlib"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

func testJWTHelper(t *testing.T) *jwthelper.JWTHelper {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	h, err := jwthelper.New(privPEM, pubPEM, time.Minute, time.Hour, "")
	require.NoError(t, err)
	return h
}

func TestAuthInterceptor_ValidBearerSetsUserID(t *testing.T) {
	h := testJWTHelper(t)
	pair, err := h.NewTokenPair(42)
	require.NoError(t, err)
	ic := NewAuthInterceptor(h)
	md := metadata.Pairs("authorization", "Bearer "+pair.AccessToken.Token)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err = ic(ctx, nil, &grpc.UnaryServerInfo{},
		func(c context.Context, _ any) (any, error) {
			uid, ok := contextlib.GetUserID(c)
			require.True(t, ok)
			assert.Equal(t, int64(42), uid)
			return nil, nil
		})
	require.NoError(t, err)
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	ic := NewAuthInterceptor(testJWTHelper(t))
	_, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{},
		func(context.Context, any) (any, error) { return nil, nil })
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_InvalidBearerFormat(t *testing.T) {
	ic := NewAuthInterceptor(testJWTHelper(t))
	md := metadata.Pairs("authorization", "not-bearer x")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := ic(ctx, nil, &grpc.UnaryServerInfo{},
		func(context.Context, any) (any, error) { return nil, nil })
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestStreamAuthInterceptor_ValidBearer(t *testing.T) {
	h := testJWTHelper(t)
	pair, err := h.NewTokenPair(7)
	require.NoError(t, err)
	ic := NewStreamAuthInterceptor(h)
	md := metadata.Pairs("authorization", "Bearer "+pair.AccessToken.Token)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &wrappedStreamStub{ctx: ctx}
	err = ic(nil, ss, &grpc.StreamServerInfo{},
		func(_ any, stream grpc.ServerStream) error {
			uid, ok := contextlib.GetUserID(stream.Context())
			require.True(t, ok)
			assert.Equal(t, int64(7), uid)
			return nil
		})
	require.NoError(t, err)
}

type wrappedStreamStub struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStreamStub) Context() context.Context { return w.ctx }
