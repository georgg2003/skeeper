// Package authermigrate ships embedded goose migrations for the Auther Postgres DB.
package authermigrate

import "embed"

//go:embed *.sql
var GooseFiles embed.FS
