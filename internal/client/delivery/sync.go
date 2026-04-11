package delivery

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) Sync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	if err := d.sync.Sync(ctx); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Sync completed.")
	return nil
}
