package libraries

import (
	"context"
	"database/sql"
	"gestor-documental/internal/documentformat"
)

func admissionSkip(err error) bool {
	if err == nil {
		return false
	}
	switch failureCode(err) {
	case "FILE_TYPE_BLOCKED", "FILE_TYPE_UNKNOWN", "FILE_TYPE_MISMATCH", "INVALID_OFFICE_DOCUMENT", "DOCUMENT_COMPLEXITY_LIMIT", "FILE_FORMAT_DISABLED", "FILE_NOT_INDEXABLE":
		return true
	}
	return false
}

// A policy skip must not masquerade as deletion of a previously registered
// file. Keep its historical index and record that the path was observed.
func (service *Service) skipFile(ctx context.Context, root Root, scanID, relative string, detection documentformat.Detection, code string) error {
	return service.Database.Write(ctx, func(tx *sql.Tx) error {
		if err := service.checkRootRevision(ctx, tx, root); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO document_file_skips VALUES(?,?,?,?,?) ON CONFLICT(root_id,relative_path) DO UPDATE SET error_code=excluded.error_code,detection_json=excluded.detection_json,observed_at=excluded.observed_at`, root.ID, relative, code, encode(detection), now()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM scan_errors WHERE root_id=? AND relative_path=?", root.ID, relative); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO scan_file_observations SELECT ?,id,? FROM physical_file_locations WHERE root_id=? AND relative_path=? AND retired_at IS NULL ON CONFLICT DO NOTHING", scanID, now(), root.ID, relative)
		return err
	})
}
