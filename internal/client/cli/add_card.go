package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
)

var addCardCmd = &cobra.Command{
	Use:   "add-card",
	Short: "Add an encrypted bank card entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if secretUC == nil {
			return fmt.Errorf("client not initialized")
		}
		writePrompt(cmd, "Entry name: ")
		name, err := readLine(cmd)
		if err != nil {
			return err
		}
		writePrompt(cmd, "Notes (optional): ")
		notes, _ := readLine(cmd)

		card := models.CardPayload{}
		writePrompt(cmd, "Cardholder: ")
		card.Holder, err = readLine(cmd)
		if err != nil {
			return err
		}
		writePrompt(cmd, "Number: ")
		card.Number, err = readLine(cmd)
		if err != nil {
			return err
		}
		writePrompt(cmd, "Expiry (MM/YY): ")
		card.Expiry, err = readLine(cmd)
		if err != nil {
			return err
		}
		writePrompt(cmd, "CVC: ")
		card.CVC, err = readPasswordLine(cmd)
		if err != nil {
			return err
		}

		writePrompt(cmd, "Master password: ")
		masterStr, err := readPasswordLine(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetCard(ctx, meta, card, masterStr); err != nil {
			return fmt.Errorf("save card: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted card saved locally (run `sync` to upload).")
		return nil
	},
}
