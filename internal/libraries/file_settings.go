package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type FileConfiguration struct {
	Defaults       documentformat.Policy    `json:"defaults"`
	Global         documentformat.Policy    `json:"global"`
	Overrides      documentformat.Overrides `json:"overrides"`
	Effective      documentformat.Policy    `json:"effective"`
	Revision       int64                    `json:"revision"`
	GlobalRevision int64                    `json:"global_revision"`
	Formats        []documentformat.Format  `json:"formats"`
}

func (service *Service) fileConfiguration(ctx context.Context, query storage.Querier, libraryID string) (FileConfiguration, error) {
	result := FileConfiguration{Defaults: documentformat.Default(service.Identity.Config.Indexing.MaximumFileMB), Formats: documentformat.Formats()}
	read := func(scope string, target any, revision *int64) error {
		var raw string
		err := query.QueryRowContext(ctx, "SELECT configuration_json,revision FROM document_file_settings WHERE scope=?", scope).Scan(&raw, revision)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(raw), target)
	}
	result.Global = result.Defaults
	if err := read("global", &result.Global, &result.GlobalRevision); err != nil {
		return result, err
	}
	result.Revision = result.GlobalRevision
	result.Effective = result.Global
	if libraryID != "" {
		result.Revision = 0
		if err := read("library:"+libraryID, &result.Overrides, &result.Revision); err != nil {
			return result, err
		}
		result.Effective = documentformat.Resolve(result.Global, result.Overrides)
	}
	return result, nil
}

func (service *Service) FileConfiguration(ctx context.Context, principal domain.Principal, libraryID string) (FileConfiguration, error) {
	if libraryID == "" {
		if !principal.Can("system.configure") {
			return FileConfiguration{}, domain.Failure("FORBIDDEN", "No tienes permiso para configurar archivos.", 403)
		}
	} else {
		if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
			return FileConfiguration{}, err
		}
	}
	return service.fileConfiguration(ctx, service.Database.Reader, libraryID)
}

func (service *Service) ConfigureFiles(ctx context.Context, principal domain.Principal, libraryID string, configuration documentformat.Policy, overrides documentformat.Overrides, revision, globalRevision int64, metadata domain.RequestMetadata) error {
	change := func(tx *sql.Tx, current domain.Principal) error {
		previous, err := service.fileConfiguration(ctx, tx, libraryID)
		if err != nil {
			return err
		}
		if previous.Revision != revision || libraryID != "" && previous.GlobalRevision != globalRevision {
			return domain.Failure("REVISION_CONFLICT", "La configuración cambió. Actualiza antes de guardar.", 409)
		}
		scope := "global"
		var value any = configuration
		if libraryID != "" {
			scope = "library:" + libraryID
			value = overrides
			configuration = documentformat.Resolve(previous.Global, overrides)
		}
		if err = configuration.Validate(); err != nil {
			return err
		}
		if libraryID == "" {
			rows, err := tx.QueryContext(ctx, "SELECT configuration_json FROM document_file_settings WHERE scope LIKE 'library:%'")
			if err != nil {
				return err
			}
			for rows.Next() {
				var raw string
				var override documentformat.Overrides
				if err = rows.Scan(&raw); err == nil {
					err = json.Unmarshal([]byte(raw), &override)
				}
				if err == nil {
					err = documentformat.Resolve(configuration, override).Validate()
				}
				if err != nil {
					rows.Close()
					return invalid("Una biblioteca tiene una excepción incompatible con estos valores. Ajusta primero su configuración de archivos.")
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO document_file_settings VALUES(?,?,?,?) ON CONFLICT(scope) DO UPDATE SET configuration_json=excluded.configuration_json,revision=excluded.revision,updated_at=excluded.updated_at`, scope, encode(value), revision+1, now()); err != nil {
			return err
		}
		if err = service.queueNewlyIndexable(ctx, tx, libraryID); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "configuration.files_changed", libraryID, "", map[string]any{"scope": scope, "configuration": value, "revision": revision + 1})
	}
	if libraryID == "" {
		return service.Identity.AuthorizedWrite(ctx, principal, "system.configure", change)
	}
	return service.write(ctx, principal, libraryID, "libraries.configure", change)
}

func saveDetection(ctx context.Context, tx *sql.Tx, version string, detection documentformat.Detection, indexReason string) error {
	_, err := tx.ExecContext(ctx, "UPDATE content_versions SET document_format=?,detected_mime=?,original_extension=?,extension_mismatch=?,index_block_reason=? WHERE id=?", detection.Format, detection.MIME, detection.Extension, detection.Mismatch, indexReason, version)
	return err
}

func (service *Service) versionIndexReason(ctx context.Context, query storage.Querier, libraryID, versionID string) (string, error) {
	settings, err := service.fileConfiguration(ctx, query, libraryID)
	if err != nil {
		return "", err
	}
	var detection documentformat.Detection
	var size int64
	err = query.QueryRowContext(ctx, "SELECT document_format,detected_mime,original_extension,extension_mismatch,size_bytes FROM content_versions WHERE id=?", versionID).Scan(&detection.Format, &detection.MIME, &detection.Extension, &detection.Mismatch, &size)
	if err != nil {
		return "", err
	}
	return settings.Effective.IndexReason(detection, size), nil
}
