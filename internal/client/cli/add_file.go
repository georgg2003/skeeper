package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func runAddFile(cmd *cobra.Command, args []string) error {
	if err := requireSecretUC(); err != nil {
		return err
	}
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	meta, err := promptEntryMetadata(cmd, "")
	if err != nil {
		return err
	}
	masterStr, err := promptMasterPassword(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()
	origName := filepath.Base(path)
	if err := secretUC.SetFile(ctx, meta, origName, data, masterStr); err != nil {
		return fmt.Errorf("save file: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted file saved locally (ciphertext in vault payload; run `sync` to upload).")
	return nil
}

var addFileCmd = &cobra.Command{
	Use:   "add-file PATH",
	Short: "Encrypt a small file; ciphertext is stored in the entry payload (syncs to the server)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddFile,
}
