package cli

import (
	"context"
	"fmt"
	"syscall"

	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addPasswordCmd = &cobra.Command{
	Use:   "add-password",
	Short: "Add an encrypted login/password entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if secretUC == nil {
			return fmt.Errorf("client not initialized")
		}
		var name, notes string
		fmt.Print("Entry name (e.g. Gmail): ")
		if _, err := fmt.Scanln(&name); err != nil {
			return err
		}
		fmt.Print("Notes (optional): ")
		_, _ = fmt.Scanln(&notes)

		fmt.Print("Password to store: ")
		secretBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()

		fmt.Print("Master password: ")
		masterBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetPassword(ctx, meta, string(secretBytes), string(masterBytes)); err != nil {
			return fmt.Errorf("save secret: %w", err)
		}

		fmt.Println("Encrypted entry saved locally (run `sync` to upload).")
		return nil
	},
}
