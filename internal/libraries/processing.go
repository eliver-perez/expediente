package libraries

import (
	"context"
	"gestor-documental/internal/extraction"
	"time"

	"gestor-documental/internal/domain"
)

type ProcessingStats struct {
	Total     int `json:"total"`
	Linked    int `json:"linked"`
	Managed   int `json:"managed"`
	Pending   int `json:"pending"`
	Processed int `json:"processed"`
	Errors    int `json:"errors"`
	Native    int `json:"native"`
	OCR       int `json:"ocr"`
}
type ScanProgress struct {
	ID                 string `json:"id"`
	RootID             string `json:"root_id"`
	Path               string `json:"server_path"`
	Status             string `json:"status"`
	Started            string `json:"started_at"`
	Updated            string `json:"updated_at"`
	Completed          string `json:"completed_at"`
	Current            string `json:"current_path"`
	Directories        int    `json:"directories"`
	PendingDirectories int    `json:"pending_directories"`
	Files              int    `json:"files"`
	Unchanged          int    `json:"unchanged"`
	Hashed             int    `json:"hashed"`
	Excluded           int    `json:"excluded"`
	Errors             int    `json:"errors"`
	Paused             bool   `json:"paused"`
}
type ScanError struct {
	RootID string `json:"root_id"`
	Path   string `json:"relative_path"`
	Code   string `json:"error_code"`
	At     string `json:"occurred_at"`
}
type ProcessingPerformance struct {
	NativeActive       int     `json:"native_active"`
	OCRActive          int     `json:"ocr_active"`
	WaitingOCR         int     `json:"waiting_ocr"`
	Queued             int     `json:"queued"`
	DocumentsPerMinute float64 `json:"documents_per_minute"`
	AverageSeconds     float64 `json:"average_seconds"`
	WindowMinutes      int     `json:"window_minutes"`
}
type ProcessingReport struct {
	Workers     extraction.Concurrency `json:"workers"`
	Performance ProcessingPerformance  `json:"performance"`
	Stats       ProcessingStats        `json:"stats"`
	Scans       []ScanProgress         `json:"scans"`
	Active      []Job                  `json:"active_jobs"`
	Errors      []ScanError            `json:"scan_errors"`
}

func (service *Service) Processing(ctx context.Context, principal domain.Principal, libraryID string) (ProcessingReport, error) {
	report := ProcessingReport{Scans: []ScanProgress{}, Active: []Job{}, Errors: []ScanError{}}
	if err := service.Read(ctx, principal, libraryID, "indexing.run"); err != nil {
		return report, err
	}
	// One row per physical file/document, never per location or historical job.
	err := service.Database.Reader.QueryRowContext(ctx, `WITH file_states AS (
 SELECT f.storage_source AS source,f.extraction_freshness AS freshness,
 coalesce((SELECT status FROM jobs WHERE physical_file_id=f.id AND target_version=f.current_content_version_id AND job_type='extract' ORDER BY created_at DESC,id DESC LIMIT 1),'') AS status,
 EXISTS(SELECT 1 FROM extraction_pages WHERE extraction_id=f.indexed_extraction_id AND extraction_method IN ('ocr','empty')) AS ocr
 FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE f.library_id=? AND d.deleted_at IS NULL AND `+visibleDocumentSQL+`)
 SELECT count(*),coalesce(sum(source='linked'),0),coalesce(sum(source='managed'),0),coalesce(sum(freshness='current'),0),coalesce(sum(freshness<>'current' AND status='failed'),0),coalesce(sum(freshness<>'current' AND status<>'failed'),0),coalesce(sum(freshness='current' AND ocr=0),0),coalesce(sum(freshness='current' AND ocr=1),0) FROM file_states`, libraryID, principal.User.ID, principal.User.ID).Scan(&report.Stats.Total, &report.Stats.Linked, &report.Stats.Managed, &report.Stats.Processed, &report.Stats.Errors, &report.Stats.Pending, &report.Stats.Native, &report.Stats.OCR)
	if err != nil {
		return report, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, `SELECT s.id,s.root_id,r.canonical_path,s.status,s.started_at,coalesce(p.updated_at,s.started_at),coalesce(s.completed_at,''),coalesce(p.current_path,''),s.directories_seen,json_array_length(s.checkpoint_json),s.files_seen,coalesce(p.unchanged_files,0),coalesce(p.hashed_files,0),coalesce(p.excluded_directories,0),(SELECT count(*) FROM scan_errors e WHERE e.root_id=r.id),coalesce(c.paused,0)
 FROM storage_roots r JOIN root_scans s ON s.id=(SELECT id FROM root_scans WHERE root_id=r.id ORDER BY started_at DESC,id DESC LIMIT 1) LEFT JOIN scan_progress p ON p.scan_id=s.id LEFT JOIN root_scan_controls c ON c.root_id=r.id WHERE r.library_id=? AND r.status<>'superseded' ORDER BY r.created_at`, libraryID)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var scan ScanProgress
		if err = rows.Scan(&scan.ID, &scan.RootID, &scan.Path, &scan.Status, &scan.Started, &scan.Updated, &scan.Completed, &scan.Current, &scan.Directories, &scan.PendingDirectories, &scan.Files, &scan.Unchanged, &scan.Hashed, &scan.Excluded, &scan.Errors, &scan.Paused); err != nil {
			rows.Close()
			return report, err
		}
		if service.require(ctx, service.Database.Reader, principal, libraryID, "storage.view_paths") != nil {
			scan.Path = ""
		}
		report.Scans = append(report.Scans, scan)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	rows, err = service.Database.Reader.QueryContext(ctx, `SELECT j.id FROM jobs j WHERE j.library_id=? AND j.status='running' AND (j.physical_file_id IS NULL OR EXISTS(SELECT 1 FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE f.id=j.physical_file_id AND `+visibleDocumentSQL+`)) ORDER BY j.created_at LIMIT 32`, libraryID, principal.User.ID, principal.User.ID)
	if err != nil {
		return report, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return report, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	for _, id := range ids {
		job, err := service.Job(ctx, principal, id)
		if err != nil {
			return report, err
		}
		report.Active = append(report.Active, job)
	}
	configuration, err := service.ProcessingConfiguration(ctx)
	if err != nil {
		return report, err
	}
	report.Workers = configuration.Effective
	for _, job := range report.Active {
		if job.Kind != "extract" && job.Kind != "materialize" {
			continue
		}
		switch job.Operation {
		case "waiting_ocr":
			report.Performance.WaitingOCR++
		case "render", "ocr":
			report.Performance.OCRActive++
		default:
			report.Performance.NativeActive++
		}
	}
	if err = service.Database.Reader.QueryRowContext(ctx, `SELECT count(*) FROM jobs j JOIN documents d ON d.physical_file_id=j.physical_file_id JOIN physical_files f ON f.id=j.physical_file_id WHERE j.library_id=? AND j.job_type='extract' AND j.status IN ('queued','retry_wait') AND `+visibleDocumentSQL, libraryID, principal.User.ID, principal.User.ID).Scan(&report.Performance.Queued); err != nil {
		return report, err
	}
	report.Performance.WindowMinutes = 5
	if err = service.Database.Reader.QueryRowContext(ctx, `SELECT count(*)/5.0,coalesce(avg(max(0,(julianday(a.finished_at)-julianday(a.started_at))*86400)),0) FROM job_attempts a JOIN jobs j ON j.id=a.job_id AND j.fencing_token=a.fencing_token JOIN documents d ON d.physical_file_id=j.physical_file_id JOIN physical_files f ON f.id=j.physical_file_id WHERE a.finished_at>=? AND coalesce(a.error_code,'')='' AND j.status='succeeded' AND j.job_type='extract' AND j.library_id=? AND `+visibleDocumentSQL, domain.Timestamp(time.Now().Add(-5*time.Minute)), libraryID, principal.User.ID, principal.User.ID).Scan(&report.Performance.DocumentsPerMinute, &report.Performance.AverageSeconds); err != nil {
		return report, err
	}

	rows, err = service.Database.Reader.QueryContext(ctx, `SELECT e.root_id,e.relative_path,e.error_code,e.occurred_at FROM scan_errors e JOIN storage_roots r ON r.id=e.root_id WHERE r.library_id=? ORDER BY e.occurred_at DESC LIMIT 100`, libraryID)
	if err != nil {
		return report, err
	}
	defer rows.Close()
	for rows.Next() {
		var issue ScanError
		if err = rows.Scan(&issue.RootID, &issue.Path, &issue.Code, &issue.At); err != nil {
			return report, err
		}
		report.Errors = append(report.Errors, issue)
	}
	return report, rows.Err()
}
func (service *Service) enrichJob(ctx context.Context, job *Job) error {
	return service.Database.Reader.QueryRowContext(ctx, `SELECT b.name,coalesce(d.original_filename,''),coalesce(p.relative_path,l.relative_path,''),coalesce(r.canonical_path,''),coalesce(r.id,''),coalesce(p.operation,j.job_type),coalesce(p.completed_units,0),p.total_units,coalesce(p.updated_at,''),coalesce((SELECT started_at FROM job_attempts WHERE job_id=j.id ORDER BY attempt_number DESC LIMIT 1),''),coalesce((SELECT finished_at FROM job_attempts WHERE job_id=j.id ORDER BY attempt_number DESC LIMIT 1),''),coalesce(c.cancel_requested,0)
 FROM jobs j JOIN libraries b ON b.id=j.library_id LEFT JOIN documents d ON d.physical_file_id=j.physical_file_id LEFT JOIN physical_files f ON f.id=j.physical_file_id LEFT JOIN physical_file_locations l ON l.id=f.primary_location_id LEFT JOIN storage_roots r ON r.id=CASE WHEN j.job_type IN ('scan','verify_managed') THEN j.target_version ELSE l.root_id END LEFT JOIN job_progress p ON p.job_id=j.id LEFT JOIN job_controls c ON c.job_id=j.id WHERE j.id=?`, job.ID).Scan(&job.LibraryName, &job.Filename, &job.Relative, &job.RootPath, &job.RootID, &job.Operation, &job.CompletedUnits, &job.TotalUnits, &job.Updated, &job.Started, &job.Finished, &job.Cancelling)
}
