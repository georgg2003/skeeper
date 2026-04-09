// Package cli is the skeepercli command tree (Cobra). Commands depend on small interfaces in usecase_ports.go for testing.
package cli

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"
)

var (
	// Version is the CLI release label (overridden via -ldflags from main).
	Version = "dev"
	// BuildTime is an optional build timestamp string (overridden via -ldflags from main).
	BuildTime = "unknown"

	authUC   AuthCommands
	secretUC SecretCommands
	syncUC   SyncCommands
)

// SetUseCases injects application use cases before commands run (used from cmd/client).
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

// Execute runs the root Cobra command and exits the process on error.
func Execute() {
	rootCmd.Version = fmt.Sprintf("%s (built %s)", Version, BuildTime)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Run is like Execute but with injected argv and stdio (tests). Args must not include argv[0].
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
		addFileCmd,
		addCardCmd,
		updateCmd,
		deleteCmd,
		listCmd,
		getCmd,
	)
}
