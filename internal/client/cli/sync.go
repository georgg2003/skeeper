package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Upload local changes and download remote updates from Skeeper",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSyncUC(); err != nil {
			return err
		}
		ctx := context.Background()
		if err := syncUC.Sync(ctx); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Sync completed.")
		return nil
	},
}
