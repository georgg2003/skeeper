// Package clientmigrate embeds goose SQL for the CLI's local SQLite schema.
package clientmigrate

import (
	"embed"
)

//go:embed *.sql
var ClientMigrationsFS embed.FS
