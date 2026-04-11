package delivery

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func (d *Delivery) List(cmd *cobra.Command, args []string) error {
	entries, err := d.secret.ListLocal(context.Background())
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
		_, _ = fmt.Fprintf(out, "%s  %-8s  %s%s\n", e.UUID.String(), DisplayType(e.Type), e.UpdatedAt.Format("2006-01-02 15:04"), dirty)
	}
	return nil
}
