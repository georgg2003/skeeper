package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the local Auther session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if authUC == nil {
			return fmt.Errorf("client not initialized")
		}
		if err := authUC.Logout(context.Background()); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
		return nil
	},
}
