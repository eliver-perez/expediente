package libraries

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type PreviewSettings struct {
	Enabled          bool   `json:"enabled"`
	MaximumCacheMB   int    `json:"maximum_cache_mb"`
	MaximumIdleDays  int    `json:"maximum_idle_days"`
	MaximumSourceMB  int    `json:"maximum_source_mb"`
	AutomaticCleanup bool   `json:"automatic_cleanup"`
	ConverterPath    string `json:"converter_path"`
	Revision         int64  `json:"revision"`
	UsedBytes        int64  `json:"used_bytes"`
	Entries          int    `json:"entries"`
}

func previewSettings(ctx context.Context, query storage.Querier) (PreviewSettings, error) {
	var settings PreviewSettings
	err := query.QueryRowContext(ctx, `SELECT enabled,maximum_cache_mb,maximum_idle_days,maximum_source_mb,automatic_cleanup,converter_path,revision,(SELECT coalesce(sum(size_bytes),0) FROM preview_cache),(SELECT count(*) FROM preview_cache WHERE status='ready') FROM preview_settings WHERE singleton=1`).Scan(&settings.Enabled, &settings.MaximumCacheMB, &settings.MaximumIdleDays, &settings.MaximumSourceMB, &settings.AutomaticCleanup, &settings.ConverterPath, &settings.Revision, &settings.UsedBytes, &settings.Entries)
	return settings, err
}
func (service *Service) PreviewSettings(ctx context.Context) (PreviewSettings, error) {
	return previewSettings(ctx, service.Database.Reader)
}
func (service *Service) ConfigurePreviews(ctx context.Context, principal domain.Principal, settings PreviewSettings, metadata domain.RequestMetadata) error {
	if settings.MaximumCacheMB < 1 || settings.MaximumCacheMB > 20480 || settings.MaximumIdleDays < 1 || settings.MaximumIdleDays > 365 || settings.MaximumSourceMB < 1 || settings.MaximumSourceMB > 512 || (settings.ConverterPath != "" && !filepath.IsAbs(settings.ConverterPath)) {
		return invalid("Comprueba los límites de caché, antigüedad y tamaño; la ruta del conversor debe ser absoluta.")
	}
	return service.Identity.AuthorizedWrite(ctx, principal, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
		result, err := tx.ExecContext(ctx, `UPDATE preview_settings SET enabled=?,maximum_cache_mb=?,maximum_idle_days=?,maximum_source_mb=?,automatic_cleanup=?,converter_path=?,revision=revision+1 WHERE singleton=1 AND revision=?`, settings.Enabled, settings.MaximumCacheMB, settings.MaximumIdleDays, settings.MaximumSourceMB, settings.AutomaticCleanup, settings.ConverterPath, settings.Revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("REVISION_CONFLICT", "La configuración cambió. Actualiza antes de guardar.", 409)
		}
		if err = service.trimPreviews(ctx, tx, settings, 0, settings.AutomaticCleanup); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "system.previews_configured", "", "", map[string]any{"enabled": settings.Enabled, "maximum_cache_mb": settings.MaximumCacheMB, "maximum_idle_days": settings.MaximumIdleDays, "maximum_source_mb": settings.MaximumSourceMB, "automatic_cleanup": settings.AutomaticCleanup})
	})
}
func (service *Service) previewDirectory() string {
	return filepath.Join(service.Identity.Config.StateDirectory, "previews")
}
func (service *Service) previewWorkDirectory() string {
	return filepath.Join(service.Identity.Config.StateDirectory, "preview-work")
}
func (service *Service) previewPath(id string) string {
	return filepath.Join(service.previewDirectory(), id+".preview")
}

// Only generated IDs address this private directory. No document filename or root
// path participates in cache deletion. The writer transaction serializes eviction,
// publication and opening, also across the HTTP and runtime Service instances.
func (service *Service) deletePreview(ctx context.Context, tx *sql.Tx, id string) error {
	if filepath.Base(id) != id || id == "" || id == "." || id == ".." {
		return scanFailure("PREVIEW_STORAGE_ERROR")
	}
	if err := os.Remove(service.previewPath(id)); err != nil && !os.IsNotExist(err) {
		return scanFailure("PREVIEW_CACHE_BUSY")
	}
	_, err := tx.ExecContext(ctx, "DELETE FROM preview_cache WHERE id=?", id)
	return err
}
func (service *Service) trimPreviews(ctx context.Context, tx *sql.Tx, settings PreviewSettings, incoming int64, expire bool) error {
	rows, err := tx.QueryContext(ctx, "SELECT id,size_bytes,last_accessed_at FROM preview_cache WHERE status='ready' ORDER BY last_accessed_at,id")
	if err != nil {
		return err
	}
	type entry struct {
		id       string
		size     int64
		accessed string
	}
	entries := []entry{}
	var total int64
	for rows.Next() {
		var item entry
		if err = rows.Scan(&item.id, &item.size, &item.accessed); err != nil {
			rows.Close()
			return err
		}
		total += item.size
		entries = append(entries, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	limit := int64(settings.MaximumCacheMB) << 20
	if incoming > limit {
		return scanFailure("PREVIEW_SIZE_LIMIT")
	}
	cutoff := domain.Timestamp(time.Now().Add(-time.Duration(settings.MaximumIdleDays) * 24 * time.Hour))
	for _, item := range entries {
		// Recover a missing cache file after an interrupted eviction/publication.
		// Originals and indexed content never participate in this repair.
		if _, err := os.Lstat(service.previewPath(item.id)); os.IsNotExist(err) {
			if err = service.deletePreview(ctx, tx, item.id); err != nil {
				return err
			}
			total -= item.size
			continue
		}
		if total+incoming <= limit && (!expire || item.accessed >= cutoff) {
			continue
		}
		if err = service.deletePreview(ctx, tx, item.id); err != nil {
			return err
		}
		total -= item.size
	}
	return nil
}
func (service *Service) ClearPreviews(ctx context.Context, principal domain.Principal, metadata domain.RequestMetadata) error {
	return service.Identity.AuthorizedWrite(ctx, principal, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
		rows, err := tx.QueryContext(ctx, "SELECT id FROM preview_cache")
		if err != nil {
			return err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err = service.deletePreview(ctx, tx, id); err != nil {
				return err
			}
		}
		return record(ctx, tx, current, metadata, "system.preview_cache_cleared", "", "", map[string]any{"entries": len(ids)})
	})
}
