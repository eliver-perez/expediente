package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"
)

// A SQLite snapshot before the first additive processing migration. Open is called
// under the application's exclusive state lock. VACUUM INTO includes committed WAL.
func backupBeforeProcessingUpgrade(ctx context.Context, connection *sql.DB, stateDirectory string) error {
	var exists int
	if err := connection.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return nil
	}
	var old, current int
	if err := connection.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(name='0009_document_formats.up.sql'),0) FROM schema_migrations").Scan(&old, &current); err != nil {
		return err
	}
	if old == 0 || current > 0 {
		return nil
	}
	directory := filepath.Join(stateDirectory, "upgrade-backups")
	if err := PreparePrivateDirectory(directory); err != nil {
		return err
	}
	destination := filepath.Join(directory, "before-processing-"+time.Now().UTC().Format("20060102T150405.000000000")+".db")
	if _, err := connection.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		return err
	}
	return ProtectPrivatePath(destination, false)
}
