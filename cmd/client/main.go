// Command gophkeeper is the GophKeeper CLI client (Windows, Linux, macOS).
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
