// Package cli defines Cobra commands for the skeeper client.
package cli

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"
)

// Version and BuildTime are injected at link time (e.g. -ldflags).
var (
	Version   = "dev"
	BuildTime = "unknown"

	authUC   AuthCommands
	secretUC SecretCommands
	syncUC   SyncCommands
)

// SetUseCases wires use cases (used from tests or after bootstrap).
func SetUseCases(a AuthCommands, s SecretCommands, y SyncCommands) {
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

// Run executes the root command with explicit stdio and argv (e.g. integration tests). Args should not include the program name.
// Nil readers/writers fall back to [os.Stdin], [os.Stdout], and [os.Stderr].
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	rootCmd.SetArgs(args)
	if stdin != nil {
		rootCmd.SetIn(stdin)
	} else {
		rootCmd.SetIn(os.Stdin)
	}
	if stdout != nil {
		rootCmd.SetOut(stdout)
	} else {
		rootCmd.SetOut(os.Stdout)
	}
	if stderr != nil {
		rootCmd.SetErr(stderr)
	} else {
		rootCmd.SetErr(os.Stderr)
	}
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().String(
		"config",
		"config/client.yaml",
		"YAML config (Viper); if this path is missing and you did not pass --config, defaults and SKEEPERCLI_* env apply",
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
