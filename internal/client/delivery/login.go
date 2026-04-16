package delivery

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) Login(cmd *cobra.Command, args []string) error {
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

	ctx := cmd.Context()
	if err := d.auth.Login(ctx, username, password); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged in.")
	return nil
}
