package libraries

import (
	"context"
	"database/sql"

	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
)

// Schedule only a first extraction previously withheld by format policy or by
// missing capability. Existing indexes, terminal failures and user cancellations
// are preserved; this is not a reindex command or an extractor-version upgrade.
func (service *Service) queueNewlyIndexable(ctx context.Context, tx *sql.Tx, libraryID string) error {
	rows, err := tx.QueryContext(ctx, `SELECT f.id,f.library_id,v.id,v.document_format,v.detected_mime,v.original_extension,v.extension_mismatch,v.size_bytes FROM physical_files f JOIN content_versions v ON v.id=f.current_content_version_id JOIN documents d ON d.physical_file_id=f.id WHERE (?='' OR f.library_id=?) AND f.indexed_extraction_id IS NULL AND f.availability IN ('available','staged') AND f.integrity_status='verified' AND v.index_block_reason<>'' AND v.index_block_reason<>'content_rejected' AND d.deleted_at IS NULL AND d.approval_status NOT IN ('cancelled','materializing') AND NOT EXISTS(SELECT 1 FROM jobs j WHERE j.physical_file_id=f.id AND j.target_version=v.id AND j.job_type='extract' AND (j.status<>'cancelled' OR EXISTS(SELECT 1 FROM job_controls c WHERE c.job_id=j.id AND c.cancel_requested=1)))`, libraryID, libraryID)
	if err != nil {
		return err
	}
	type candidate struct {
		file, library, version string
		detection              documentformat.Detection
		size                   int64
	}
	items := []candidate{}
	for rows.Next() {
		var item candidate
		if err = rows.Scan(&item.file, &item.library, &item.version, &item.detection.Format, &item.detection.MIME, &item.detection.Extension, &item.detection.Mismatch, &item.size); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	policies := map[string]documentformat.Policy{}
	for _, item := range items {
		policy, ok := policies[item.library]
		if !ok {
			config, err := service.fileConfiguration(ctx, tx, item.library)
			if err != nil {
				return err
			}
			policy = config.Effective
			policies[item.library] = policy
		}
		reason := policy.IndexReason(item.detection, item.size)
		if err = saveDetection(ctx, tx, item.version, item.detection, reason); err != nil {
			return err
		}
		if reason != "" {
			continue
		}
		jobID := domain.NewID()
		if _, err = tx.ExecContext(ctx, `INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','queued',?,?)`, jobID, item.library, item.file, item.version, jobID, now(), now()); err != nil {
			return err
		}
	}
	return nil
}
