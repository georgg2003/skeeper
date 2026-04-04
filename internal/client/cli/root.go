// Package cli defines Cobra commands for the skeeper client.
package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
)

// Version and BuildTime are injected at link time (e.g. -ldflags).
var (
	Version   = "dev"
	BuildTime = "unknown"

	authUC   *usecase.AuthUseCase
	secretUC *usecase.SecretUseCase
	syncUC   *usecase.SyncUseCase
)

// SetUseCases wires use cases (used from tests or after bootstrap).
func SetUseCases(a *usecase.AuthUseCase, s *usecase.SecretUseCase, y *usecase.SyncUseCase) {
	authUC, secretUC, syncUC = a, s, y
}

var rootCmd = &cobra.Command{
	Use:   "skeepercli",
	Short: "skeeper password manager CLI",
	Long:  "Local encrypted vault with Auther (auth) and Skeeper (sync) backends.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if slices.ContainsFunc(os.Args, func(s string) bool {
			return s == "-h" || s == "--help" || s == "-?"
		}) {
			return nil
		}
		return ensureApp(cmd)
	},
}

// Execute runs the root command tree.
func Execute() {
	rootCmd.Version = fmt.Sprintf("%s (built %s)", Version, BuildTime)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String(
		"config",
		"config/client.yaml",
		"Path to client YAML config (Viper)",
	)
	rootCmd.PersistentFlags().String(
		"auther",
		"",
		"Auther gRPC host:port (overrides config and SKEEPERCLI_AUTHER)",
	)
	rootCmd.PersistentFlags().String(
		"skeeper",
		"",
		"Skeeper gRPC host:port (overrides config and SKEEPERCLI_SKEEPER)",
	)
	rootCmd.PersistentFlags().String(
		"data-dir",
		"",
		"Local vault directory (overrides config and SKEEPERCLI_DATA)",
	)

	rootCmd.AddCommand(
		loginCmd,
		registerCmd,
		logoutCmd,
		syncCmd,
		addPasswordCmd,
		addTextCmd,
		addBinaryCmd,
		addCardCmd,
		listCmd,
		getCmd,
	)
}
