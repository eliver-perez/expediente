package libraries

import (
	"context"
	"database/sql"
	"gestor-documental/internal/domain"
	"time"
)

// Aggregates never include names, paths or text from private documents. This
// administrative overview does not grant document access or bypass library ACLs.
const dashboardFiles = `WITH base AS (
 SELECT d.id,d.created_at,d.approval_status,f.library_id,f.storage_source,f.extraction_freshness,
 coalesce(v.document_format,'unknown') AS format,coalesce(v.sha256,'') AS hash,coalesce(v.index_block_reason,'') AS reason,
 coalesce(j.status,'') AS job_status,coalesce(e.result_code,'') AS result,
 EXISTS(SELECT 1 FROM extraction_pages ep WHERE ep.extraction_id=f.indexed_extraction_id AND ep.extraction_method IN ('ocr','empty')) AS ocr,
 ((SELECT paused FROM processing_policy WHERE singleton=1) OR NOT coalesce((1=1 ` + advancedJobEligibility + `),1)) AS paused
 FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN libraries b ON b.id=d.library_id
 LEFT JOIN content_versions v ON v.id=f.current_content_version_id LEFT JOIN extraction_runs e ON e.id=f.indexed_extraction_id
 LEFT JOIN jobs j ON j.id=(SELECT id FROM jobs WHERE physical_file_id=f.id AND target_version=f.current_content_version_id AND job_type='extract' ORDER BY created_at DESC,id DESC LIMIT 1)
 WHERE d.deleted_at IS NULL AND b.disabled_at IS NULL
), files AS (SELECT *,CASE
 WHEN reason<>'' AND extraction_freshness<>'current' THEN 'stored_only'
 WHEN job_status='failed' THEN 'error'
 WHEN job_status='running' THEN 'processing'
 WHEN job_status='paused' OR (paused AND job_status IN ('queued','retry_wait')) THEN 'paused'
 WHEN job_status IN ('queued','retry_wait') THEN 'pending'
 WHEN job_status='cancelled' THEN 'cancelled'
 WHEN extraction_freshness='current' THEN CASE WHEN result='complete_with_warnings' THEN 'completed_with_warnings' ELSE 'completed' END
 ELSE 'pending' END AS state FROM base) `

type MetricGroup struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}
type DashboardDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
type DashboardLibrary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Mode    string `json:"mode"`
	Total   int    `json:"total"`
	Indexed int    `json:"indexed"`
	Errors  int    `json:"errors"`
}
type Dashboard struct {
	Generated       string            `json:"generated_at"`
	Timezone        string            `json:"timezone"`
	Today           int               `json:"today"`
	Week            int               `json:"week"`
	Month           int               `json:"month"`
	Total           int               `json:"total"`
	Indexed         int               `json:"indexed"`
	OCR             int               `json:"ocr"`
	Linked          int               `json:"linked"`
	Managed         int               `json:"managed"`
	DuplicateGroups int               `json:"duplicate_groups"`
	DuplicateFiles  int               `json:"duplicate_files"`
	ExtraCopies     int               `json:"extra_copies"`
	Queue           []MetricGroup     `json:"queue"`
	Formats         []MetricGroup     `json:"formats"`
	States          []MetricGroup     `json:"states"`
	Approval        []MetricGroup     `json:"approval"`
	CacheBytes      int64             `json:"cache_bytes"`
	CacheLimitMB    int               `json:"cache_limit_mb"`
	CacheEntries    int               `json:"cache_entries"`
	AverageSeconds  *float64          `json:"average_seconds"`
	TimingSamples   int               `json:"timing_samples"`
	Paused          bool              `json:"paused"`
	Recent          []DiagnosticEvent `json:"recent_errors"`
	Daily           []DashboardDay    `json:"daily"`
}

// Use calendar days in the server's zone, including DST transitions. Bucketing
// UTC substrings or subtracting 24-hour durations would misplace local-midnight
// documents. Only timestamps are read; document names/content never leave SQL.
func dashboardDays(ctx context.Context, tx *sql.Tx, instant time.Time) ([]DashboardDay, error) {
	day := time.Date(instant.Year(), instant.Month(), instant.Day(), 0, 0, 0, 0, instant.Location())
	first := day.AddDate(0, 0, -29)
	result := make([]DashboardDay, 30)
	positions := make(map[string]int, 30)
	for i := range result {
		result[i].Date = first.AddDate(0, 0, i).Format("2006-01-02")
		positions[result[i].Date] = i
	}
	rows, err := tx.QueryContext(ctx, `SELECT d.created_at FROM documents d JOIN libraries b ON b.id=d.library_id WHERE d.deleted_at IS NULL AND b.disabled_at IS NULL AND d.created_at>=? AND d.created_at<=?`, domain.Timestamp(first), domain.Timestamp(instant))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		at, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, err
		}
		if index, exists := positions[at.In(instant.Location()).Format("2006-01-02")]; exists {
			result[index].Count++
		}
	}
	return result, rows.Err()
}

func metricGroups(ctx context.Context, tx *sql.Tx, query string) ([]MetricGroup, error) {
	result := []MetricGroup{}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var g MetricGroup
		if err = rows.Scan(&g.Key, &g.Count); err != nil {
			return result, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}
func (s *Service) Dashboard(ctx context.Context, p domain.Principal) (Dashboard, error) {
	result := Dashboard{}
	if err := systemAccess(p); err != nil {
		return result, err
	}
	instant := s.Identity.Now()
	day := time.Date(instant.Year(), instant.Month(), instant.Day(), 0, 0, 0, 0, instant.Location())
	week := day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
	month := time.Date(instant.Year(), instant.Month(), 1, 0, 0, 0, 0, instant.Location())
	result.Generated = domain.Timestamp(instant)
	result.Timezone, _ = instant.Zone()
	tx, err := s.Database.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, dashboardFiles+`SELECT count(*),coalesce(sum(created_at>=? AND created_at<=?),0),coalesce(sum(created_at>=? AND created_at<=?),0),coalesce(sum(created_at>=? AND created_at<=?),0),coalesce(sum(storage_source='linked'),0),coalesce(sum(storage_source='managed'),0),coalesce(sum(extraction_freshness='current'),0),coalesce(sum(extraction_freshness='current' AND ocr),0) FROM files`, domain.Timestamp(day), result.Generated, domain.Timestamp(week), result.Generated, domain.Timestamp(month), result.Generated).Scan(&result.Total, &result.Today, &result.Week, &result.Month, &result.Linked, &result.Managed, &result.Indexed, &result.OCR)
	if err != nil {
		return result, err
	}
	if result.Formats, err = metricGroups(ctx, tx, dashboardFiles+"SELECT format,count(*) FROM files GROUP BY format ORDER BY count(*) DESC,format"); err != nil {
		return result, err
	}
	if result.States, err = metricGroups(ctx, tx, dashboardFiles+"SELECT state,count(*) FROM files GROUP BY state ORDER BY count(*) DESC,state"); err != nil {
		return result, err
	}
	if result.Approval, err = metricGroups(ctx, tx, dashboardFiles+"SELECT approval_status,count(*) FROM files GROUP BY approval_status ORDER BY count(*) DESC,approval_status"); err != nil {
		return result, err
	}
	if err = tx.QueryRowContext(ctx, dashboardFiles+`SELECT count(*),coalesce(sum(copies),0),coalesce(sum(copies-1),0) FROM (SELECT count(*) AS copies FROM files WHERE hash<>'' GROUP BY hash HAVING count(*)>1)`).Scan(&result.DuplicateGroups, &result.DuplicateFiles, &result.ExtraCopies); err != nil {
		return result, err
	}
	if result.Queue, err = metricGroups(ctx, tx, `SELECT status,count(*) FROM jobs j JOIN libraries b ON b.id=j.library_id WHERE b.disabled_at IS NULL AND status IN ('queued','retry_wait','paused','running') GROUP BY status UNION ALL SELECT 'preview_'||status,count(*) FROM preview_cache WHERE status IN ('queued','generating') GROUP BY status`); err != nil {
		return result, err
	}
	if err = tx.QueryRowContext(ctx, `SELECT (SELECT coalesce(sum(size_bytes),0) FROM preview_cache),(SELECT count(*) FROM preview_cache WHERE status='ready'),maximum_cache_mb FROM preview_settings WHERE singleton=1`).Scan(&result.CacheBytes, &result.CacheEntries, &result.CacheLimitMB); err != nil {
		return result, err
	}
	if err = tx.QueryRowContext(ctx, `SELECT paused FROM processing_policy WHERE singleton=1`).Scan(&result.Paused); err != nil {
		return result, err
	}
	// Final successful attempts only, with valid recorded times. Excludes interrupted
	// or failed attempts and time spent waiting in the queue before acquisition.
	if err = tx.QueryRowContext(ctx, `SELECT count(*),avg((julianday(a.finished_at)-julianday(a.started_at))*86400) FROM job_attempts a JOIN jobs j ON j.id=a.job_id AND j.fencing_token=a.fencing_token JOIN libraries b ON b.id=j.library_id WHERE b.disabled_at IS NULL AND j.job_type='extract' AND j.status='succeeded' AND coalesce(a.error_code,'')='' AND a.finished_at>=? AND a.finished_at<=? AND julianday(a.finished_at)>=julianday(a.started_at)`, domain.Timestamp(instant.Add(-24*time.Hour)), result.Generated).Scan(&result.TimingSamples, &result.AverageSeconds); err != nil {
		return result, err
	}
	if result.Recent, err = recentDiagnostics(ctx, tx); err != nil {
		return result, err
	}
	if result.Daily, err = dashboardDays(ctx, tx, instant); err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s *Service) DashboardLibraries(ctx context.Context, p domain.Principal, cursor string) (Page[DashboardLibrary], error) {
	result := Page[DashboardLibrary]{Items: []DashboardLibrary{}}
	if err := systemAccess(p); err != nil {
		return result, err
	}
	scope := p.User.ID + ":dashboard-libraries"
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := s.Database.Reader.QueryContext(ctx, dashboardFiles+`SELECT b.id,b.name,b.mode,count(f.id),coalesce(sum(f.extraction_freshness='current'),0),coalesce(sum(f.state='error'),0) FROM libraries b LEFT JOIN files f ON f.library_id=b.id WHERE b.disabled_at IS NULL AND (?='' OR b.created_at<? OR (b.created_at=? AND b.id<?)) GROUP BY b.id ORDER BY b.created_at DESC,b.id DESC LIMIT 26`, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var library DashboardLibrary
		if err = rows.Scan(&library.ID, &library.Name, &library.Mode, &library.Total, &library.Indexed, &library.Errors); err != nil {
			return result, err
		}
		result.Items = append(result.Items, library)
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	rows.Close()
	if len(result.Items) > 25 {
		result.Items = result.Items[:25]
		last := result.Items[24]
		var created string
		if err = s.Database.Reader.QueryRowContext(ctx, "SELECT created_at FROM libraries WHERE id=?", last.ID).Scan(&created); err != nil {
			return result, err
		}
		result.Next = nextListCursor(scope, created, last.ID)
	}
	return result, nil
}
