package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var addPasswordCmd = &cobra.Command{
	Use:   "add-password",
	Short: "Add an encrypted login/password entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "Entry name (e.g. Gmail): ")
		if err != nil {
			return err
		}
		secretStr, err := promptPasswordValue(cmd, "Password to store: ")
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.SetPassword(ctx, meta, secretStr, masterStr); err != nil {
			return fmt.Errorf("save secret: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted entry saved locally (run `sync` to upload).")
		return nil
	},
}
