package delivery

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) UpdatePassword(cmd *cobra.Command, args []string) error {
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
	if err := d.secret.UpdatePassword(ctx, id, meta, secretStr, masterStr); err != nil {
		return fmt.Errorf("update password entry: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
	return nil
}

func (d *Delivery) UpdateText(cmd *cobra.Command, args []string) error {
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
	if err := d.secret.UpdateText(ctx, id, meta, body, masterStr); err != nil {
		return fmt.Errorf("update text entry: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
	return nil
}

func (d *Delivery) UpdateCard(cmd *cobra.Command, args []string) error {
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
	if err := d.secret.UpdateCard(ctx, id, meta, card, masterStr); err != nil {
		return fmt.Errorf("update card entry: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
	return nil
}

func (d *Delivery) UpdateFile(cmd *cobra.Command, args []string) error {
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
	if err := d.secret.UpdateFile(ctx, id, meta, masterStr, replace, data, orig); err != nil {
		return fmt.Errorf("update file entry: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry updated locally (run `sync` to upload).")
	return nil
}
