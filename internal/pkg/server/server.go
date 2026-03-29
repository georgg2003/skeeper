package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/georgg2003/skeeper/internal/pkg/interceptors"
	"github.com/georgg2003/skeeper/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type ServerConfig struct {
	ListenAddr      string        `mapstructure:"listen_addr"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout"`
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

func New(
	cfg ServerConfig,
	l *slog.Logger,
	modifyServer ServerModifyFunc,
	opt ...grpc.ServerOption,
) *Server {
	defaultOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.NewRequestInfoInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			interceptors.NewStreamRequestInfoInterceptor(),
		),
	}

	opts := append(defaultOpts, opt...)

	server := grpc.NewServer(opts...)
	modifyServer(server)
	reflection.Register(server)

	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)

	return &Server{
		l:          l,
		cfg:        cfg,
		grpcServer: server,
	}
}
