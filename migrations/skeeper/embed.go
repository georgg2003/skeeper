// Package skeepermigrate ships embedded goose migrations for the Skeeper Postgres DB.
package skeepermigrate

import "embed"

//go:embed *.sql
var GooseFiles embed.FS
