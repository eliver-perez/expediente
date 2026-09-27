package libraries

import (
	"context"
	"database/sql"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/extraction"
	"time"
)

type ProcessingConfiguration struct {
	Configuration extraction.Concurrency `json:"configuration"`
	Effective     extraction.Concurrency `json:"effective"`
	Resources     extraction.Resources   `json:"resources"`
	Revision      int64                  `json:"revision"`
}

func (service *Service) ProcessingConfiguration(ctx context.Context) (ProcessingConfiguration, error) {
	service.processingMutex.Lock()
	defer service.processingMutex.Unlock()
	if time.Since(service.processingCachedAt) < 5*time.Second {
		return service.processingCached, nil
	}
	c := ProcessingConfiguration{Resources: service.resources, Configuration: service.Identity.Config.Indexing.Concurrency}
	if c.Configuration.Mode == "" {
		c.Configuration = extraction.Concurrency{Mode: "auto"}
		if n := service.Identity.Config.Indexing.Workers; n > 1 {
			c.Configuration = extraction.Concurrency{Mode: "manual", Native: n, OCR: 1, Total: n}
		}
	}
	err := service.Database.Reader.QueryRowContext(ctx, "SELECT mode,native_workers,ocr_workers,total_workers,revision FROM processing_settings WHERE singleton=1").Scan(&c.Configuration.Mode, &c.Configuration.Native, &c.Configuration.OCR, &c.Configuration.Total, &c.Revision)
	if err != nil && err != sql.ErrNoRows {
		return c, err
	}
	c.Effective = c.Configuration
	if c.Configuration.Mode == "auto" {
		c.Effective = extraction.Recommend(c.Resources)
	}
	service.processingCached = c
	service.processingCachedAt = time.Now()
	return c, nil
}
func (service *Service) ConfigureProcessing(ctx context.Context, principal domain.Principal, c extraction.Concurrency, revision int64, metadata domain.RequestMetadata) error {
	if err := c.Validate(); err != nil {
		return invalid(err.Error())
	}
	if c.Mode == "auto" {
		c = extraction.Recommend(service.resources)
	}
	err := service.Identity.AuthorizedWrite(ctx, principal, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
		var actual int64
		if err := tx.QueryRowContext(ctx, "SELECT coalesce((SELECT revision FROM processing_settings WHERE singleton=1),0)").Scan(&actual); err != nil {
			return err
		}
		if actual != revision {
			return domain.Failure("REVISION_CONFLICT", "La configuración cambió. Actualiza antes de guardar.", 409)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO processing_settings VALUES(1,?,?,?,?,?,?) ON CONFLICT(singleton) DO UPDATE SET mode=excluded.mode,native_workers=excluded.native_workers,ocr_workers=excluded.ocr_workers,total_workers=excluded.total_workers,revision=excluded.revision,updated_at=excluded.updated_at`, c.Mode, c.Native, c.OCR, c.Total, actual+1, now()); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "system.processing_configured", "", "", map[string]any{"configuration": c, "revision": actual + 1})
	})
	if err == nil {
		service.processingMutex.Lock()
		service.processingCachedAt = time.Time{}
		service.processingMutex.Unlock()
	}
	return err
}
