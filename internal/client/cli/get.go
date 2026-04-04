package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/spf13/cobra"
	"github.com/google/uuid"
	"golang.org/x/term"
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

		fmt.Print("Master password: ")
		masterBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()

		payload, meta, err := secretUC.GetDecryptedEntry(ctx, id, string(masterBytes))
		if err != nil {
			return err
		}

		fmt.Printf("Name: %s\n", meta.Name)
		if meta.Notes != "" {
			fmt.Printf("Notes: %s\n", meta.Notes)
		}
		fmt.Printf("Type: %s\n", row.Type)

		switch row.Type {
		case models.EntryTypePassword:
			fmt.Printf("Password: %s\n", string(payload))
		case models.EntryTypeText:
			fmt.Printf("Text:\n%s\n", string(payload))
		case models.EntryTypeBinary:
			outPath := fmt.Sprintf("%s.bin", id.String())
			if err := os.WriteFile(outPath, payload, 0o600); err != nil {
				return err
			}
			fmt.Printf("Wrote %d bytes to %s\n", len(payload), outPath)
		case models.EntryTypeCard:
			var c models.CardPayload
			if err := json.Unmarshal(payload, &c); err != nil {
				return err
			}
			fmt.Printf("Holder: %s\nNumber: %s\nExpiry: %s\nCVC: %s\n", c.Holder, c.Number, c.Expiry, c.CVC)
		default:
			fmt.Printf("Raw payload (%d bytes)\n", len(payload))
		}
		return nil
	},
}
