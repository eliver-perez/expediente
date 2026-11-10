package libraries

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

// Only parameters without an existing administrative owner live here. Cache and
// worker budgets remain installation-wide in their original settings tables.
type AdvancedPolicy struct {
	PreviewEnabled        bool     `json:"preview_enabled"`
	PreviewSourceMB       int      `json:"preview_source_mb"`
	AutomaticProcessing   bool     `json:"automatic_processing"`
	ReprocessChanges      bool     `json:"reprocess_changes"`
	ManualPriority        bool     `json:"manual_priority"`
	OCREnabled            bool     `json:"ocr_enabled"`
	OCRAutomatic          bool     `json:"ocr_automatic"`
	OCRLanguages          string   `json:"ocr_languages"`
	OCRMinimumCharacters  int      `json:"ocr_minimum_characters"`
	MaximumPages          int      `json:"maximum_pages"`
	PageTimeoutSeconds    int      `json:"page_timeout_seconds"`
	WatcherEnabled        bool     `json:"watcher_enabled"`
	IgnoreOfficeTemporary bool     `json:"ignore_office_temporary"`
	IgnoreHidden          bool     `json:"ignore_hidden"`
	IgnoreTemporary       bool     `json:"ignore_temporary"`
	IgnoredExtensions     []string `json:"ignored_extensions"`
}
type AdvancedConfiguration struct {
	Global          AdvancedPolicy             `json:"global"`
	Effective       AdvancedPolicy             `json:"effective"`
	Overrides       map[string]json.RawMessage `json:"overrides"`
	Revision        int64                      `json:"revision"`
	GlobalRevision  int64                      `json:"global_revision"`
	PreviewRevision int64                      `json:"preview_revision"`
}

func mergeAdvanced(base AdvancedPolicy, raw []byte) (AdvancedPolicy, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return base, err
	}
	for key, value := range fields {
		if string(value) == "null" {
			delete(fields, key)
		}
	}
	raw, _ = json.Marshal(fields)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&base)
	return base, err
}
func (policy AdvancedPolicy) validate() error {
	if policy.PreviewSourceMB < 1 || policy.PreviewSourceMB > 512 {
		return invalid("El límite por vista previa debe ser de 1 a 512 MiB.")
	}
	if policy.OCRLanguages != "spa" && policy.OCRLanguages != "eng" && policy.OCRLanguages != "spa+eng" {
		return invalid("Idioma OCR inválido.")
	}
	if policy.OCRMinimumCharacters < 1 || policy.OCRMinimumCharacters > 10000 || policy.MaximumPages < 1 || policy.MaximumPages > 10000 || policy.PageTimeoutSeconds < 5 || policy.PageTimeoutSeconds > 300 {
		return invalid("Revisa los límites del OCR: 1–10000 caracteres y páginas; 5–300 segundos por operación PDF/OCR.")
	}
	if len(policy.IgnoredExtensions) > 32 {
		return invalid("Se permiten hasta 32 extensiones ignoradas.")
	}
	for _, extension := range policy.IgnoredExtensions {
		if len(extension) < 2 || len(extension) > 16 || extension[0] != '.' || strings.ContainsAny(extension[1:], "./\\ *?\t\r\n") {
			return invalid("Usa extensiones como .tmp, sin rutas ni comodines.")
		}
	}
	return nil
}
func (service *Service) advancedConfiguration(ctx context.Context, q storage.Querier, library string) (AdvancedConfiguration, error) {
	defaults := service.Identity.Config.Indexing
	result := AdvancedConfiguration{Global: AdvancedPolicy{PreviewEnabled: true, PreviewSourceMB: 64, AutomaticProcessing: true, ReprocessChanges: true, ManualPriority: true, OCREnabled: true, OCRAutomatic: true, OCRLanguages: "spa", OCRMinimumCharacters: 32, MaximumPages: defaults.MaximumPages, PageTimeoutSeconds: defaults.PageTimeoutSeconds, WatcherEnabled: true, IgnoreOfficeTemporary: true, IgnoreTemporary: true, IgnoredExtensions: []string{}}, Overrides: map[string]json.RawMessage{}}
	if result.Global.MaximumPages == 0 {
		result.Global.MaximumPages = 1000
	}
	if result.Global.PageTimeoutSeconds == 0 {
		result.Global.PageTimeoutSeconds = 120
	}
	var raw string
	err := q.QueryRowContext(ctx, "SELECT configuration_json,revision FROM advanced_settings WHERE scope='global'").Scan(&raw, &result.GlobalRevision)
	if err == nil {
		result.Global, err = mergeAdvanced(result.Global, []byte(raw))
	}
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	preview, previewError := previewSettings(ctx, q)
	if previewError != nil {
		return result, previewError
	}
	result.PreviewRevision = preview.Revision
	result.Global.PreviewEnabled = preview.Enabled
	result.Global.PreviewSourceMB = preview.MaximumSourceMB
	result.Effective = result.Global
	result.Revision = result.GlobalRevision
	if library != "" {
		result.Revision = 0
		err = q.QueryRowContext(ctx, "SELECT configuration_json,revision FROM advanced_settings WHERE scope=?", "library:"+library).Scan(&raw, &result.Revision)
		if err == nil {
			if err = json.Unmarshal([]byte(raw), &result.Overrides); err != nil {
				return result, err
			}
			result.Effective, err = mergeAdvanced(result.Global, []byte(raw))
		}
		if err != nil && err != sql.ErrNoRows {
			return result, err
		}
	}
	return result, nil
}
func (service *Service) AdvancedConfiguration(ctx context.Context, p domain.Principal, library string) (AdvancedConfiguration, error) {
	if library == "" {
		if !p.Can("system.configure") {
			return AdvancedConfiguration{}, domain.Failure("FORBIDDEN", "No tienes permiso para configurar la instalación.", 403)
		}
	} else if err := service.Read(ctx, p, library, "documents.read"); err != nil {
		return AdvancedConfiguration{}, err
	}
	return service.advancedConfiguration(ctx, service.Database.Reader, library)
}
func (service *Service) ConfigureAdvanced(ctx context.Context, p domain.Principal, library string, input AdvancedConfiguration, m domain.RequestMetadata) error {
	change := func(tx *sql.Tx, current domain.Principal) error {
		previous, err := service.advancedConfiguration(ctx, tx, library)
		if err != nil {
			return err
		}
		if previous.Revision != input.Revision || library != "" && previous.GlobalRevision != input.GlobalRevision || library != "" && previous.PreviewRevision != input.PreviewRevision {
			return domain.Failure("REVISION_CONFLICT", "La configuración cambió. Actualiza antes de guardar.", 409)
		}
		scope := "global"
		value := encode(input.Global)
		effective := input.Global
		if library != "" {
			scope = "library:" + library
			if input.Overrides == nil {
				input.Overrides = map[string]json.RawMessage{}
			}
			value = encode(input.Overrides)
			effective, err = mergeAdvanced(previous.Global, []byte(value))
			if err != nil {
				return invalid("Configuración avanzada inválida.")
			}
		}
		if library == "" {
			effective.PreviewEnabled = previous.Global.PreviewEnabled
			effective.PreviewSourceMB = previous.Global.PreviewSourceMB
			var fields map[string]json.RawMessage
			_ = json.Unmarshal([]byte(value), &fields)
			delete(fields, "preview_enabled")
			delete(fields, "preview_source_mb")
			value = encode(fields)
		}
		if err = effective.validate(); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO advanced_settings VALUES(?,?,?,?) ON CONFLICT(scope) DO UPDATE SET configuration_json=excluded.configuration_json,revision=excluded.revision,updated_at=excluded.updated_at`, scope, value, input.Revision+1, now()); err != nil {
			return err
		}
		// The legacy language column remains a compatibility projection, never a
		// second authority. Existing library APIs can still explicitly override it.
		if library != "" {
			_, err = tx.ExecContext(ctx, "UPDATE libraries SET ocr_languages=?,revision=revision+1 WHERE id=?", effective.OCRLanguages, library)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE libraries SET ocr_languages=coalesce((SELECT json_extract(configuration_json,'$.ocr_languages') FROM advanced_settings WHERE scope='library:'||libraries.id),?),revision=revision+1`, effective.OCRLanguages)
		}
		if err != nil {
			return err
		}
		return record(ctx, tx, current, m, "configuration.advanced_changed", library, "", map[string]any{"scope": scope, "configuration": json.RawMessage(value), "revision": input.Revision + 1})
	}
	if library == "" {
		return service.Identity.AuthorizedWrite(ctx, p, "system.configure", change)
	}
	return service.write(ctx, p, library, "libraries.configure", change)
}
func (policy AdvancedPolicy) ignoredPath(relative string) string {
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		lower := strings.ToLower(part)
		if excludedDirectory(part) {
			return "WATCH_SYSTEM_IGNORED"
		}
		if policy.IgnoreOfficeTemporary && (strings.HasPrefix(lower, "~$") || strings.HasPrefix(lower, ".~lock.")) {
			return "WATCH_OFFICE_TEMPORARY"
		}
		if policy.IgnoreHidden && strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return "WATCH_HIDDEN_IGNORED"
		}
		if policy.IgnoreTemporary && (strings.HasSuffix(lower, "~") || strings.HasSuffix(lower, ".tmp") || strings.HasSuffix(lower, ".temp") || strings.HasSuffix(lower, ".swp")) {
			return "WATCH_TEMPORARY_IGNORED"
		}
		for _, extension := range policy.IgnoredExtensions {
			if strings.EqualFold(filepath.Ext(part), extension) {
				return "WATCH_EXTENSION_IGNORED"
			}
		}
	}
	return ""
}

type manualProcessingKey struct{}
type scanPolicyKey struct{}

// Filter in SQL before LIMIT so a disabled library cannot starve another queue.
const manualJobSQL = `coalesce(json_extract(j.payload_json,'$.manual'),json_extract(j.payload_json,'$.manual_reindex'),0)`
const automaticJobSQL = `coalesce((SELECT json_extract(configuration_json,'$.automatic_processing') FROM advanced_settings WHERE scope='library:'||j.library_id),(SELECT json_extract(configuration_json,'$.automatic_processing') FROM advanced_settings WHERE scope='global'),1)`
const reprocessJobSQL = `coalesce((SELECT json_extract(configuration_json,'$.reprocess_changes') FROM advanced_settings WHERE scope='library:'||j.library_id),(SELECT json_extract(configuration_json,'$.reprocess_changes') FROM advanced_settings WHERE scope='global'),1)`
const priorityJobSQL = `coalesce((SELECT json_extract(configuration_json,'$.manual_priority') FROM advanced_settings WHERE scope='library:'||j.library_id),(SELECT json_extract(configuration_json,'$.manual_priority') FROM advanced_settings WHERE scope='global'),1)`
const advancedJobEligibility = ` AND (j.job_type NOT IN ('scan','extract') OR ` + manualJobSQL + `=1 OR (` + automaticJobSQL + `=1 AND (j.job_type<>'extract' OR ` + reprocessJobSQL + `=1 OR NOT EXISTS(SELECT 1 FROM physical_files WHERE id=j.physical_file_id AND indexed_extraction_id IS NOT NULL)))) `
const advancedJobPriority = `CASE WHEN ` + manualJobSQL + `=1 AND ` + priorityJobSQL + `=1 THEN 0 ELSE 1 END`

func markManualRoot(ctx context.Context, tx *sql.Tx, rootID string) error {
	_, err := tx.ExecContext(ctx, `UPDATE jobs SET payload_json=json_set(payload_json,'$.manual',json('true')) WHERE target_version=? AND job_type IN ('scan','verify_managed') AND status IN ('queued','retry_wait','paused')`, rootID)
	return err
}

func (service *Service) libraryPreviewSettings(ctx context.Context, q storage.Querier, library string) (PreviewSettings, error) {
	settings, err := previewSettings(ctx, q)
	if err != nil {
		return settings, err
	}
	rules, err := service.advancedConfiguration(ctx, q, library)
	if err != nil {
		return settings, err
	}
	settings.Enabled = settings.Enabled && rules.Effective.PreviewEnabled
	settings.MaximumSourceMB = rules.Effective.PreviewSourceMB
	return settings, nil
}
