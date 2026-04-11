// Command skeepercli is the desktop CLI: local encrypted vault, talks to auther and skeeper over gRPC.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/georgg2003/skeeper/internal/client/delivery/cli"
)

// version and buildTime are injected with: -ldflags "-X main.version=... -X main.buildTime=..."
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	app, err := cli.New(cli.Config{
		Version:   version,
		BuildTime: buildTime,
		Wire:      BuildDelivery,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app.ExecuteContext(ctx)
}
