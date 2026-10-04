package libraries

import (
	"context"
	"database/sql"
	"encoding/json"

	"gestor-documental/internal/extraction"
)

type ProcessingRun struct {
	ID             string                `json:"id"`
	VersionID      string                `json:"content_version_id"`
	Extractor      extraction.Descriptor `json:"extractor"`
	LegacyRevision string                `json:"extractor_revision"`
	Started        string                `json:"started_at"`
	Completed      string                `json:"completed_at"`
	Status         string                `json:"status"`
	Error          string                `json:"error_code"`
	Units          int                   `json:"unit_count"`
	Summary        map[string]any        `json:"summary"`
	Warnings       []string              `json:"warnings"`
}

// Called only after document-level authorization, including private uploads.
// Existing runs retain their unknown start time/version instead of inventing it.
func (service *Service) latestProcessing(ctx context.Context, fileID string) (*ProcessingRun, error) {
	var result ProcessingRun
	var summary, warnings string
	err := service.Database.Reader.QueryRowContext(ctx, `SELECT id,content_version_id,extractor_id,extractor_version,document_format,extractor_revision,coalesce(started_at,''),coalesce(completed_at,''),result_code,coalesce(error_code,''),unit_count,summary_json,warnings_json FROM extraction_runs WHERE physical_file_id=? ORDER BY rowid DESC LIMIT 1`, fileID).Scan(&result.ID, &result.VersionID, &result.Extractor.ID, &result.Extractor.Version, &result.Extractor.Format, &result.LegacyRevision, &result.Started, &result.Completed, &result.Status, &result.Error, &result.Units, &summary, &warnings)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(summary), &result.Summary); err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(warnings), &result.Warnings); err != nil {
		return nil, err
	}
	return &result, nil
}
