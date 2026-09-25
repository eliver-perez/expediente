package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type Root struct {
	Source        string `json:"storage_source"`
	ID            string `json:"id"`
	LibraryID     string `json:"library_id"`
	Path          string `json:"server_path,omitempty"`
	Identity      string `json:"-"`
	Status        string `json:"status"`
	WatchMode     string `json:"watch_mode"`
	Revision      int64  `json:"revision"`
	Interval      int    `json:"reconcile_interval_seconds"`
	CaseSensitive bool   `json:"-"`
	LastError     string `json:"last_error_code"`
	LastScan      string `json:"last_scan_at"`
}

const rootColumns = "id,library_id,canonical_path,directory_identity,status,watch_mode,configuration_revision,reconcile_interval_seconds,case_sensitive,coalesce(last_error_code,''),coalesce(last_scan_at,''),storage_source"

func scanRoot(row rowScanner) (Root, error) {
	var root Root
	err := row.Scan(&root.ID, &root.LibraryID, &root.Path, &root.Identity, &root.Status, &root.WatchMode, &root.Revision, &root.Interval, &root.CaseSensitive, &root.LastError, &root.LastScan, &root.Source)
	return root, err
}

func rootsQuery(ctx context.Context, query storage.Querier, libraryID string) ([]Root, error) {
	roots := []Root{}
	rows, err := query.QueryContext(ctx, "SELECT id,library_id,canonical_path,directory_identity,status,watch_mode,configuration_revision,reconcile_interval_seconds,case_sensitive,coalesce(last_error_code,''),coalesce(last_scan_at,''),storage_source FROM storage_roots WHERE (?='' OR library_id=?) ORDER BY id", libraryID, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var root Root
		if err = rows.Scan(&root.ID, &root.LibraryID, &root.Path, &root.Identity, &root.Status, &root.WatchMode, &root.Revision, &root.Interval, &root.CaseSensitive, &root.LastError, &root.LastScan, &root.Source); err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	return roots, rows.Err()
}
func (service *Service) Roots(ctx context.Context, principal domain.Principal, libraryID string) ([]Root, error) {
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, err
	}
	roots, err := rootsQuery(ctx, service.Database.Reader, libraryID)
	if err != nil {
		return nil, err
	}
	canSee := service.require(ctx, service.Database.Reader, principal, libraryID, "storage.view_paths") == nil
	for index := range roots {
		if !canSee {
			roots[index].Path = ""
		}
	}
	return roots, nil
}

type Plan struct {
	Source   string   `json:"storage_source"`
	ID       string   `json:"id"`
	Relation string   `json:"relation"`
	Revision int64    `json:"expected_configuration_revision"`
	Related  []string `json:"related_root_ids"`
	Path     string   `json:"server_path"`
}

func (service *Service) inspect(ctx context.Context, query storage.Querier, libraryID, path string) (directoryInfo, string, []string, error) {
	directory, err := inspectDirectory(path)
	if err != nil {
		return directory, "", nil, err
	}
	protected := []string{service.Identity.Config.StateDirectory, service.Identity.Config.PrivateUploadDirectory()}
	if executable, err := os.Executable(); err == nil {
		protected = append(protected, filepath.Dir(executable))
	}
	if working, err := os.Getwd(); err == nil {
		protected = append(protected, working)
	}
	for _, private := range protected {
		private, err = filepath.EvalSymlinks(private)
		if err == nil && (within(private, directory.Path, directory.CaseSensitive) || within(directory.Path, private, directory.CaseSensitive)) {
			return directory, "", nil, invalid("La raíz se superpone con el programa o su estado privado.")
		}
	}
	roots, err := rootsQuery(ctx, query, "")
	if err != nil {
		return directory, "", nil, err
	}
	relation := "distinct"
	related := []string{}
	for _, root := range roots {
		if root.Status == "superseded" {
			continue
		}
		sensitive := directory.CaseSensitive && root.CaseSensitive
		equal := directory.Identity == root.Identity || comparison(directory.Path, sensitive) == comparison(root.Path, sensitive)
		if currentRoot, err := os.Stat(root.Path); err == nil {
			if selected, err := os.Stat(directory.Path); err == nil && os.SameFile(currentRoot, selected) {
				equal = true
			}
		}
		descendant := within(root.Path, directory.Path, sensitive)
		ancestor := within(directory.Path, root.Path, sensitive)
		// Compare directory handles along ancestors to catch case/Unicode aliases on live volumes.
		for parent := filepath.Dir(directory.Path); parent != filepath.Dir(parent) && !descendant; parent = filepath.Dir(parent) {
			if parentInfo, err := os.Stat(parent); err == nil {
				if rootInfo, err := os.Stat(root.Path); err == nil && os.SameFile(parentInfo, rootInfo) {
					descendant = true
				}
			}
		}
		for parent := filepath.Dir(root.Path); parent != filepath.Dir(parent) && !ancestor; parent = filepath.Dir(parent) {
			if parentInfo, err := os.Stat(parent); err == nil {
				if selectedInfo, err := os.Stat(directory.Path); err == nil && os.SameFile(parentInfo, selectedInfo) {
					ancestor = true
				}
			}
		}
		if !equal && !descendant && !ancestor {
			continue
		}
		if root.LibraryID != libraryID {
			return directory, "", nil, domain.Failure("ROOT_AUTHORIZATION_CONFLICT", "La carpeta se superpone con una raíz registrada. Requiere reorganización autorizada.", 409)
		}
		if equal {
			relation = "equal"
		} else if descendant {
			relation = "descendant"
		} else if relation == "distinct" {
			relation = "ancestor"
		}
		related = append(related, root.ID)
	}
	return directory, relation, related, nil
}
func (service *Service) PlanRoot(ctx context.Context, principal domain.Principal, libraryID, path string, metadata domain.RequestMetadata) (Plan, error) {
	return service.PlanStorageRoot(ctx, principal, libraryID, path, "linked", metadata)
}
func (service *Service) PlanStorageRoot(ctx context.Context, principal domain.Principal, libraryID, path, source string, metadata domain.RequestMetadata) (Plan, error) {
	if source != "linked" && source != "managed" {
		return Plan{}, invalid("Origen de almacenamiento inválido.")
	}

	plan := Plan{ID: domain.NewID(), Source: source}
	err := service.write(ctx, principal, libraryID, "storage.manage_roots", func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.requireRootSource(ctx, transaction, libraryID, source, nil); err != nil {
			return err
		}
		directory, relation, related, err := service.inspect(ctx, transaction, libraryID, path)
		if err != nil {
			return err
		}
		if err = transaction.QueryRowContext(ctx, "SELECT current_configuration_revision FROM libraries WHERE id=?", libraryID).Scan(&plan.Revision); err != nil {
			return err
		}
		if err = service.requireRootSource(ctx, transaction, libraryID, source, related); err != nil {
			return err
		}
		plan.Relation = relation
		plan.Related = related
		plan.Path = directory.Path
		_, err = transaction.ExecContext(ctx, "INSERT INTO root_plans(id,library_id,actor_user_id,server_path,directory_identity,relation,related_root_ids_json,expected_revision,expires_at,storage_source) VALUES(?,?,?,?,?,?,?,?,?,?)", plan.ID, libraryID, current.User.ID, directory.Path, directory.Identity, relation, encode(related), plan.Revision, domain.Timestamp(time.Now().Add(15*time.Minute)), source)
		return err
	})
	return plan, err
}
func (service *Service) ConfirmRoot(ctx context.Context, principal domain.Principal, libraryID, planID string, revision int64, consolidate bool, metadata domain.RequestMetadata) (string, error) {
	rootID := domain.NewID()
	err := service.write(ctx, principal, libraryID, "storage.manage_roots", func(transaction *sql.Tx, current domain.Principal) error {
		var path, identity, relation, relatedJSON, expires, source string
		var expected int64
		var committed sql.NullString
		err := transaction.QueryRowContext(ctx, "SELECT server_path,directory_identity,relation,related_root_ids_json,expected_revision,expires_at,committed_root_id,storage_source FROM root_plans WHERE id=? AND library_id=? AND actor_user_id=?", planID, libraryID, current.User.ID).Scan(&path, &identity, &relation, &relatedJSON, &expected, &expires, &committed, &source)
		if err == sql.ErrNoRows {
			return notFound()
		}
		if err != nil {
			return err
		}
		if committed.Valid {
			rootID = committed.String
			return nil
		}
		var currentRevision int64
		if err = transaction.QueryRowContext(ctx, "SELECT current_configuration_revision FROM libraries WHERE id=?", libraryID).Scan(&currentRevision); err != nil {
			return err
		}
		stale := domain.Failure("ROOT_PLAN_STALE", "La configuración o carpeta cambió. Vuelve a inspeccionarla.", 409)
		if expires < now() || expected != revision || currentRevision != expected {
			return stale
		}
		directory, newRelation, related, err := service.inspect(ctx, transaction, libraryID, path)
		if err != nil {
			return err
		}
		if err = service.requireRootSource(ctx, transaction, libraryID, source, related); err != nil {
			return err
		}
		if source == "managed" {
			if err = probeManagedRoot(directory.Path); err != nil {
				return err
			}
		}
		if directory.Identity != identity || newRelation != relation || encode(related) != relatedJSON {
			return stale
		}
		if relation == "equal" {
			return domain.Failure("ROOT_ALREADY_REGISTERED", "La carpeta ya está registrada.", 409)
		}
		if relation == "descendant" {
			return domain.Failure("ROOT_IS_DESCENDANT", "Crea una vista lógica dentro de la raíz existente.", 409)
		}
		if relation == "ancestor" && !consolidate {
			return domain.Failure("ROOT_CONSOLIDATION_REQUIRED", "Confirma la consolidación de las raíces hijas.", 409)
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO storage_roots(id,library_id,storage_source,canonical_path,comparison_key,volume_identity,case_sensitive,status,watch_mode,reconcile_interval_seconds,created_at,directory_identity) VALUES(?,?,?,?,?,?,?, 'active','polling',900,?,?)", rootID, libraryID, source, path, comparison(path, directory.CaseSensitive), volumeKey(identity), directory.CaseSensitive, now(), identity); err != nil {
			return err
		}
		if relation == "ancestor" {
			if err = service.consolidate(ctx, transaction, libraryID, rootID, path, related); err != nil {
				return err
			}
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO storage_root_configuration_versions VALUES(?,1,?,?,?, ?,?)", rootID, path, comparison(path, directory.CaseSensitive), encode(map[string]any{"source": source}), now(), current.User.ID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE libraries SET current_configuration_revision=current_configuration_revision+1,revision=revision+1 WHERE id=?", libraryID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE root_plans SET committed_root_id=? WHERE id=?", rootID, planID); err != nil {
			return err
		}
		if source == "linked" {
			if err = enqueueScan(ctx, transaction, libraryID, rootID); err != nil {
				return err
			}
		} else {
			if _, err = transaction.ExecContext(ctx, "UPDATE storage_roots SET watch_mode='paused' WHERE id=?", rootID); err != nil {
				return err
			}
		}
		return record(ctx, transaction, current, metadata, "storage.root_added", libraryID, "", map[string]any{"root_id": rootID, "storage_source": source, "consolidated_root_ids": related})
	})
	return rootID, err
}
func (service *Service) consolidate(ctx context.Context, transaction *sql.Tx, libraryID, newRootID, parent string, children []string) error {
	for _, childID := range children {
		var childPath string
		if err := transaction.QueryRowContext(ctx, "SELECT canonical_path FROM storage_roots WHERE id=?", childID).Scan(&childPath); err != nil {
			return err
		}
		rows, err := transaction.QueryContext(ctx, "SELECT id,physical_file_id,relative_path,canonical_path,comparison_key,first_seen_at,last_seen_at FROM physical_file_locations WHERE root_id=? AND retired_at IS NULL", childID)
		if err != nil {
			return err
		}
		type location struct{ ID, FileID, Relative, Path, Key, First, Last string }
		locations := []location{}
		for rows.Next() {
			var entry location
			if err = rows.Scan(&entry.ID, &entry.FileID, &entry.Relative, &entry.Path, &entry.Key, &entry.First, &entry.Last); err != nil {
				rows.Close()
				return err
			}
			locations = append(locations, entry)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		prefix, err := filepath.Rel(parent, childPath)
		if err != nil {
			return err
		}
		for _, entry := range locations {
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_file_locations SET retired_at=? WHERE id=?", now(), entry.ID); err != nil {
				return err
			}
			identifier := domain.NewID()
			relative := filepath.ToSlash(filepath.Join(prefix, filepath.FromSlash(entry.Relative)))
			if _, err = transaction.ExecContext(ctx, "INSERT INTO physical_file_locations VALUES(?,?,?,'linked',?,?,?,?,1,?,?,NULL)", identifier, entry.FileID, libraryID, newRootID, relative, entry.Path, entry.Key, entry.First, entry.Last); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_files SET primary_location_id=? WHERE id=? AND primary_location_id=?", identifier, entry.FileID, entry.ID); err != nil {
				return err
			}
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE logical_folder_views SET root_id=?,relative_prefix=? || CASE WHEN relative_prefix='' THEN '' ELSE '/' || relative_prefix END WHERE root_id=?", newRootID, filepath.ToSlash(prefix), childID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE storage_roots SET status='superseded',watch_mode='paused',configuration_revision=configuration_revision+1 WHERE id=?", childID); err != nil {
			return err
		}
	}
	return nil
}
func (service *Service) AddView(ctx context.Context, principal domain.Principal, libraryID, rootID, prefix, name string, metadata domain.RequestMetadata) (string, error) {
	if !relativeSafe(prefix) || strings.TrimSpace(name) == "" || len(name) > 160 {
		return "", invalid("Escribe un nombre y subcarpeta relativa válidos.")
	}
	identifier := domain.NewID()
	err := service.write(ctx, principal, libraryID, "storage.manage_roots", func(transaction *sql.Tx, current domain.Principal) error {
		var count int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM storage_roots WHERE id=? AND library_id=? AND status<>'superseded'", rootID, libraryID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return notFound()
		}
		_, err := transaction.ExecContext(ctx, "INSERT INTO logical_folder_views VALUES(?,?,?,?,?,?) ON CONFLICT(library_id,root_id,relative_prefix) DO NOTHING", identifier, libraryID, rootID, strings.TrimSpace(name), prefix, now())
		if err != nil {
			return err
		}
		if err = transaction.QueryRowContext(ctx, "SELECT id FROM logical_folder_views WHERE library_id=? AND root_id=? AND relative_prefix=?", libraryID, rootID, prefix).Scan(&identifier); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "storage.view_created", libraryID, "", map[string]any{"view_id": identifier, "root_id": rootID})
	})
	return identifier, err
}
func enqueueScan(ctx context.Context, transaction *sql.Tx, libraryID, rootID string) error {
	var count int
	if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM jobs WHERE job_type='scan' AND target_version=? AND status IN ('queued','running','retry_wait')", rootID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	identifier := domain.NewID()
	_, err := transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,'scan',?,?,?,'queued',?,?)", identifier, libraryID, rootID, identifier, encode(map[string]string{"root_id": rootID}), now(), now())
	return err
}
func decodeStrings(contents string) ([]string, error) {
	var result []string
	if err := json.Unmarshal([]byte(contents), &result); err != nil {
		return nil, err
	}
	for _, directory := range result {
		if directory != "." && !relativeSafe(directory) {
			return nil, invalid("Checkpoint de escaneo inválido.")
		}
	}
	return result, nil
}

func (service *Service) requireRootSource(ctx context.Context, query storage.Querier, libraryID, source string, related []string) error {
	_, mode, err := settingsFor(ctx, query, libraryID)
	if err != nil {
		return err
	}
	if source == "managed" && mode == "linked" || source == "linked" && mode == "managed" {
		return invalid("Esta modalidad no admite ese origen; amplía la biblioteca a híbrida.")
	}
	for _, identifier := range related {
		var existing string
		if err = query.QueryRowContext(ctx, "SELECT storage_source FROM storage_roots WHERE id=?", identifier).Scan(&existing); err != nil {
			return err
		}
		if existing != source || source == "managed" {
			return domain.Failure("ROOT_STORAGE_OVERLAP", "Los destinos administrados deben estar separados de cualquier raíz existente.", 409)
		}
	}
	return nil
}
func probeManagedRoot(path string) error {
	root, err := os.OpenRoot(path)
	if err != nil {
		return invalid("No se puede abrir el destino administrado.")
	}
	defer root.Close()
	name := ".documental-probe-" + domain.NewID()
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return invalid("El destino necesita permisos de escritura.")
	}
	defer root.Remove(name)
	_, writeErr := file.Write([]byte("probe"))
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return invalid("No se pudo verificar escritura en el destino.")
	}
	return nil
}
