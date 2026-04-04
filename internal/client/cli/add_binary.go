package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
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

		writePrompt(cmd, "Entry name: ")
		name, err := readLine(cmd)
		if err != nil {
			return err
		}
		writePrompt(cmd, "Notes (optional): ")
		notes, _ := readLine(cmd)

		writePrompt(cmd, "Master password: ")
		masterStr, err := readPasswordLine(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetBinary(ctx, meta, data, masterStr); err != nil {
			return fmt.Errorf("save binary: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted binary saved locally (run `sync` to upload).")
		return nil
	},
}
