package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/delivery"
	"github.com/georgg2003/skeeper/internal/client/delivery/cli"
	clientcfg "github.com/georgg2003/skeeper/internal/client/pkg/config"
	"github.com/georgg2003/skeeper/internal/client/repository/auther"
	"github.com/georgg2003/skeeper/internal/client/repository/db"
	skeeperremote "github.com/georgg2003/skeeper/internal/client/repository/skeeper"
	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/georgg2003/skeeper/internal/pkg/grpcclient"
	"github.com/georgg2003/skeeper/pkg/errors"
)

func expandPath(p string) (string, error) {
	if len(p) >= 2 && p[0] == '~' && (p[1] == '/' || p[1] == '\\') {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(h, p[2:]), nil
	}
	return p, nil
}

func parseSlogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "err", "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func newClientLogger(cmd *cobra.Command, logCfg clientcfg.ClientLogging) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: parseSlogLevel(logCfg.Level)}
	out := io.Writer(cmd.Root().OutOrStdout())
	if path := strings.TrimSpace(logCfg.File); path != "" {
		p, err := expandPath(path)
		if err != nil {
			return nil, fmt.Errorf("log file path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			return nil, fmt.Errorf("log file dir: %w", err)
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		out = f
	}
	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(logCfg.Format)) {
	case "text":
		h = slog.NewTextHandler(out, opts)
	default:
		h = slog.NewJSONHandler(out, opts)
	}
	return slog.New(h), nil
}

// BuildDelivery loads settings through Viper (YAML + env), opens local DB and gRPC clients, and returns [cli.Handlers].
func BuildDelivery(cmd *cobra.Command) (cli.Handlers, error) {
	fileCfg, err := clientcfg.Load(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load client config")
	}

	dir, err := expandPath(fileCfg.DataDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, errors.Wrap(err, "data dir")
	}

	maxFileBytes := fileCfg.MaxFileBytes
	dbPath, err := filepath.Abs(filepath.Join(dir, "local.db"))
	if err != nil {
		return nil, err
	}

	l, err := newClientLogger(cmd, fileCfg.Logging)
	if err != nil {
		return nil, err
	}

	dbRepo, err := db.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err != nil {
			if repoErr := dbRepo.Close(); repoErr != nil {
				l.Error("failed to close db", "err", repoErr)
			}
		}
	}()

	if err = dbRepo.RunMigrations(cmd.Context()); err != nil {
		return nil, errors.Wrap(err, "migrations")
	}

	dialOpts, err := grpcclient.DialOptions(grpcclient.TLSConfig{
		Enabled: fileCfg.GRPCTLS.Enabled,
		CAFile:  fileCfg.GRPCTLS.CAFile,
	})
	if err != nil {
		return nil, errors.Wrap(err, "grpc dial options")
	}

	autherCLI, err := auther.NewAutherClient(fileCfg.AutherAddr, dialOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "auther client")
	}

	authUC := usecase.NewAuthUseCase(dbRepo, autherCLI, l)

	skeeperCLI, err := skeeperremote.NewSkeeperClient(fileCfg.SkeeperAddr, authUC, dialOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "skeeper client")
	}
	defer func() {
		if err != nil {
			if skeeperErr := skeeperCLI.Close(cmd.Context()); skeeperErr != nil {
				l.Error("failed to skeeper client", "err", skeeperErr)
			}
		}
	}()

	secretUC := usecase.NewSecretUseCase(dbRepo, dbRepo, skeeperCLI, l, maxFileBytes)
	syncUC := usecase.NewSyncUseCase(dbRepo, skeeperCLI, dbRepo, l)

	delivery, err := delivery.New(authUC, secretUC, syncUC)
	return delivery, err
}
