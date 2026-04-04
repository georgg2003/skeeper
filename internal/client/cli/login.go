package cli

import (
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Auther service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if authUC == nil {
			return fmt.Errorf("client not initialized")
		}
		var username string
		fmt.Print("Email: ")
		if _, err := fmt.Scanln(&username); err != nil {
			return fmt.Errorf("read email: %w", err)
		}

		fmt.Print("Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password := string(bytePassword)
		fmt.Println()

		ctx := context.Background()
		if err := authUC.Login(ctx, username, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Println("Successfully logged in.")
		return nil
	},
}
