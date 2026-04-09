package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

var getCmd = &cobra.Command{
	Use:   "get UUID",
	Short: "Decrypt and print one entry (requires master password)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if secretUC == nil {
			return fmt.Errorf("client not initialized")
		}
		id, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid uuid: %w", err)
		}

		ctx := context.Background()
		row, err := secretUC.GetLocalEntry(ctx, id)
		if err != nil {
			return err
		}

		writePrompt(cmd, "Master password: ")
		masterStr, err := readPasswordLine(cmd)
		if err != nil {
			return err
		}

		payload, meta, origFileName, err := secretUC.GetDecryptedEntry(ctx, id, masterStr)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(out, "Name: %s\n", meta.Name)
		if meta.Notes != "" {
			_, _ = fmt.Fprintf(out, "Notes: %s\n", meta.Notes)
		}
		_, _ = fmt.Fprintf(out, "Type: %s\n", row.Type)

		switch row.Type {
		case models.EntryTypePassword:
			_, _ = fmt.Fprintf(out, "Password: %s\n", string(payload))
		case models.EntryTypeText:
			_, _ = fmt.Fprintf(out, "Text:\n%s\n", string(payload))
		case models.EntryTypeBinary:
			outPath := fmt.Sprintf("%s.bin", id.String())
			if err := os.WriteFile(outPath, payload, 0o600); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(out, "Wrote %d bytes to %s\n", len(payload), outPath)
		case models.EntryTypeFile:
			base := filepath.Base(origFileName)
			if base == "." || base == string(filepath.Separator) || base == "" {
				base = "file"
			}
			outPath := fmt.Sprintf("%s_%s", id.String(), base)
			if err := os.WriteFile(outPath, payload, 0o600); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(out, "Wrote %d bytes to %s\n", len(payload), outPath)
		case models.EntryTypeCard:
			var c models.CardPayload
			if err := json.Unmarshal(payload, &c); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(out, "Holder: %s\nNumber: %s\nExpiry: %s\nCVC: %s\n", c.Holder, c.Number, c.Expiry, c.CVC)
		default:
			_, _ = fmt.Fprintf(out, "Raw payload (%d bytes)\n", len(payload))
		}
		return nil
	},
}
