package delivery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func (d *Delivery) Get(cmd *cobra.Command, args []string) error {
	id, err := parseUUIDArg(args[0])
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	masterStr, err := promptMasterPassword(cmd)
	if err != nil {
		return err
	}
	entry, err := d.secret.GetDecryptedEntry(ctx, id, masterStr)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Name: %s\n", entry.Meta.Name)
	if entry.Meta.Notes != "" {
		_, _ = fmt.Fprintf(out, "Notes: %s\n", entry.Meta.Notes)
	}
	_, _ = fmt.Fprintf(out, "Type: %s\n", entry.Type)
	switch entry.Type {
	case models.EntryTypePassword:
		_, _ = fmt.Fprintf(out, "Password: %s\n", string(entry.Payload))
	case models.EntryTypeText:
		_, _ = fmt.Fprintf(out, "Text:\n%s\n", string(entry.Payload))
	case models.EntryTypeFile:
		base := filepath.Base(entry.Meta.OriginalFilename)
		if base == "." || base == string(filepath.Separator) || base == "" {
			base = "file"
		}
		outPath := fmt.Sprintf("%s_%s", id.String(), base)
		if err := os.WriteFile(outPath, entry.Payload, 0o600); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Wrote %d bytes to %s\n", len(entry.Payload), outPath)
	case models.EntryTypeCard:
		var c models.CardPayload
		if err := json.Unmarshal(entry.Payload, &c); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Holder: %s\nNumber: %s\nExpiry: %s\nCVC: %s\n", c.Holder, c.Number, c.Expiry, c.CVC)
	default:
		_, _ = fmt.Fprintf(out, "Raw payload (%d bytes)\n", len(entry.Payload))
	}
	return nil
}
