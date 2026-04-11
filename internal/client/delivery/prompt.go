package delivery

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func writePrompt(cmd *cobra.Command, format string, args ...any) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}

func readLine(cmd *cobra.Command) (string, error) {
	return readLineFrom(cmd.InOrStdin())
}

func readLineFrom(r io.Reader) (string, error) {
	var b strings.Builder
	var buf [1]byte
	for {
		n, err := r.Read(buf[:])
		if n == 0 {
			if err == io.EOF {
				return strings.TrimSuffix(b.String(), "\r"), nil
			}
			if err != nil {
				return "", err
			}
			continue
		}
		if buf[0] == '\n' {
			return strings.TrimSuffix(b.String(), "\r"), nil
		}
		b.WriteByte(buf[0])
	}
}

func readPasswordLine(cmd *cobra.Command) (string, error) {
	in := cmd.InOrStdin()
	if f, ok := in.(*os.File); ok {
		fd := int(f.Fd())
		if term.IsTerminal(fd) {
			b, err := term.ReadPassword(fd)
			if err != nil {
				return "", err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
			return string(b), nil
		}
	}
	return readLine(cmd)
}
