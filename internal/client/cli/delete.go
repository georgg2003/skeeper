package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete UUID",
	Short: "Soft-delete an entry (hidden from list; sync removes on server after upload)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		id, err := parseUUIDArg(args[0])
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.DeleteEntry(ctx, id, masterStr); err != nil {
			return fmt.Errorf("delete entry: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry deleted locally (run `sync` to upload).")
		return nil
	},
}
