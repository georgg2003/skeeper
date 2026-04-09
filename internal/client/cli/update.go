package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Change an existing vault entry (increments version for sync)",
}

var updatePasswordCmd = &cobra.Command{
	Use:   "password UUID",
	Short: "Update a PASSWORD entry (new secret + name/notes)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		id, err := parseUUIDArg(args[0])
		if err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "")
		if err != nil {
			return err
		}
		secretStr, err := promptPasswordValue(cmd, "New password to store: ")
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.UpdatePassword(ctx, id, meta, secretStr, masterStr); err != nil {
			return fmt.Errorf("update password entry: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
		return nil
	},
}

var updateTextCmd = &cobra.Command{
	Use:   "text UUID",
	Short: "Update a TEXT entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		id, err := parseUUIDArg(args[0])
		if err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "")
		if err != nil {
			return err
		}
		body, err := promptMultilineText(cmd, "New text (empty line ends input):")
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.UpdateText(ctx, id, meta, body, masterStr); err != nil {
			return fmt.Errorf("update text entry: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
		return nil
	},
}

var updateCardCmd = &cobra.Command{
	Use:   "card UUID",
	Short: "Update a CARD entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		id, err := parseUUIDArg(args[0])
		if err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "")
		if err != nil {
			return err
		}
		card, err := promptCard(cmd)
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.UpdateCard(ctx, id, meta, card, masterStr); err != nil {
			return fmt.Errorf("update card entry: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
		return nil
	},
}

var updateFileCmd = &cobra.Command{
	Use:   "file UUID",
	Short: "Update a FILE entry (metadata only, or replace bytes from a path)",
	Long: "Updates name/notes. Optionally replace file content: enter a path to read new bytes, or leave empty to keep " +
		"the existing ciphertext. Sync uses the same versioned last-write-wins rule as other entry types: the server stores " +
		"one encrypted blob per entry—conflicting edits are not merged at the binary level.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		id, err := parseUUIDArg(args[0])
		if err != nil {
			return err
		}
		meta, err := promptEntryMetadata(cmd, "")
		if err != nil {
			return err
		}
		data, orig, replace, err := promptOptionalFilePath(cmd, "Path to new file (empty = keep stored file bytes, update name/notes only): ")
		if err != nil {
			return err
		}
		masterStr, err := promptMasterPassword(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err := secretUC.UpdateFile(ctx, id, meta, masterStr, replace, data, orig); err != nil {
			return fmt.Errorf("update file entry: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
		return nil
	},
}

func init() {
	updateCmd.AddCommand(updatePasswordCmd, updateTextCmd, updateCardCmd, updateFileCmd)
}
