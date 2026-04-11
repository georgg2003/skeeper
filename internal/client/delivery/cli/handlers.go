package cli

import "github.com/spf13/cobra"

// Handlers is the command surface the CLI calls. [*delivery.Delivery] implements it; tests use a mock.
type Handlers interface {
	Login(cmd *cobra.Command, args []string) error
	Register(cmd *cobra.Command, args []string) error
	Logout(cmd *cobra.Command, args []string) error
	Sync(cmd *cobra.Command, args []string) error
	AddPassword(cmd *cobra.Command, args []string) error
	AddText(cmd *cobra.Command, args []string) error
	AddFile(cmd *cobra.Command, args []string) error
	AddCard(cmd *cobra.Command, args []string) error
	UpdatePassword(cmd *cobra.Command, args []string) error
	UpdateText(cmd *cobra.Command, args []string) error
	UpdateCard(cmd *cobra.Command, args []string) error
	UpdateFile(cmd *cobra.Command, args []string) error
	Delete(cmd *cobra.Command, args []string) error
	List(cmd *cobra.Command, args []string) error
	Get(cmd *cobra.Command, args []string) error
}
