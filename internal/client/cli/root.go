// Package cli defines Cobra commands for the GophKeeper client.
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
	Use:   "gophkeeper",
	Short: "GophKeeper password manager CLI",
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
	rootCmd.PersistentFlags().String("auther", getenv("GOPHKEEPER_AUTHER", "127.0.0.1:50051"), "Auther gRPC host:port")
	rootCmd.PersistentFlags().String("skeeper", getenv("GOPHKEEPER_SKEEPER", "127.0.0.1:50052"), "Skeeper gRPC host:port")
	rootCmd.PersistentFlags().String("data-dir", getenv("GOPHKEEPER_DATA", "~/.gophkeeper"), "Local vault directory (SQLite)")

	rootCmd.AddCommand(loginCmd, registerCmd, logoutCmd, syncCmd)
	rootCmd.AddCommand(addPasswordCmd, addTextCmd, addBinaryCmd, addCardCmd)
	rootCmd.AddCommand(listCmd, getCmd)
}
