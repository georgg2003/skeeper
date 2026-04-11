package delivery

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"
)

func testCmd(stdin io.Reader) (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(stdin)
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, &out
}
