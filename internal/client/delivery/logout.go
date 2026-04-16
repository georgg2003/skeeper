package delivery

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) Logout(cmd *cobra.Command, args []string) error {
	if err := d.auth.Logout(cmd.Context()); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
	return nil
}
