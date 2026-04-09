package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var addTextCmd = &cobra.Command{
	Use:   "add-text",
	Short: "Add arbitrary encrypted text",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "")
		if err != nil {
			return err
		}
		body, err := promptMultilineText(cmd, "Text body (empty line ends input; on a TTY you can still use EOF / Ctrl-D):")
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.SetText(ctx, meta, body, masterStr); err != nil {
			return fmt.Errorf("save text: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted text saved locally (run `sync` to upload).")
		return nil
	},
}
