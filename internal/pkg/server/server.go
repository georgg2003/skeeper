package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/georgg2003/skeeper/internal/pkg/interceptors"
	"github.com/georgg2003/skeeper/pkg/errors"
)

// TLSConfig holds server TLS material for gRPC over TLS.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type ServerConfig struct {
	ListenAddr      string        `mapstructure:"listen_addr"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout"`
	TLS             TLSConfig     `mapstructure:"tls"`
}

type Server struct {
	l          *slog.Logger
	cfg        ServerConfig
	grpcServer *grpc.Server
}

func (s *Server) shutdownGRPC(
	ctx context.Context,
) func() error {
	return func() error {
		<-ctx.Done()

		s.l.Info("shutting down grpc server gracefully...")

		stopped := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
			s.l.Info("server stopped gracefully")
		case <-time.After(s.cfg.GracefulTimeout):
			s.l.Warn("graceful shutdown timed out, forcing stop")
			s.grpcServer.Stop()
		}

		return nil
	}
}

func (s *Server) serveGRPC() error {
	lis, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen addr %s", s.cfg.ListenAddr)
	}

	s.l.Info("gRPC server started", "addr", s.cfg.ListenAddr)

	if err := s.grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		return errors.Wrap(err, "grpc serve failed")
	}
	return nil
}

func (s *Server) Serve(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(s.serveGRPC)
	g.Go(s.shutdownGRPC(gCtx))

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "serve failed")
	}

	return nil
}

type ServerModifyFunc func(*grpc.Server)

// New builds a gRPC server with request logging, panic recovery, and optional TLS.
func New(
	cfg ServerConfig,
	l *slog.Logger,
	modifyServer ServerModifyFunc,
	opt ...grpc.ServerOption,
) (*Server, error) {
	var serverOpts []grpc.ServerOption

	if cfg.TLS.Enabled {
		if cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "" {
			return nil, errors.New("tls enabled but cert_file or key_file is empty")
		}
		creds, err := credentials.NewServerTLSFromFile(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "load grpc tls credentials")
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	defaultOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.NewUnaryRecoveryInterceptor(l),
			interceptors.NewRequestInfoInterceptor(l),
		),
		grpc.ChainStreamInterceptor(
			interceptors.NewStreamRecoveryInterceptor(l),
			interceptors.NewStreamRequestInfoInterceptor(l),
		),
	}

	serverOpts = append(serverOpts, defaultOpts...)
	serverOpts = append(serverOpts, opt...)

	grpcSrv := grpc.NewServer(serverOpts...)
	modifyServer(grpcSrv)
	reflection.Register(grpcSrv)

	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthService)

	return &Server{
		l:          l,
		cfg:        cfg,
		grpcServer: grpcSrv,
	}, nil
}
