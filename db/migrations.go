// Package db embeds the PostgreSQL schema migrations used by every installation.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
