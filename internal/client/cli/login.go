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
		var username string
		fmt.Print("Enter username: ")
		fmt.Scanln(&username)

		fmt.Print("Enter password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password := string(bytePassword)
		fmt.Println()

		ctx := context.Background()
		err = authUseCase.Login(ctx, username, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Println("Successfully logged in!")
		return nil
	},
}
