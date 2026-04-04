package cli

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addBinaryCmd = &cobra.Command{
	Use:   "add-binary PATH",
	Short: "Encrypt a file and store it as a binary entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if secretUC == nil {
			return fmt.Errorf("client not initialized")
		}
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		var name, notes string
		fmt.Print("Entry name: ")
		if _, err := fmt.Scanln(&name); err != nil {
			return err
		}
		fmt.Print("Notes (optional): ")
		_, _ = fmt.Scanln(&notes)

		fmt.Print("Master password: ")
		masterBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetBinary(ctx, meta, data, string(masterBytes)); err != nil {
			return fmt.Errorf("save binary: %w", err)
		}
		fmt.Println("Encrypted binary saved locally (run `sync` to upload).")
		return nil
	},
}
