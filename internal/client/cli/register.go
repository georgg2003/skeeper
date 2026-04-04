package cli

import (
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account on the Auther service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if authUC == nil {
			return fmt.Errorf("client not initialized")
		}
		var email string
		fmt.Print("Email: ")
		if _, err := fmt.Scanln(&email); err != nil {
			return fmt.Errorf("read email: %w", err)
		}

		fmt.Print("Password: ")
		pw1, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Print("Confirm password: ")
		pw2, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()
		if string(pw1) != string(pw2) {
			return fmt.Errorf("passwords do not match")
		}

		ctx := context.Background()
		if err := authUC.Register(ctx, email, string(pw1)); err != nil {
			return fmt.Errorf("register failed: %w", err)
		}
		fmt.Println("Registered and logged in.")
		return nil
	},
}
