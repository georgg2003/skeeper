package clientmigrate

import (
	"embed"
)

//go:embed *.sql
var ClientMigrationsFS embed.FS
