package auther

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/georgg2003/skeeper/api"
	delivery "github.com/georgg2003/skeeper/internal/auther/delivery"
	"github.com/georgg2003/skeeper/internal/auther/repository/db/postgres"
	"github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/internal/pkg/config"
	"github.com/georgg2003/skeeper/internal/pkg/log"
	"github.com/georgg2003/skeeper/pkg/errors"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

type ServiceConfig struct {
	ListenAddr      string        `mapstructure:"listen_addr"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout"`
}

type AutherConfig struct {
	Postgres postgres.PostgresConfig `mapstructure:"postgres"`
	JWT      JWTConfig               `mapstructure:"jwt"`
	Service  ServiceConfig           `mapstructure:"service"`
}

type JWTConfig struct {
	PrivateKeyFile string `mapstructure:"private_key_file"`
	PublicKeyFile  string `mapstructure:"public_key_file"`
}

func initJWTHelper(cfg JWTConfig) (jwthelper.JWTHelper, error) {
	privBytes, err := os.ReadFile(cfg.PrivateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file")
	}
	pubBytes, err := os.ReadFile(cfg.PublicKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key file")
	}
	return jwthelper.New(privBytes, pubBytes)
}

func serveGRPC(cfg ServiceConfig, server *grpc.Server, l *slog.Logger) func() error {
	return func() error {
		lis, err := net.Listen("tcp", cfg.ListenAddr)
		if err != nil {
			return errors.Wrapf(err, "failed to listen addr %s", cfg.ListenAddr)
		}

		l.Info("gRPC server started", "addr", cfg.ListenAddr)

		if err := server.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			return errors.Wrap(err, "grpc serve failed")
		}
		return nil
	}
}

func shutdownGRPC(
	ctx context.Context,
	cfg ServiceConfig,
	server *grpc.Server,
	l *slog.Logger,
) func() error {
	return func() error {
		<-ctx.Done()

		l.Info("shutting down grpc server gracefully...")

		stopped := make(chan struct{})
		go func() {
			server.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
			l.Info("server stopped gracefully")
		case <-time.After(cfg.GracefulTimeout):
			l.Warn("graceful shutdown timed out, forcing stop")
			server.Stop()
		}

		return nil
	}
}

func main() {
	l := log.New()
	configPath := flag.String("config", "config/auther.yaml", "config file in yaml/json format")
	flag.Parse()

	cfg, err := config.New[AutherConfig](*configPath, "AUTHER")
	if err != nil {
		l.Error("failed to init config", "err", err)
		os.Exit(1)
	}

	jwtHelper, err := initJWTHelper(cfg.JWT)
	if err != nil {
		l.Error("failed to init jwt helper", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	repo, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		l.Error("failed to init repo", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	uc := usecase.New(l, repo, jwtHelper)
	service := delivery.New(l, uc)

	server := grpc.NewServer()
	api.RegisterAutherServer(server, service)
	reflection.Register(server)

	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(serveGRPC(cfg.Service, server, l))
	g.Go(shutdownGRPC(gCtx, cfg.Service, server, l))

	if err = g.Wait(); err != nil {
		l.Error("failed with error", "err", err)
		os.Exit(1)
	}
}
