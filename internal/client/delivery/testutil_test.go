package delivery

import (
	"bytes"
	"context"
	"io"

	"github.com/spf13/cobra"
)

func testCmd(stdin io.Reader) (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetIn(stdin)
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	return cmd, &out
}
