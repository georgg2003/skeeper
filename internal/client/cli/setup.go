package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	clientcfg "github.com/georgg2003/skeeper/internal/client/pkg/config"
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
	root := cmd.Root()
	cfgPath, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return err
	}
	fileCfg, err := clientcfg.New(cfgPath)
	if err != nil {
		return fmt.Errorf("client config: %w", err)
	}

	autherAddr := pickAddr(root, "auther", fileCfg.AutherAddr, "127.0.0.1:50051")
	skeeperAddr := pickAddr(root, "skeeper", fileCfg.SkeeperAddr, "127.0.0.1:50052")
	dataDir := pickAddr(root, "data-dir", fileCfg.DataDir, "~/.skeepercli")

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

// pickAddr returns the flag value if that persistent flag was set on the CLI;
// otherwise value from config/env (already merged by Viper), then defaultDefault.
func pickAddr(root *cobra.Command, flagName, fromCfg, defaultVal string) string {
	if root.PersistentFlags().Changed(flagName) {
		v, _ := root.PersistentFlags().GetString(flagName)
		return v
	}
	if fromCfg != "" {
		return fromCfg
	}
	return defaultVal
}
