package libraries

import (
	"context"
	"database/sql"
	"path/filepath"

	"gestor-documental/internal/domain"
)

// Validar la asignación antes de disponer de permisos para inspeccionar el
// contenido: una denegación no permite autorizar otra biblioteca por accidente.
func (service *Service) ValidateFolderAuthorizationScope(ctx context.Context, p domain.Principal, libraryID, path, source string) error {
	return service.write(ctx, p, libraryID, "storage.manage_roots", func(tx *sql.Tx, current domain.Principal) error {
		if err := service.requireRootSource(ctx, tx, libraryID, source, nil); err != nil {
			return err
		}
		for _, private := range []string{service.Identity.Config.StateDirectory, service.Identity.Config.PrivateUploadDirectory()} {
			canonical, err := filepath.EvalSymlinks(private)
			if err != nil {
				canonical = filepath.Clean(private)
			}
			if within(canonical, path, false) || within(path, canonical, false) {
				return invalid("La autorización se superpone con el estado privado de AIDEX; no se cambiaron permisos.")
			}
		}
		rows, err := tx.QueryContext(ctx, "SELECT library_id,canonical_path,storage_source FROM storage_roots WHERE status<>'superseded'")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var library, registered, origin string
			if err = rows.Scan(&library, &registered, &origin); err != nil {
				return err
			}
			if (within(filepath.Clean(registered), path, false) || within(path, filepath.Clean(registered), false)) && (library != libraryID || source != origin) {
				return domain.Failure("ROOT_AUTHORIZATION_CONFLICT", "La autorización se superpone con otra biblioteca o un origen distinto; no se cambiaron permisos.", 409)
			}
		}
		return rows.Err()
	})
}
