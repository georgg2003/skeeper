package delivery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func (d *Delivery) AddPassword(cmd *cobra.Command, args []string) error {
	meta, err := promptEntryMetadata(cmd, "Entry name (e.g. Gmail): ")
	if err != nil {
		return err
	}
	secretStr, err := promptPasswordValue(cmd, "Password to store: ")
	if err != nil {
		return err
	}
	masterStr, err := promptMasterPassword(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if err := d.secret.SetPassword(ctx, meta, secretStr, masterStr); err != nil {
		return fmt.Errorf("save secret: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted entry saved locally (run `sync` to upload).")
	return nil
}

func (d *Delivery) AddText(cmd *cobra.Command, args []string) error {
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
	ctx := cmd.Context()
	if err := d.secret.SetText(ctx, meta, body, masterStr); err != nil {
		return fmt.Errorf("save text: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted text saved locally (run `sync` to upload).")
	return nil
}

func (d *Delivery) AddFile(cmd *cobra.Command, args []string) error {
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
	ctx := cmd.Context()
	origName := filepath.Base(path)
	if err := d.secret.SetFile(ctx, meta, origName, data, masterStr); err != nil {
		return fmt.Errorf("save file: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted file saved locally (ciphertext in vault payload; run `sync` to upload).")
	return nil
}

func (d *Delivery) AddCard(cmd *cobra.Command, args []string) error {
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
	ctx := cmd.Context()
	if err := d.secret.SetCard(ctx, meta, card, masterStr); err != nil {
		return fmt.Errorf("save card: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted card saved locally (run `sync` to upload).")
	return nil
}
