// Package authermigrate exposes embedded Goose SQL files for the Auther Postgres database.
package authermigrate

import "embed"

// GooseFiles contains ordered *.sql migrations from this directory.
//
//go:embed *.sql
var GooseFiles embed.FS
