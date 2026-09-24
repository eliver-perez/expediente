// Package db embeds only implemented migrations, never schema.proposed.sql.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
