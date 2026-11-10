package libraries

import (
	"context"
	"database/sql"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

func validateManagedDestination(ctx context.Context, q storage.Querier, libraryID, rootID string) error {
	root, err := scanRoot(q.QueryRowContext(ctx, "SELECT "+rootColumns+" FROM storage_roots WHERE id=? AND library_id=? AND storage_source='managed' AND status='active'", rootID, libraryID))
	if err == sql.ErrNoRows {
		return invalid("Selecciona un destino administrado activo de esta biblioteca.")
	}
	if err != nil {
		return err
	}
	info, err := inspectDirectory(root.Path)
	if err != nil {
		return err
	}
	if info.Identity != root.Identity {
		return domain.Failure("ROOT_UNAVAILABLE", "La carpeta del destino cambió. Comprueba su acceso antes de seleccionarla.", 409)
	}
	return probeManagedRoot(root.Path)
}

// Cambia únicamente la configuración. Nunca mueve los documentos incorporados.
func storeManagedDestination(ctx context.Context, tx *sql.Tx, p domain.Principal, libraryID, previous, next string, metadata domain.RequestMetadata) error {
	if _, err := tx.ExecContext(ctx, "UPDATE libraries SET settings_json=json_set(settings_json,'$.default_managed_root_id',?) WHERE id=?", next, libraryID); err != nil {
		return err
	}
	return record(ctx, tx, p, metadata, "storage.destination_changed", libraryID, "", map[string]any{"previous_root_id": previous, "new_root_id": next})
}

func clearManagedDestination(ctx context.Context, tx *sql.Tx, p domain.Principal, libraryID, rootID string, metadata domain.RequestMetadata) error {
	settings, _, err := settingsFor(ctx, tx, libraryID)
	if err != nil {
		return err
	}
	if settings.ManagedRootID != rootID {
		return nil
	}
	return storeManagedDestination(ctx, tx, p, libraryID, rootID, "", metadata)
}

func (service *Service) SetManagedDestination(ctx context.Context, principal domain.Principal, libraryID, rootID string, revision int64, metadata domain.RequestMetadata) error {
	return service.write(ctx, principal, libraryID, "storage.manage_roots", func(tx *sql.Tx, current domain.Principal) error {
		settings, mode, err := settingsFor(ctx, tx, libraryID)
		if err != nil {
			return err
		}
		if mode == "linked" {
			return invalid("Esta biblioteca no admite destinos administrados.")
		}
		if err = service.Identity.License.CheckFeatures(ctx, modeCapabilities(mode)...); err != nil {
			return err
		}
		var live int64
		if err = tx.QueryRowContext(ctx, "SELECT revision FROM libraries WHERE id=?", libraryID).Scan(&live); err != nil {
			return err
		}
		if live != revision {
			return domain.Failure("VERSION_CONFLICT", "La biblioteca cambió. Recarga la página.", 412)
		}
		if err = validateManagedDestination(ctx, tx, libraryID, rootID); err != nil {
			return err
		}
		if settings.ManagedRootID == rootID {
			return nil
		}
		if err = storeManagedDestination(ctx, tx, current, libraryID, settings.ManagedRootID, rootID, metadata); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE libraries SET revision=revision+1,current_configuration_revision=current_configuration_revision+1,updated_at=? WHERE id=?", now(), libraryID); err != nil {
			return err
		}
		return configurationSnapshot(ctx, tx, libraryID, current.User.ID)
	})
}
