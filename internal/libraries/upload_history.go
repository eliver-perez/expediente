package libraries

import (
	"context"
	"gestor-documental/internal/domain"
)

// One bounded page, with status derived from the existing transfer, job and run.
func (service *Service) BatchItems(ctx context.Context, principal domain.Principal, batchID, cursor string) (Page[UploadItem], error) {
	result := Page[UploadItem]{Items: []UploadItem{}}
	if _, err := service.batchHeader(ctx, principal, batchID); err != nil {
		return result, err
	}
	scope := principal.User.ID + ":upload-items:" + batchID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, `SELECT u.id,coalesce(u.document_id,''),u.original_filename,u.status,u.error_code,u.size_bytes,u.created_at,
 coalesce(v.document_format,''),coalesce(v.index_block_reason,''),
 d.id IS NOT NULL AND d.deleted_at IS NULL AND `+visibleDocumentSQL+`,
 coalesce(j.status,''),coalesce(j.last_error_code,''),coalesce(p.operation,''),coalesce(e.result_code,''),
 ((SELECT paused FROM processing_policy WHERE singleton=1) OR NOT coalesce((1=1 `+advancedJobEligibility+`),1))
 FROM upload_items u LEFT JOIN documents d ON d.id=u.document_id LEFT JOIN physical_files f ON f.id=d.physical_file_id
 LEFT JOIN content_versions v ON v.id=f.current_content_version_id
 LEFT JOIN jobs j ON j.id=(SELECT id FROM jobs WHERE physical_file_id=f.id AND job_type='extract' AND target_version=f.current_content_version_id ORDER BY created_at DESC,id DESC LIMIT 1)
 LEFT JOIN job_progress p ON p.job_id=j.id LEFT JOIN extraction_runs e ON e.id=f.indexed_extraction_id
 WHERE u.batch_id=? AND (?='' OR u.created_at<? OR (u.created_at=? AND u.id<?)) ORDER BY u.created_at DESC,u.id DESC LIMIT 26`, principal.User.ID, principal.User.ID, batchID, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item UploadItem
		var job Job
		var outcome string
		var paused bool
		if err = rows.Scan(&item.ID, &item.DocumentID, &item.Filename, &item.Status, &item.Error, &item.Size, &item.Created, &item.Format, &item.IndexReason, &item.CanView, &job.Status, &job.Error, &job.Operation, &outcome, &paused); err != nil {
			return result, err
		}
		if job.Status != "" {
			item.ProcessingState = jobProcessingState(job, outcome, paused)
		}
		result.Items = append(result.Items, item)
	}
	if len(result.Items) > 25 {
		result.Items = result.Items[:25]
		last := result.Items[24]
		result.Next = nextListCursor(scope, last.Created, last.ID)
	}
	return result, rows.Err()
}
