package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/usecase"
)

var addTextCmd = &cobra.Command{
	Use:   "add-text",
	Short: "Add arbitrary encrypted text",
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

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Text body (empty line ends input; on a TTY you can still use EOF / Ctrl-D):")
		var b strings.Builder
		for {
			line, err := readLine(cmd)
			if err != nil {
				return err
			}
			if line == "" {
				break
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(line)
		}

		writePrompt(cmd, "Master password: ")
		masterStr, err := readPasswordLine(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		meta := usecase.EntryMetadata{Name: name, Notes: notes}
		if err := secretUC.SetText(ctx, meta, strings.TrimSuffix(b.String(), "\n"), masterStr); err != nil {
			return fmt.Errorf("save text: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Encrypted text saved locally (run `sync` to upload).")
		return nil
	},
}
