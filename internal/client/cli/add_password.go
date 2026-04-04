package cli

import (
	"context"
	"fmt"

	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
)

var addPasswordCmd = &cobra.Command{
	Use:   "add-password",
	Short: "Add an encrypted login/password entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if secretUC == nil {
			return fmt.Errorf("client not initialized")
		}
		writePrompt(cmd, "Entry name (e.g. Gmail): ")
		name, err := readLine(cmd)
		if err != nil {
			return err
		}
		writePrompt(cmd, "Notes (optional): ")
		notes, _ := readLine(cmd)

		writePrompt(cmd, "Password to store: ")
		secretStr, err := readPasswordLine(cmd)
		if err != nil {
			return err
		}

		writePrompt(cmd, "Master password: ")
		masterStr, err := readPasswordLine(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetPassword(ctx, meta, secretStr, masterStr); err != nil {
			return fmt.Errorf("save secret: %w", err)
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted entry saved locally (run `sync` to upload).")
		return nil
	},
}
