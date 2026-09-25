package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/db"
	"gestor-documental/internal/domain"
	_ "modernc.org/sqlite"
)

type Database struct {
	Writer *sql.DB
	Reader *sql.DB
}

func Open(ctx context.Context, stateDirectory string) (*Database, error) {
	if err := PreparePrivateDirectory(stateDirectory); err != nil {
		return nil, err
	}
	if err := requireLocalFilesystem(stateDirectory); err != nil {
		return nil, err
	}
	databasePath := filepath.Join(stateDirectory, "documental.db")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if info, err := os.Lstat(databasePath + suffix); err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
			return nil, fmt.Errorf("database path must be a regular local file")
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	databaseURL := url.URL{Scheme: "file", Path: filepath.ToSlash(databasePath)}
	settings := url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(5000)", "synchronous(FULL)"}, "_txlock": {"immediate"}}
	databaseURL.RawQuery = settings.Encode()
	writer, err := sql.Open("sqlite", databaseURL.String())
	if err != nil {
		return nil, err
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	fail := func(cause error) (*Database, error) { writer.Close(); return nil, cause }
	var journalMode string
	if err := writer.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		return fail(err)
	}
	if journalMode != "wal" {
		return fail(fmt.Errorf("SQLite WAL unavailable"))
	}
	var engineVersion string
	if err := writer.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&engineVersion); err != nil {
		return fail(err)
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(engineVersion, "%d.%d.%d", &major, &minor, &patch); err != nil || major < 3 || (major == 3 && (minor < 51 || (minor == 51 && patch < 3))) {
		return fail(fmt.Errorf("SQLite engine must include the WAL fix (3.51.3 or newer)"))
	}
	if err := ProtectPrivatePath(databasePath, false); err != nil {
		return fail(err)
	}
	// Probe the embedded engine, not the unrelated sqlite3 command installed on the host.
	if _, err := writer.ExecContext(ctx, "CREATE VIRTUAL TABLE temp.fts_probe USING fts5(body); DROP TABLE temp.fts_probe;"); err != nil {
		return fail(err)
	}
	if err := migrate(ctx, writer); err != nil {
		return fail(err)
	}
	settings.Set("mode", "ro")
	settings.Del("_txlock")
	settings["_pragma"] = []string{"foreign_keys(1)", "busy_timeout(5000)", "query_only(1)"}
	databaseURL.RawQuery = settings.Encode()
	reader, err := sql.Open("sqlite", databaseURL.String())
	if err != nil {
		return fail(err)
	}
	reader.SetMaxOpenConns(4)
	reader.SetMaxIdleConns(4)
	if err := reader.PingContext(ctx); err != nil {
		reader.Close()
		return fail(err)
	}
	return &Database{Writer: writer, Reader: reader}, nil
}

func (database *Database) Close() error {
	readerError := database.Reader.Close()
	writerError := database.Writer.Close()
	if readerError != nil {
		return readerError
	}
	return writerError
}

func (database *Database) Write(ctx context.Context, operation func(*sql.Tx) error) error {
	transaction, err := database.Writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if err := operation(transaction); err != nil {
		return err
	}
	return transaction.Commit()
}

// RollbackEmpty is deliberately restricted to a never-used installation.
// Once there is any user or audit evidence, recovery requires a consistent backup.
func (database *Database) RollbackEmpty(ctx context.Context) error {
	return database.Write(ctx, func(transaction *sql.Tx) error {
		var count int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil {
			return err
		}
		if count == 4 {
			contents, err := db.Migrations.ReadFile("migrations/0004_document_workflow.down.sql")
			if err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, string(contents)); err != nil {
				return fmt.Errorf("rollback refused: H5 contains data")
			}
			count--
		}
		if count == 3 {
			contents, err := db.Migrations.ReadFile("migrations/0003_managed_organization.down.sql")
			if err != nil {
				return err
			}
			if _, err := transaction.ExecContext(ctx, string(contents)); err != nil {
				return fmt.Errorf("rollback refused: H4 contains data")
			}
			count--
		}
		if count == 2 {
			contents, err := db.Migrations.ReadFile("migrations/0002_linked_libraries.down.sql")
			if err != nil {
				return err
			}
			if _, err := transaction.ExecContext(ctx, string(contents)); err != nil {
				return fmt.Errorf("rollback refused: H3 contains data")
			}
		} else if count != 1 {
			return fmt.Errorf("empty rollback is only supported for the initial H2 schema")
		}
		contents, err := db.Migrations.ReadFile("migrations/0001_identity.down.sql")
		if err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, string(contents)); err != nil {
			return fmt.Errorf("rollback refused: database has data or cannot be safely reversed")
		}
		return nil
	})
}

func migrate(ctx context.Context, connection *sql.DB) error {
	transaction, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL) STRICT`); err != nil {
		return err
	}
	entries, err := db.Migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	known := make(map[string]bool)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		known[entry.Name()] = true
		contents, err := db.Migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(contents))
		var previousChecksum string
		err = transaction.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE name = ?", entry.Name()).Scan(&previousChecksum)
		if err == nil {
			if previousChecksum != checksum {
				return fmt.Errorf("migration checksum mismatch: %s", entry.Name())
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err := transaction.ExecContext(ctx, string(contents)); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO schema_migrations VALUES (?, ?, ?)", entry.Name(), checksum, domain.Timestamp(time.Now())); err != nil {
			return err
		}
	}
	rows, err := transaction.QueryContext(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		if !known[name] {
			return fmt.Errorf("database belongs to a newer release")
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return transaction.Commit()
}
