package server

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func testServerLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNew_DisallowsPlaintextWithoutExplicitOptIn(t *testing.T) {
	_, err := New(ServerConfig{
		ListenAddr:      "127.0.0.1:0",
		GracefulTimeout: time.Millisecond * 50,
	}, testServerLog(), func(*grpc.Server) {})
	if err == nil {
		t.Fatal("expected error when TLS is off and allow_insecure_transport is false")
	}
}

func TestNew_TLSEnabledMissingCertFiles(t *testing.T) {
	_, err := New(ServerConfig{
		ListenAddr:      "127.0.0.1:0",
		GracefulTimeout: time.Millisecond * 50,
		TLS:             TLSConfig{Enabled: true},
	}, testServerLog(), func(*grpc.Server) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNew_ServeStopsOnCancel(t *testing.T) {
	srv, err := New(ServerConfig{
		ListenAddr:               "127.0.0.1:0",
		GracefulTimeout:          time.Millisecond * 100,
		AllowInsecureTransport:   true,
	}, testServerLog(), func(*grpc.Server) {})
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatal("server did not stop")
	}
}
