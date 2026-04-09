package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account on the Auther service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthUC(); err != nil {
			return err
		}
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

		ctx := context.Background()
		if err := authUC.Register(ctx, email, pw1); err != nil {
			return fmt.Errorf("register failed: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Registered and logged in.")
		return nil
	},
}
