package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var addCardCmd = &cobra.Command{
	Use:   "add-card",
	Short: "Add an encrypted bank card entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "")
		if err != nil {
			return err
		}
		card, err := promptCard(cmd)
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.SetCard(ctx, meta, card, masterStr); err != nil {
			return fmt.Errorf("save card: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted card saved locally (run `sync` to upload).")
		return nil
	},
}
