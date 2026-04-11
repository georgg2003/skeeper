package delivery

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) Delete(cmd *cobra.Command, args []string) error {
	id, err := parseUUIDArg(args[0])
	if err != nil {
		return err
	}
	masterStr, err := promptMasterPassword(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if err := d.secret.DeleteEntry(ctx, id, masterStr); err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Entry deleted locally (run `sync` to upload).")
	return nil
}
