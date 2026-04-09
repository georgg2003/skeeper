package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local entries (uuid, type, updated time; ciphertext only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireSecretUC(); err != nil {
			return err
		}
		entries, err := secretUC.ListLocal(context.Background())
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(entries) == 0 {
			_, _ = fmt.Fprintln(out, "No entries.")
			return nil
		}
		for _, e := range entries {
			dirty := ""
			if e.IsDirty {
				dirty = " (dirty)"
			}
			_, _ = fmt.Fprintf(out, "%s  %-8s  %s%s\n", e.UUID.String(), displayType(e.Type), e.UpdatedAt.Format("2006-01-02 15:04"), dirty)
		}
		return nil
	},
}

func displayType(t string) string {
	switch t {
	case models.EntryTypePassword:
		return "password"
	case models.EntryTypeText:
		return "text"
	case models.EntryTypeFile:
		return "file"
	case models.EntryTypeCard:
		return "card"
	default:
		return t
	}
}
