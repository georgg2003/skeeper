// Package skeepermigrate exposes embedded Goose SQL files for the Skeeper Postgres database.
package skeepermigrate

import "embed"

// GooseFiles contains ordered *.sql migrations from this directory.
//
//go:embed *.sql
var GooseFiles embed.FS
