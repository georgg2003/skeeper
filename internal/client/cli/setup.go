package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/georgg2003/skeeper/internal/client/repository/auther"
	"github.com/georgg2003/skeeper/internal/client/repository/db"
	skeeperremote "github.com/georgg2003/skeeper/internal/client/repository/skeeper"
	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
)

var (
	setupOnce sync.Once
	setupErr  error
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

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

func ensureApp(cmd *cobra.Command) error {
	setupOnce.Do(func() {
		setupErr = bootstrap(cmd)
	})
	return setupErr
}

func bootstrap(cmd *cobra.Command) error {
	autherAddr, err := cmd.Flags().GetString("auther")
	if err != nil {
		return err
	}
	skeeperAddr, err := cmd.Flags().GetString("skeeper")
	if err != nil {
		return err
	}
	dataDir, err := cmd.Flags().GetString("data-dir")
	if err != nil {
		return err
	}

	dir, err := expandPath(dataDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("data dir: %w", err)
	}
	dbPath, err := filepath.Abs(filepath.Join(dir, "local.db"))
	if err != nil {
		return err
	}

	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbRepo, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	if err := dbRepo.RunMigrations(context.Background()); err != nil {
		_ = dbRepo.Close()
		return fmt.Errorf("migrations: %w", err)
	}

	autherCLI, err := auther.NewAutherClient(autherAddr)
	if err != nil {
		_ = dbRepo.Close()
		return fmt.Errorf("auther client: %w", err)
	}

	authUC := usecase.NewAuthUseCase(dbRepo, autherCLI, l)
	secretUC := usecase.NewSecretUseCase(dbRepo, l)

	skeeperCLI, err := skeeperremote.NewSkeeperClient(skeeperAddr, authUC)
	if err != nil {
		_ = dbRepo.Close()
		return fmt.Errorf("skeeper client: %w", err)
	}
	syncUC := usecase.NewSyncUseCase(dbRepo, skeeperCLI, l)

	SetUseCases(authUC, secretUC, syncUC)
	return nil
}
