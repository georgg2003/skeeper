// Command skeepercli is the desktop CLI: local encrypted vault, talks to auther and skeeper over gRPC.
package main

import (
	"github.com/georgg2003/skeeper/internal/client/cli"
)

// Injected via: -ldflags "-X main.version=... -X main.buildTime=..."
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cli.Version = version
	cli.BuildTime = buildTime
	cli.Execute()
}
