package interceptors

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestFirstForwardedClientIP(t *testing.T) {
	md := metadata.Pairs("x-forwarded-for", "203.0.113.1, 10.0.0.1")
	if got := firstForwardedClientIP(md); got != "203.0.113.1" {
		t.Fatalf("got %q", got)
	}
	md2 := metadata.Pairs("x-forwarded-for", "[::1]:12345, 10.0.0.2")
	if got := firstForwardedClientIP(md2); got != "::1" {
		t.Fatalf("got %q", got)
	}
	if got := firstForwardedClientIP(metadata.MD{}); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestClientRateKeyForwardedAndPeer(t *testing.T) {
	md := metadata.Pairs("x-forwarded-for", "198.51.100.2")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 50000}})

	if got := clientRateKey(ctx, RateLimitConfig{TrustForwardedFor: true}); got != "xff:198.51.100.2" {
		t.Fatalf("got %q", got)
	}
	if got := clientRateKey(ctx, RateLimitConfig{TrustForwardedFor: false}); got != "peer:10.0.0.1:50000" {
		t.Fatalf("got %q", got)
	}
}
