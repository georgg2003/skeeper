package main

import (
	"context"
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
		return nil, fmt.Errorf("client config: %w", err)
	}

	dir, err := expandPath(fileCfg.DataDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("data dir: %w", err)
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

	if err := dbRepo.RunMigrations(context.Background()); err != nil {
		_ = dbRepo.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	dialOpts, err := grpcclient.DialOptions(grpcclient.TLSConfig{
		Enabled: fileCfg.GRPCTLS.Enabled,
		CAFile:  fileCfg.GRPCTLS.CAFile,
	})
	if err != nil {
		_ = dbRepo.Close()
		return nil, fmt.Errorf("grpc dial options: %w", err)
	}

	autherCLI, err := auther.NewAutherClient(fileCfg.AutherAddr, dialOpts...)
	if err != nil {
		_ = dbRepo.Close()
		return nil, fmt.Errorf("auther client: %w", err)
	}

	authUC := usecase.NewAuthUseCase(dbRepo, autherCLI, l)

	skeeperCLI, err := skeeperremote.NewSkeeperClient(fileCfg.SkeeperAddr, authUC, dialOpts...)
	if err != nil {
		_ = dbRepo.Close()
		return nil, fmt.Errorf("skeeper client: %w", err)
	}

	secretUC := usecase.NewSecretUseCase(dbRepo, dbRepo, skeeperCLI, l, maxFileBytes)
	syncUC := usecase.NewSyncUseCase(dbRepo, skeeperCLI, dbRepo, l)

	return delivery.New(authUC, secretUC, syncUC)
}
