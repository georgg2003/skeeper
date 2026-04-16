package server

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func testServerLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNew_TLSEnabledMissingCertFiles(t *testing.T) {
	_, err := New(ServerConfig{
		ListenAddr:      "127.0.0.1:0",
		GracefulTimeout: time.Millisecond * 50,
		TLS:             TLSConfig{Enabled: true},
	}, testServerLog(), func(*grpc.Server) {})
	require.Error(t, err, "expected error")
}

func TestNew_ServeStopsOnCancel(t *testing.T) {
	srv, err := New(ServerConfig{
		ListenAddr:      "127.0.0.1:0",
		GracefulTimeout: time.Millisecond * 100,
	}, testServerLog(), func(*grpc.Server) {})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		require.Fail(t, "server did not stop")
	}
}
