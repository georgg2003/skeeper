package delivery

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) Register(cmd *cobra.Command, args []string) error {
	writePrompt(cmd, "Email: ")
	email, err := readLine(cmd)
	if err != nil {
		return fmt.Errorf("read email: %w", err)
	}

	writePrompt(cmd, "Password: ")
	pw1, err := readPasswordLine(cmd)
	if err != nil {
		return err
	}
	writePrompt(cmd, "Confirm password: ")
	pw2, err := readPasswordLine(cmd)
	if err != nil {
		return err
	}
	if pw1 != pw2 {
		return fmt.Errorf("passwords do not match")
	}

	ctx := cmd.Context()
	if err := d.auth.Register(ctx, email, pw1); err != nil {
		return fmt.Errorf("register failed: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Registered and logged in.")
	return nil
}
