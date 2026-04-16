package main

import (
	"github.com/georgg2003/skeeper/internal/client/delivery"
	"github.com/georgg2003/skeeper/internal/client/delivery/cli"
)

// Compile-time check: real delivery implements the CLI boundary.
var _ cli.Handlers = (*delivery.Delivery)(nil)
