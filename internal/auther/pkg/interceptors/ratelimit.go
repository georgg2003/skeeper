// Package interceptors provides gRPC middleware for the Auther service.
package interceptors

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/georgg2003/skeeper/api"
)

const (
	defaultMaxTrackedClients = 4096
	defaultStaleClientTTL    = 30 * time.Minute
)

// RateLimitConfig controls per-client rate limiting for sensitive Auther RPCs.
type RateLimitConfig struct {
	// TrustForwardedFor, when true, uses the first hop in the "x-forwarded-for" gRPC metadata
	// value as the client key when present. Enable only behind a trusted reverse proxy that
	// overwrites or appends this header; otherwise clients can spoof addresses.
	TrustForwardedFor bool `mapstructure:"trust_forwarded_for"`
	// MaxTrackedClients caps distinct client keys; stale entries are evicted first, then oldest.
	MaxTrackedClients int `mapstructure:"max_tracked_clients"`
	// StaleClientTTL drops limiter entries not seen for this long before evicting by age.
	StaleClientTTL time.Duration `mapstructure:"stale_client_ttl"`
}

type limiterEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

func firstForwardedClientIP(md metadata.MD) string {
	v := md.Get("x-forwarded-for")
	if len(v) == 0 {
		return ""
	}
	for _, part := range strings.Split(v[0], ",") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		if h, _, err := net.SplitHostPort(s); err == nil {
			s = h
		}
		s = strings.Trim(s, "[]")
		if ip := net.ParseIP(s); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func clientRateKey(ctx context.Context, cfg RateLimitConfig) string {
	if cfg.TrustForwardedFor {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ip := firstForwardedClientIP(md); ip != "" {
				return "xff:" + ip
			}
		}
	}
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "unknown"
	}
	return "peer:" + p.Addr.String()
}

// NewSensitiveMethodRateLimit rate-limits auth RPCs per client key (best-effort; not shared across replicas).
func NewSensitiveMethodRateLimit(l *slog.Logger, cfg RateLimitConfig) grpc.UnaryServerInterceptor {
	maxClients := cfg.MaxTrackedClients
	if maxClients <= 0 {
		maxClients = defaultMaxTrackedClients
	}
	staleTTL := cfg.StaleClientTTL
	if staleTTL <= 0 {
		staleTTL = defaultStaleClientTTL
	}

	var mu sync.Mutex
	byKey := make(map[string]*limiterEntry)

	evictStale := func(now time.Time) {
		for k, e := range byKey {
			if now.Sub(e.lastSeen) > staleTTL {
				delete(byKey, k)
			}
		}
	}

	evictOldest := func() {
		var oldestK string
		var oldestT time.Time
		first := true
		for k, e := range byKey {
			if first || e.lastSeen.Before(oldestT) {
				oldestT = e.lastSeen
				oldestK = k
				first = false
			}
		}
		if oldestK != "" {
			delete(byKey, oldestK)
		}
	}

	limiterFor := func(key string, now time.Time) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		evictStale(now)
		for len(byKey) >= maxClients {
			evictStale(now)
			if len(byKey) >= maxClients {
				evictOldest()
			}
		}

		e, ok := byKey[key]
		if !ok {
			lim := rate.NewLimiter(rate.Every(3*time.Second), 15)
			e = &limiterEntry{lim: lim, lastSeen: now}
			byKey[key] = e
		}
		e.lastSeen = now
		return e.lim
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		switch info.FullMethod {
		case api.Auther_Login_FullMethodName,
			api.Auther_ExchangeToken_FullMethodName,
			api.Auther_CreateUser_FullMethodName:
		default:
			return handler(ctx, req)
		}

		key := clientRateKey(ctx, cfg)
		if !limiterFor(key, time.Now()).Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
