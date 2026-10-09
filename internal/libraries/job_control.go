package libraries

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
)

// A completed operation must not be abandoned in running/generating when its
// final database write fails. Retry only the journal, never extraction/conversion.
// SQLite bounds lock waits; the context cancels writes/backoff on shutdown.
// Do not impose a small deadline on startup cleanup of a large preview cache.
// Shutdown leaves restart recovery
// in charge. Log once outside SQLite because the database itself may be full.
func retryWorkerWrite(ctx context.Context, details diagnostics.Context, write func(context.Context) error) bool {
	details = details.Safe()
	delay := time.Second
	failed := false
	for ctx.Err() == nil {
		err := write(ctx)
		if err == nil {
			if failed {
				slog.Info("worker state recovered", "code", "WORKER_STATE_RECOVERED", "context", details)
			}
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		if !failed {
			slog.Error("worker state could not be saved; retrying journal", "code", "WORKER_STATE_WRITE_FAILED", "context", details)
			failed = true
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
		delay = min(delay*2, 30*time.Second)
	}
	return false
}

func (service *Service) reportProgress(ctx context.Context, job Job, operation, relative string, completed, total int) error {
	var totalValue any
	if total > 0 {
		totalValue = total
	}
	_, err := service.Database.Writer.ExecContext(ctx, `INSERT INTO job_progress(job_id,operation,relative_path,completed_units,total_units,updated_at) SELECT id,?,?,?,?,? FROM jobs WHERE id=? AND status='running' AND fencing_token=? ON CONFLICT(job_id) DO UPDATE SET operation=excluded.operation,relative_path=CASE WHEN excluded.relative_path='' THEN job_progress.relative_path ELSE excluded.relative_path END,completed_units=excluded.completed_units,total_units=excluded.total_units,updated_at=excluded.updated_at`, operation, relative, completed, totalValue, now(), job.ID, job.Fence)
	return err
}
func (service *Service) monitorCancellation(ctx context.Context, job Job, cancel context.CancelFunc, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		var requested int
		err := service.Database.Reader.QueryRowContext(ctx, "SELECT coalesce((SELECT cancel_requested FROM job_controls WHERE job_id=?),0)", job.ID).Scan(&requested)
		if err == nil && requested == 1 {
			cancel()
			return
		}
	}
}
func (service *Service) CancelJob(ctx context.Context, principal domain.Principal, jobID string, metadata domain.RequestMetadata) error {
	job, err := service.Job(ctx, principal, jobID)
	if err != nil {
		return err
	}
	if job.Kind != "scan" && job.Kind != "extract" {
		return invalid("Esta operación no admite cancelación. Espera a que termine.")
	}
	return service.write(ctx, principal, job.LibraryID, "indexing.run", func(tx *sql.Tx, current domain.Principal) error {
		var status, target string
		if err := tx.QueryRowContext(ctx, "SELECT status,target_version FROM jobs WHERE id=?", jobID).Scan(&status, &target); err != nil {
			return err
		}
		if status != "queued" && status != "running" && status != "retry_wait" && status != "paused" {
			return invalid("La tarea ya terminó.")
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO job_controls VALUES(?,1) ON CONFLICT(job_id) DO UPDATE SET cancel_requested=1", jobID); err != nil {
			return err
		}
		if job.Kind == "scan" {
			if _, err := tx.ExecContext(ctx, "INSERT INTO root_scan_controls VALUES(?,1) ON CONFLICT(root_id) DO UPDATE SET paused=1", target); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE jobs SET status='cancelled',last_error_code='USER_CANCELLED' WHERE job_type='scan' AND target_version=? AND status IN ('queued','retry_wait','paused')", target); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE jobs SET status='cancelled',last_error_code='USER_CANCELLED' WHERE id=? AND status<>'running'", jobID); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "indexing.cancel_requested", job.LibraryID, "", map[string]any{"job_id": jobID})
	})
}
func (service *Service) RetryScanErrors(ctx context.Context, principal domain.Principal, rootID string, metadata domain.RequestMetadata) error {
	root, err := service.root(ctx, rootID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, root.LibraryID, "indexing.retry", func(tx *sql.Tx, current domain.Principal) error {
		var busy int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM jobs WHERE job_type='scan' AND target_version=? AND status IN ('queued','running','retry_wait','paused')", rootID).Scan(&busy); err != nil {
			return err
		}
		if busy > 0 {
			return invalid("Espera a que termine el recorrido activo o cancélalo primero.")
		}
		rows, err := tx.QueryContext(ctx, "SELECT relative_path FROM scan_errors WHERE root_id=? ORDER BY relative_path LIMIT 1001", rootID)
		if err != nil {
			return err
		}
		paths := []string{}
		for rows.Next() {
			var path string
			if err = rows.Scan(&path); err != nil {
				rows.Close()
				return err
			}
			paths = append(paths, path)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return invalid("No hay rutas pendientes de reintento.")
		}
		if len(paths) > 1000 {
			paths = paths[:1000]
		}
		if _, err = tx.ExecContext(ctx, "UPDATE root_scans SET status='superseded' WHERE root_id=? AND status='running'", rootID); err != nil {
			return err
		}
		id := domain.NewID()
		if _, err = tx.ExecContext(ctx, "INSERT INTO jobs(id,library_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,'scan',?, ?,?,'queued',?,?)", id, root.LibraryID, rootID, id, encode(map[string]any{"paths": paths, "manual": true}), now(), now()); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "indexing.errors_retry_requested", root.LibraryID, "", map[string]any{"root_id": rootID, "paths_count": len(paths)})
	})
}
