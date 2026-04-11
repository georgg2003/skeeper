// Package cli is the Cobra front-end for the skeeper client ([Handlers], typically [*delivery.Delivery]).
//
//go:generate go tool mockgen -typed -destination=mock_handlers_test.go -package=cli -source=handlers.go Handlers
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"

	"github.com/spf13/cobra"
)

// Config configures [App].
type Config struct {
	Version   string
	BuildTime string
	// Handlers is injected in tests. If nil, Wire builds handlers on first command.
	Handlers Handlers
	Wire     func(*cobra.Command) (Handlers, error)
}

// App is the runnable CLI (Cobra root + optional lazy wiring).
type App struct {
	cfg Config

	handlers Handlers

	root     *cobra.Command
	wireOnce sync.Once
	wireErr  error
}

// New returns an [App]. Provide either cfg.Handlers (tests) or cfg.Wire (production).
func New(cfg Config) (*App, error) {
	if cfg.Handlers == nil && cfg.Wire == nil {
		return nil, fmt.Errorf("cli.New: set either Config.Handlers or Config.Wire")
	}
	a := &App{cfg: cfg, handlers: cfg.Handlers}
	a.root = a.buildRoot()
	return a, nil
}

func (a *App) buildRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "skeepercli",
		Short: "skeeper password manager CLI",
		Long:  "Local encrypted vault with Auther (auth) and Skeeper (sync) backends.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if slices.ContainsFunc(os.Args, func(s string) bool {
				return s == "-h" || s == "--help" || s == "-?"
			}) {
				return nil
			}
			return a.ensureHandlers(cmd)
		},
	}
	root.PersistentFlags().String(
		"config",
		"config/client.yaml",
		"Path to client YAML merged into Viper (optional file; overrides via SKEEPERCLI_* env).",
	)

	add := &cobra.Command{
		Use:   "add",
		Short: "Create a new encrypted vault entry",
	}
	add.AddCommand(
		&cobra.Command{Use: "password", Short: "Add an encrypted login/password entry", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.AddPassword(cmd, args)
		}},
		&cobra.Command{Use: "text", Short: "Add arbitrary encrypted text", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.AddText(cmd, args)
		}},
		&cobra.Command{
			Use:   "file PATH",
			Short: "Encrypt a small file; ciphertext is stored in the entry payload (syncs to the server)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.AddFile(cmd, args)
			},
		},
		&cobra.Command{Use: "card", Short: "Add an encrypted bank card entry", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.AddCard(cmd, args)
		}},
	)

	update := &cobra.Command{
		Use:   "update",
		Short: "Change an existing vault entry (increments version for sync)",
	}
	update.AddCommand(
		&cobra.Command{
			Use:   "password UUID",
			Short: "Update a PASSWORD entry (new secret + name/notes)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.UpdatePassword(cmd, args)
			},
		},
		&cobra.Command{
			Use:   "text UUID",
			Short: "Update a TEXT entry",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.UpdateText(cmd, args)
			},
		},
		&cobra.Command{
			Use:   "card UUID",
			Short: "Update a CARD entry",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.UpdateCard(cmd, args)
			},
		},
		&cobra.Command{
			Use:   "file UUID",
			Short: "Update a FILE entry (metadata only, or replace bytes from a path)",
			Long: "Updates name/notes. Optionally replace file content: enter a path to read new bytes, or leave empty to keep " +
				"the existing ciphertext. Sync uses the same versioned last-write-wins rule as other entry types: the server stores " +
				"one encrypted blob per entry—conflicting edits are not merged at the binary level.",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.UpdateFile(cmd, args)
			},
		},
	)

	root.AddCommand(
		&cobra.Command{Use: "login", Short: "Authenticate with the Auther service", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.Login(cmd, args)
		}},
		&cobra.Command{Use: "register", Short: "Create a new account on the Auther service", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.Register(cmd, args)
		}},
		&cobra.Command{Use: "logout", Short: "Clear the local session", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.Logout(cmd, args)
		}},
		&cobra.Command{Use: "sync", Short: "Upload dirty entries and fetch remote updates from Skeeper", RunE: func(cmd *cobra.Command, args []string) error {
			return a.handlers.Sync(cmd, args)
		}},
		add,
		update,
		&cobra.Command{
			Use:   "delete UUID",
			Short: "Soft-delete an entry (hidden from list; sync removes on server after upload)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.Delete(cmd, args)
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List local entries (uuid, type, updated time; ciphertext only)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.List(cmd, args)
			},
		},
		&cobra.Command{
			Use:   "get UUID",
			Short: "Decrypt and print one entry (requires master password)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.handlers.Get(cmd, args)
			},
		},
	)

	return root
}

func (a *App) ensureHandlers(cmd *cobra.Command) error {
	if a.handlers != nil {
		return nil
	}
	if a.cfg.Wire == nil {
		return fmt.Errorf("client not configured")
	}
	a.wireOnce.Do(func() {
		a.handlers, a.wireErr = a.cfg.Wire(cmd)
	})
	return a.wireErr
}

// Execute runs the root command and exits the process on error (no cancellation on signals).
func (a *App) Execute() {
	a.ExecuteContext(context.Background())
}

// ExecuteContext runs the root with ctx propagated to handlers ([cobra.Command.Context]); cancel on SIGINT/SIGTERM from [main] via [signal.NotifyContext].
func (a *App) ExecuteContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ver := a.cfg.Version
	if ver == "" {
		ver = "dev"
	}
	bt := a.cfg.BuildTime
	if bt == "" {
		bt = "unknown"
	}
	a.root.Version = fmt.Sprintf("%s (built %s)", ver, bt)
	a.root.SetVersionTemplate("{{.Version}}\n")
	if err := a.root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Run runs the CLI with injected argv and stdio (tests). Args must not include argv[0].
func (a *App) Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	a.root.SetArgs(args)
	if stdin != nil {
		a.root.SetIn(stdin)
	} else {
		a.root.SetIn(os.Stdin)
	}
	if stdout != nil {
		a.root.SetOut(stdout)
	} else {
		a.root.SetOut(os.Stdout)
	}
	if stderr != nil {
		a.root.SetErr(stderr)
	} else {
		a.root.SetErr(os.Stderr)
	}
	return a.root.ExecuteContext(context.Background())
}
