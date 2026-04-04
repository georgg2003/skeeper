package cli

import (
	"context"
	"fmt"
	"syscall"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addCardCmd = &cobra.Command{
	Use:   "add-card",
	Short: "Add an encrypted bank card entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		if secretUC == nil {
			return fmt.Errorf("client not initialized")
		}
		var name, notes string
		fmt.Print("Entry name: ")
		if _, err := fmt.Scanln(&name); err != nil {
			return err
		}
		fmt.Print("Notes (optional): ")
		_, _ = fmt.Scanln(&notes)

		card := models.CardPayload{}
		fmt.Print("Cardholder: ")
		if _, err := fmt.Scanln(&card.Holder); err != nil {
			return err
		}
		fmt.Print("Number: ")
		if _, err := fmt.Scanln(&card.Number); err != nil {
			return err
		}
		fmt.Print("Expiry (MM/YY): ")
		if _, err := fmt.Scanln(&card.Expiry); err != nil {
			return err
		}
		fmt.Print("CVC: ")
		cvc, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()
		card.CVC = string(cvc)

		fmt.Print("Master password: ")
		masterBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetCard(ctx, meta, card, string(masterBytes)); err != nil {
			return fmt.Errorf("save card: %w", err)
		}
		fmt.Println("Encrypted card saved locally (run `sync` to upload).")
		return nil
	},
}
