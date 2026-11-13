package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"gestor-documental/db"
	"gestor-documental/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Database struct {
	Writer *sql.DB
	Reader *sql.DB
	// Dedicated capacity keeps analytics away from authentication and progress.
	ControlReader   *sql.DB
	StatusReader    *sql.DB
	AnalyticsReader *sql.DB
	dsn             string
	schema          string
	stop            func() error
}

func Open(ctx context.Context, stateDirectory string) (*Database, error) {
	if err := PrepareLocalPrivateDirectory(stateDirectory); err != nil {
		return nil, err
	}
	if err := RejectUnknownDatabaseFiles(stateDirectory); err != nil {
		return nil, err
	}
	dsn, schema, stop, err := preparePostgres(ctx, stateDirectory)
	if err != nil {
		return nil, err
	}
	result, err := openDatabase(ctx, dsn, schema)
	if err != nil {
		if stop != nil {
			_ = stop()
		}
		return nil, err
	}
	result.stop = stop
	return result, nil
}

func openDatabase(ctx context.Context, dsn, schema string) (*Database, error) {
	return connectDatabase(ctx, dsn, schema, true)
}

func connectDatabase(ctx context.Context, dsn, schema string, initialize bool) (*Database, error) {
	writer, err := openPostgresPool(dsn, schema, false, 12)
	if err != nil {
		return nil, err
	}
	database := &Database{Writer: writer, dsn: dsn, schema: schema}
	fail := func(err error) (*Database, error) { database.Close(); return nil, err }
	if err = writer.PingContext(ctx); err != nil {
		return fail(fmt.Errorf("PostgreSQL unavailable: %w", err))
	}
	var version int
	if err = writer.QueryRowContext(ctx, "SHOW server_version_num").Scan(&version); err != nil {
		return fail(err)
	}
	if version < 180000 {
		return fail(fmt.Errorf("AIDEX requires PostgreSQL 18 or newer"))
	}
	if initialize {
		if _, err = writer.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{schema}.Sanitize()); err != nil {
			return fail(err)
		}
		if err = migrate(ctx, writer); err != nil {
			return fail(err)
		}
	}
	for _, entry := range []struct {
		target **sql.DB
		limit  int
	}{{&database.Reader, 6}, {&database.ControlReader, 2}, {&database.StatusReader, 2}, {&database.AnalyticsReader, 2}} {
		*entry.target, err = openPostgresPool(dsn, schema, true, entry.limit)
		if err != nil {
			return fail(err)
		}
		if err = (*entry.target).PingContext(ctx); err != nil {
			return fail(err)
		}
	}
	return database, nil
}

func (database *Database) Close() error {
	var first error
	for _, pool := range []*sql.DB{database.AnalyticsReader, database.StatusReader, database.ControlReader, database.Reader, database.Writer} {
		if pool != nil {
			if err := pool.Close(); first == nil {
				first = err
			}
		}
	}
	if database.stop != nil {
		if err := database.stop(); first == nil {
			first = err
		}
		database.stop = nil
	}
	return first
}

func (database *Database) Write(ctx context.Context, operation func(*sql.Tx) error) error {
	// Serializable isolation preserves read/check/write invariants previously
	// provided by writer serialization. Only conflicting rows cause retries.
	for attempt := 0; ; attempt++ {
		tx, err := database.Writer.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return err
		}
		err = operation(tx)
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		_ = tx.Rollback()
		if err == nil {
			return nil
		}
		pgError, ok := err.(*pgconn.PgError)
		if !ok || (pgError.Code != "40001" && pgError.Code != "40P01" && pgError.Code != "23505") || attempt >= 16 {
			return err
		}
		timer := time.NewTimer(time.Duration(5+min(attempt*7, 80)+rand.IntN(20)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// A committed commercial installation is recovered through verified backups.
func (database *Database) RollbackEmpty(ctx context.Context) error {
	return fmt.Errorf("schema rollback requires a verified backup")
}

// Scratch creates an isolated schema on the same server for archive validation.
// No credentials or destination license records enter the scratch schema.
func (database *Database) Scratch(ctx context.Context) (*Database, func(), error) {
	schema := "inspect_" + strings.ReplaceAll(domain.NewID(), "-", "")
	scratch, err := openDatabase(ctx, database.dsn, schema)
	if err != nil {
		_, _ = database.Writer.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		return nil, nil, err
	}
	cleanup := func() {
		_ = scratch.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = database.Writer.ExecContext(cleanupCtx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	}
	return scratch, cleanup, nil
}

func migrate(ctx context.Context, connection *sql.DB) error {
	tx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended(current_schema(),0))"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(name TEXT PRIMARY KEY,checksum TEXT NOT NULL,applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	entries, err := db.Migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		known[entry.Name()] = true
		contents, err := db.Migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(bytes.ReplaceAll(contents, []byte("\r\n"), []byte("\n"))))
		var previous string
		err = tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE name=?", entry.Name()).Scan(&previous)
		if err == nil {
			if previous != checksum {
				return fmt.Errorf("migration checksum mismatch: %s", entry.Name())
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(contents)); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations VALUES(?,?,?)", entry.Name(), checksum, domain.Timestamp(time.Now())); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		if !known[name] {
			rows.Close()
			return fmt.Errorf("database belongs to a newer release")
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	return tx.Commit()
}

// WriteOnce uses PostgreSQL's READ COMMITTED isolation for operations protected
// by row locks or atomic UPSERTs. It never replays a callback with file effects.
func (database *Database) WriteOnce(ctx context.Context, operation func(*sql.Tx) error) error {
	tx, err := database.Writer.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = operation(tx); err != nil {
		return err
	}
	return tx.Commit()
}
