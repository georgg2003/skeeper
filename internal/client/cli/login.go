package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Auther service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if authUC == nil {
			return fmt.Errorf("client not initialized")
		}
		writePrompt(cmd, "Email: ")
		username, err := readLine(cmd)
		if err != nil {
			return fmt.Errorf("read email: %w", err)
		}

		writePrompt(cmd, "Password: ")
		password, err := readPasswordLine(cmd)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		ctx := context.Background()
		if err := authUC.Login(ctx, username, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged in.")
		return nil
	},
}
