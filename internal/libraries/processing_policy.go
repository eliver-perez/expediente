package libraries

import (
	"context"
	"database/sql"
	"time"

	"gestor-documental/internal/domain"
)

type ProcessingPolicy struct {
	Paused            bool  `json:"paused"`
	MaximumAttempts   int   `json:"maximum_attempts"`
	RetryDelaySeconds int   `json:"retry_delay_seconds"`
	TimeoutSeconds    int   `json:"timeout_seconds"`
	Revision          int64 `json:"revision"`
}

func (service *Service) ProcessingPolicy(ctx context.Context) (ProcessingPolicy, error) {
	var policy ProcessingPolicy
	err := service.Database.Reader.QueryRowContext(ctx, `SELECT paused,maximum_attempts,retry_delay_seconds,timeout_seconds,revision FROM processing_policy WHERE singleton=1`).Scan(&policy.Paused, &policy.MaximumAttempts, &policy.RetryDelaySeconds, &policy.TimeoutSeconds, &policy.Revision)
	return policy, err
}

// Pause and claiming use the same writer transaction boundary. A job already
// claimed can finish (including its OCR stage); no subsequent claim can pass.
func (service *Service) ConfigureProcessingPolicy(ctx context.Context, principal domain.Principal, policy ProcessingPolicy, metadata domain.RequestMetadata) error {
	if policy.MaximumAttempts < 1 || policy.MaximumAttempts > 5 || policy.RetryDelaySeconds < 1 || policy.RetryDelaySeconds > 3600 || policy.TimeoutSeconds < 10 || policy.TimeoutSeconds > 3600 {
		return invalid("Usa de 1 a 5 intentos, una espera de 1 a 3600 segundos y un tiempo máximo de 10 a 3600 segundos.")
	}
	return service.Identity.AuthorizedWrite(ctx, principal, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
		result, err := tx.ExecContext(ctx, `UPDATE processing_policy SET paused=?,maximum_attempts=?,retry_delay_seconds=?,timeout_seconds=?,revision=revision+1,updated_at=? WHERE singleton=1 AND revision=?`, policy.Paused, policy.MaximumAttempts, policy.RetryDelaySeconds, policy.TimeoutSeconds, now(), policy.Revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("REVISION_CONFLICT", "La configuración cambió. Actualiza antes de guardar.", 409)
		}
		return record(ctx, tx, current, metadata, "system.processing_policy_configured", "", "", map[string]any{"paused": policy.Paused, "maximum_attempts": policy.MaximumAttempts, "retry_delay_seconds": policy.RetryDelaySeconds, "timeout_seconds": policy.TimeoutSeconds})
	})
}

func jobTimeout(job Job) time.Duration {
	seconds := job.TimeoutSeconds
	if seconds <= 0 {
		seconds = 3600
	}
	return time.Duration(seconds) * time.Second
}

// Only the local panic is contained. Never return a panic value/stack to the UI,
// as it may include document content or filesystem paths.
func isolatedOperation(operation func() error) (err error) {
	defer func() {
		if recover() != nil {
			err = scanFailure("PROCESSING_INTERNAL_ERROR")
		}
	}()
	return operation()
}

func processingErrorClass(code string) string {
	switch code {
	case "":
		return ""
	case "PROCESS_TIMEOUT":
		return "timeout"
	case "INVALID_OFFICE_DOCUMENT", "INVALID_PDF":
		return "invalid_document"
	case "FILE_TYPE_BLOCKED", "FILE_TYPE_UNKNOWN", "FILE_TYPE_MISMATCH", "EXTRACTOR_UNAVAILABLE", "FILE_FORMAT_DISABLED":
		return "unsupported"
	case "FILE_SIZE_LIMIT", "FILE_TOO_LARGE", "PAGE_LIMIT", "TEXT_LIMIT", "DOCUMENT_COMPLEXITY_LIMIT":
		return "limit"
	case "EXTRACTION_FAILED", "OCR_FAILED":
		return "extractor"
	case "USER_CANCELLED", "INTERRUPTED", "PROCESS_RESTARTED":
		return "interrupted"
	default:
		return "system"
	}
}
