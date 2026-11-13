package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ResetPostgres removes only the AIDEX schema. Callers hold the installation
// lock and validate document boundaries before invoking explicit maintenance.
// The private engine remains reusable; no filesystem document is removed.
func ResetPostgres(ctx context.Context, state string) error {
	database, err := Open(ctx, state)
	if err != nil {
		return err
	}
	defer database.Close()
	_, err = database.Writer.ExecContext(ctx, "DROP SCHEMA "+pgx.Identifier{database.schema}.Sanitize()+" CASCADE")
	return err
}

var postgresRelation = regexp.MustCompile(`^(?:[0-9]+(?:_(?:fsm|vm|init))?(?:\.[0-9]+)?|t[0-9]+_[0-9]+(?:_(?:fsm|vm|init))?(?:\.[0-9]+)?)$`)
var postgresSegment = regexp.MustCompile(`^[0-9A-F]{4,24}(?:\.(?:ready|done|history|backup))?$`)

// IsPostgresMetadata recognizes only PostgreSQL's reserved cluster layout.
// Unknown files are preserved by the existing uninstall document archive.
func IsPostgresMetadata(relative string) bool {
	relative = filepath.ToSlash(relative)
	switch relative {
	case "installation.json", "server.log", "service.lock":
		return true
	}
	if !strings.HasPrefix(relative, "data/") {
		return false
	}
	path := strings.TrimPrefix(relative, "data/")
	switch path {
	case "PG_VERSION", "postgresql.conf", "postgresql.auto.conf", "pg_hba.conf", "pg_ident.conf", "postmaster.opts", "global/pg_control":
		return true
	}
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if len(parts) == 2 && parts[0] == "global" || len(parts) == 3 && parts[0] == "base" && postgresRelation.MatchString(parts[1]) {
		return name == "PG_VERSION" || name == "pg_filenode.map" || name == "pg_internal.init" || postgresRelation.MatchString(name)
	}
	if len(parts) == 2 {
		switch parts[0] {
		case "pg_wal", "pg_xact", "pg_subtrans", "pg_notify", "pg_serial", "pg_commit_ts":
			return postgresSegment.MatchString(name)
		}
	}
	if len(parts) == 3 && (parts[0] == "pg_multixact" && (parts[1] == "members" || parts[1] == "offsets") || parts[0] == "pg_wal" && parts[1] == "archive_status") {
		return postgresSegment.MatchString(name)
	}
	return false
}

func PostgresPresent(state string) bool {
	if os.Getenv("AIDEX_TEST_POSTGRES_DSN") != "" {
		return true
	}
	_, err := os.Lstat(filepath.Join(state, "postgresql"))
	return !os.IsNotExist(err)
}

// RejectUnknownDatabaseFiles prevents a former database or user file from
// silently being mistaken for a new installation. Never converts legacy data.
func RejectUnknownDatabaseFiles(state string) error {
	for _, name := range []string{"documental.db", "documental.db-wal", "documental.db-shm"} {
		if _, err := os.Lstat(filepath.Join(state, name)); !os.IsNotExist(err) {
			return fmt.Errorf("unrecognized database state; existing files preserved")
		}
	}
	return nil
}
