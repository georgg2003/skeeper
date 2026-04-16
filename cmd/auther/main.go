// Command auther is the account service: register, login, refresh tokens (Postgres + JWT).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/georgg2003/skeeper/api"
	"github.com/georgg2003/skeeper/internal/auther/delivery"
	"github.com/georgg2003/skeeper/internal/auther/pkg/config"
	"github.com/georgg2003/skeeper/internal/auther/repository/postgres"
	"github.com/georgg2003/skeeper/internal/auther/usecase"
	"github.com/georgg2003/skeeper/internal/pkg/log"
	"github.com/georgg2003/skeeper/internal/pkg/server"
	"github.com/georgg2003/skeeper/pkg/jwthelper"
)

func main() {
	l := log.New()

	cfg, err := config.New()
	if err != nil {
		l.Error("failed to init config", "err", err)
		os.Exit(1)
	}

	jwtHelper, err := jwthelper.NewFromConfig(cfg.JWT)
	if err != nil {
		l.Error("failed to init jwt helper", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		l.Error("failed to init repo", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	uc, err := usecase.New(l, repo, jwtHelper)
	if err != nil {
		l.Error("failed to init usecase", "err", err)
		os.Exit(1)
	}
	service := delivery.New(l, uc)

	srv, err := server.New(
		cfg.Service,
		l,
		func(s *grpc.Server) {
			api.RegisterAutherServer(s, service)
		},
	)
	if err != nil {
		l.Error("failed to init grpc server", "err", err)
		os.Exit(1)
	}

	if err = srv.Serve(ctx); err != nil {
		l.Error("failed with error", "err", err)
		os.Exit(1)
	}
}
