// Package diagnostics stores bounded operational events, never arbitrary errors.
package diagnostics

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

// A closed set of structured fields prevents exception text, filenames, request
// bodies, headers, credentials and document content from entering support logs.
type Context struct {
	LibraryID  string `json:"library_id,omitempty"`
	RootID     string `json:"root_id,omitempty"`
	JobID      string `json:"job_id,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Attempt    int    `json:"attempt,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

func (c Context) Safe() Context {
	for _, value := range []*string{&c.LibraryID, &c.RootID, &c.JobID, &c.DocumentID, &c.RequestID} {
		if !identifier.MatchString(*value) {
			*value = ""
		}
	}
	switch c.Operation {
	case "scan", "extract", "ocr", "materialize", "verify_managed", "preview", "http", "watch", "startup":
	default:
		c.Operation = ""
	}
	if c.Attempt < 0 || c.Attempt > 100 {
		c.Attempt = 0
	}
	if c.HTTPStatus < 400 || c.HTTPStatus > 599 {
		c.HTTPStatus = 0
	}
	return c
}
func Module(value string) string {
	switch value {
	case "processing", "scan", "preview", "watcher", "http", "system", "licensing":
		return value
	}
	return "system"
}
func Severity(code string) string {
	switch code {
	case "SERVICE_STARTED":
		return "info"
	case "HTTP_PANIC", "PROCESSING_INTERNAL_ERROR", "STORAGE_INTEGRITY_FAILED":
		return "critical"
	case "PROCESS_RESTARTED", "WATCH_LIMIT", "WATCH_EVENTS_LOST", "FILE_UNSTABLE", "PROCESS_TIMEOUT", "PREVIEW_TIMEOUT", "PREVIEW_CONVERTER_UNAVAILABLE", "PREVIEW_CONVERTER_MISSING", "PREVIEW_OBSOLETE", "LICENSE_FEATURE", "LICENSE_READ_ONLY", "LICENSE_RECOVERY_REQUIRED":
		return "warning"
	}
	return "error"
}

type Policy struct {
	Days     int   `json:"retention_days"`
	Maximum  int   `json:"maximum_events"`
	Revision int64 `json:"revision"`
}

func ReadPolicy(ctx context.Context, q storage.Querier) (Policy, error) {
	var p Policy
	err := q.QueryRowContext(ctx, "SELECT retention_days,maximum_events,revision FROM diagnostic_policy WHERE singleton=1").Scan(&p.Days, &p.Maximum, &p.Revision)
	return p, err
}
func PruneTx(ctx context.Context, tx *sql.Tx, instant time.Time) error {
	p, err := ReadPolicy(ctx, tx)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM diagnostic_events WHERE last_occurred_at<?", domain.Timestamp(instant.Add(-time.Duration(p.Days)*24*time.Hour))); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM diagnostic_events WHERE id IN (SELECT id FROM diagnostic_events ORDER BY last_occurred_at DESC,id DESC LIMIT -1 OFFSET ?)`, p.Maximum)
	return err
}
func Prune(ctx context.Context, db *storage.Database) error {
	return db.Write(ctx, func(tx *sql.Tx) error { return PruneTx(ctx, tx, time.Now()) })
}
func Record(ctx context.Context, db *storage.Database, module, code string, details Context) error {
	return db.Write(ctx, func(tx *sql.Tx) error { return RecordTx(ctx, tx, module, code, details) })
}
func RecordTx(ctx context.Context, tx *sql.Tx, module, code string, details Context) error {
	module = Module(module)
	message, known := Messages[code]
	if !known {
		code = "INTERNAL_ERROR"
		message = Messages[code]
	}
	details = details.Safe()
	raw, _ := json.Marshal(details)
	// Repeated failures for the same target become occurrences, not an unbounded
	// stream of identical rows. Request/attempt identify the latest occurrence only.
	group := details
	group.Attempt = 0
	group.RequestID = ""
	key, _ := json.Marshal(group)
	signature := domain.Digest(module + ":" + code + ":" + string(key))
	instant := time.Now()
	at := domain.Timestamp(instant)
	_, err := tx.ExecContext(ctx, `INSERT INTO diagnostic_events(id,signature,occurred_at,last_occurred_at,module,code,severity,message,context_json) VALUES(?,?,?,?,?,?,?,?,?)
 ON CONFLICT(signature) DO UPDATE SET last_occurred_at=excluded.last_occurred_at,status='open',context_json=excluded.context_json,occurrences=diagnostic_events.occurrences+1,revision=diagnostic_events.revision+1`, domain.NewID(), signature, at, at, module, code, Severity(code), message, string(raw))
	if err != nil {
		return err
	}
	return PruneTx(ctx, tx, instant)
}
