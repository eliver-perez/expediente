package libraries

import (
	"context"
	"time"

	"gestor-documental/internal/domain"
)

// A missing snapshot is explicit: zero counts are only returned after a
// successful, complete aggregate. UpdatedAt is the database snapshot time.
type ProcessingStatsReport struct {
	Available bool             `json:"stats_available"`
	Stats     *ProcessingStats `json:"stats,omitempty"`
	UpdatedAt string           `json:"stats_updated_at,omitempty"`
	Error     string           `json:"stats_error,omitempty"`
}

func (service *Service) ProcessingStatistics(ctx context.Context, principal domain.Principal, libraryID string) (ProcessingStatsReport, error) {
	if err := service.Read(ctx, principal, libraryID, "indexing.run"); err != nil {
		return ProcessingStatsReport{}, err
	}
	return service.calculateProcessingStats(ctx, principal, libraryID)
}

// PostgreSQL counts have their own pool and a bounded budget, independent of
// progress polling. The former 750 ms SQLite budget cancelled useful work on
// large libraries. CASE avoids historical job/page lookups when not needed;
// MATERIALIZED prevents PostgreSQL repeating the latest-job lookup per count.
const processingStatsSQL = `WITH file_states AS MATERIALIZED (
 SELECT f.storage_source AS source,f.extraction_freshness AS freshness,coalesce(v.index_block_reason,'') AS index_reason,
 CASE WHEN f.extraction_freshness<>'current' THEN coalesce((SELECT status FROM jobs WHERE physical_file_id=f.id AND target_version=f.current_content_version_id AND job_type='extract' ORDER BY created_at DESC,id DESC LIMIT 1),'') ELSE '' END AS status,
 CASE WHEN f.extraction_freshness='current' THEN
   CASE WHEN jsonb_typeof((e.summary_json)::jsonb #> '{ocr_pages}') IS NOT NULL
   THEN (((e.summary_json)::jsonb #>> '{ocr_pages}')::bigint)>0
   ELSE EXISTS(SELECT 1 FROM extraction_pages WHERE extraction_id=f.indexed_extraction_id AND extraction_method IN ('ocr','empty')) END
 ELSE false END AS ocr
 FROM physical_files f JOIN documents d ON d.physical_file_id=f.id LEFT JOIN content_versions v ON v.id=f.current_content_version_id LEFT JOIN extraction_runs e ON e.id=f.indexed_extraction_id
 WHERE f.library_id=? AND d.deleted_at IS NULL AND ` + visibleDocumentSQL + `)
 SELECT count(*),count(*) FILTER (WHERE source='linked'),count(*) FILTER (WHERE source='managed'),
 count(*) FILTER (WHERE freshness='current'),count(*) FILTER (WHERE freshness<>'current' AND status='failed'),
 count(*) FILTER (WHERE freshness<>'current' AND status<>'failed' AND index_reason=''),
 count(*) FILTER (WHERE freshness='current' AND NOT ocr),count(*) FILTER (WHERE freshness='current' AND ocr),
 count(*) FILTER (WHERE freshness<>'current' AND index_reason<>''),statement_timestamp() FROM file_states`

func (service *Service) calculateProcessingStats(ctx context.Context, principal domain.Principal, libraryID string) (ProcessingStatsReport, error) {
	statsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var stats ProcessingStats
	var at time.Time
	err := service.Database.AnalyticsReader.QueryRowContext(statsCtx, processingStatsSQL, libraryID, principal.User.ID, principal.User.ID).Scan(
		&stats.Total, &stats.Linked, &stats.Managed, &stats.Processed, &stats.Errors, &stats.Pending, &stats.Native, &stats.OCR, &stats.StoredOnly, &at)
	if err != nil {
		if statsCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
			return ProcessingStatsReport{Error: "El cálculo de estadísticas superó el tiempo de espera. Se reintentará automáticamente."}, nil
		}
		return ProcessingStatsReport{}, err
	}
	return ProcessingStatsReport{Available: true, Stats: &stats, UpdatedAt: domain.Timestamp(at)}, nil
}
