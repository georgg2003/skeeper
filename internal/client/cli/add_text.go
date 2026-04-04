package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/georgg2003/skeeper/internal/client/usecase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addTextCmd = &cobra.Command{
	Use:   "add-text",
	Short: "Add arbitrary encrypted text",
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

		fmt.Println("Text body (end with EOF / Ctrl-D):")
		var b strings.Builder
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			b.WriteString(sc.Text())
			b.WriteByte('\n')
		}
		if err := sc.Err(); err != nil {
			return err
		}

		fmt.Print("Master password: ")
		masterBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetText(ctx, meta, strings.TrimSuffix(b.String(), "\n"), string(masterBytes)); err != nil {
			return fmt.Errorf("save text: %w", err)
		}
		fmt.Println("Encrypted text saved locally (run `sync` to upload).")
		return nil
	},
}
