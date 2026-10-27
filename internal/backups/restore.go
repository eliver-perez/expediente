package backups

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

var extensionPattern = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,10}$`)

func activeLicense(ctx context.Context, license *licensing.Client) error {
	status, err := license.Status(ctx)
	if err != nil {
		return err
	}
	if status.State == "active" || status.State == "grace" || (status.State == "development" && licensing.DevelopmentEnabled()) {
		return nil
	}
	return domain.Failure("RESTORE_LICENSE_REQUIRED", "Se necesita una licencia válida y activa en esta instalación para restaurar.", 403)
}

func safeDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("destination must be absolute")
	}
	for at := filepath.Clean(path); ; at = filepath.Dir(at) {
		info, err := os.Lstat(at)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("destination traverses a link or non-directory")
		}
		if filepath.Dir(at) == at {
			break
		}
	}
	return nil
}
func isWithin(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}

// ScheduleRestore performs no mutation to application data. ApplyPending runs
// on the next startup, while state lock is held and before workers are started.
func (s *Service) ScheduleRestore(ctx context.Context, id, destination, actor string, license *licensing.Client) (Operation, error) {
	return s.ScheduleRestoreWithMappings(ctx, id, destination, nil, actor, license)
}

func (s *Service) ScheduleRestoreWithMappings(ctx context.Context, id, destination string, mappings map[string]string, actor string, license *licensing.Client) (Operation, error) {
	if err := activeLicense(ctx, license); err != nil {
		return Operation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != "" {
		return Operation{}, fmt.Errorf("backup generation in progress")
	}
	op, err := s.Get(id)
	if err != nil {
		return Operation{}, err
	}
	if op.Kind != "restore" || (op.State != "inspected" && op.State != "attention") || op.Actor != actor {
		return Operation{}, fmt.Errorf("restore preview expired or not owned by this administrator")
	}
	list, err := s.allOperations()
	if err != nil {
		return Operation{}, err
	}
	for _, other := range list {
		if other.ID != id && (other.State == "pending_restart" || other.State == "applying") {
			return Operation{}, fmt.Errorf("another restore is pending")
		}
	}
	pkg, err := s.stagedPackage(id)
	if err != nil {
		return Operation{}, err
	}
	defer pkg.Close()
	if op.Manifest == nil || op.Manifest.CreatedAt != pkg.Manifest.CreatedAt || op.Manifest.Kind != pkg.Manifest.Kind {
		return Operation{}, fmt.Errorf("backup changed since inspection")
	}
	if pkg.Manifest.Kind == "full" {
		if err = s.validateDestination(ctx, destination, pkg); err != nil {
			return Operation{}, err
		}
	} else {
		destination = ""
	}
	known := map[string]RootSummary{}
	for _, root := range pkg.Manifest.Roots {
		known[root.ID] = root
	}
	cleanMappings := map[string]string{}
	for id, pathname := range mappings {
		root, exists := known[id]
		if !exists || (pkg.Manifest.Kind == "full" && root.Source == "managed") {
			return Operation{}, fmt.Errorf("invalid root mapping")
		}
		pathname = filepath.Clean(strings.TrimSpace(pathname))
		if err := safeDirectory(pathname); err != nil {
			return Operation{}, fmt.Errorf("root mapping is not an accessible directory: %w", err)
		}
		if isWithin(s.Config.StateDirectory, pathname) || isWithin(pathname, s.Config.StateDirectory) || isWithin(s.Config.PrivateUploadDirectory(), pathname) || isWithin(pathname, s.Config.PrivateUploadDirectory()) || destination != "" && (isWithin(destination, pathname) || isWithin(pathname, destination)) {
			return Operation{}, fmt.Errorf("root mapping overlaps private application data or restore destination")
		}
		cleanMappings[id] = pathname
	}
	op.Destination = destination
	op.Mappings = cleanMappings
	op.State = "pending_restart"
	op.Stage = "Programada; reinicia el servicio para aplicarla"
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err = s.writeOperation(op); err != nil {
		return Operation{}, err
	}
	_ = s.audit(ctx, actor, "backup.restore_scheduled", map[string]any{"operation_id": id, "kind": pkg.Manifest.Kind, "mapped_roots": len(cleanMappings)})
	return op, nil
}

func (s *Service) validateDestination(ctx context.Context, destination string, pkg *Package) error {
	if err := safeDirectory(destination); err != nil {
		return err
	}
	if isWithin(s.Config.StateDirectory, destination) || isWithin(destination, s.Config.StateDirectory) || isWithin(s.Config.PrivateUploadDirectory(), destination) || isWithin(destination, s.Config.PrivateUploadDirectory()) {
		return fmt.Errorf("destination overlaps private application data")
	}
	rows, err := s.DB.Reader.QueryContext(ctx, "SELECT canonical_path FROM storage_roots WHERE status NOT IN ('retired','superseded')")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var root string
		if err = rows.Scan(&root); err != nil {
			return err
		}
		if filepath.IsAbs(root) && (isWithin(root, destination) || isWithin(destination, root)) {
			return fmt.Errorf("destination overlaps an existing documentary root")
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	var needed uint64
	for _, entry := range pkg.Manifest.Entries {
		if strings.HasPrefix(entry.Name, "files/") {
			needed += uint64(entry.Size)
		}
	}
	available, err := storage.AvailableBytes(destination)
	if err != nil {
		return err
	}
	if available < needed+needed/5+64<<20 {
		return fmt.Errorf("insufficient space for recovered documents")
	}
	return nil
}

func (s *Service) audit(ctx context.Context, actor, kind string, details map[string]any) error {
	return s.DB.Write(ctx, func(tx *sql.Tx) error {
		return audit.Append(ctx, tx, time.Now(), audit.Event{Type: kind, ActorUserID: actor, SystemActor: actor == "", Metadata: domain.RequestMetadata{RequestID: domain.NewID(), ObservedIP: "local"}, Details: details})
	})
}

func (s *Service) RecordEvent(ctx context.Context, actor, kind string, details map[string]any) error {
	return s.audit(ctx, actor, kind, details)
}

func (s *Service) ApplyPending(ctx context.Context, license *licensing.Client) error {
	list, err := s.allOperations()
	if err != nil {
		return err
	}
	for _, op := range list {
		if op.State != "pending_restart" && op.State != "applying" {
			continue
		}
		if err := activeLicense(ctx, license); err != nil {
			// Keep the prepared operation for a later valid activation. The normal
			// administration UI must remain available for recovery.
			continue
		}
		if err = s.apply(ctx, &op); err != nil {
			op.State = "attention"
			op.Stage = "Restauración detenida; revisar diagnóstico"
			op.Error = err.Error()
			op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			_ = s.writeOperation(op)
			_ = s.audit(ctx, op.Actor, "backup.restore_failed", map[string]any{"operation_id": op.ID, "error_code": "RESTORE_APPLY_FAILED"})
			continue
		}
	}
	return nil
}

type restoreRoot struct {
	ID, LibraryID, Source, OldPath, Status string
	NewPath, Identity                      string
	Sensitive                              bool
	Verified                               bool
	Revision                               int64
}
type restoreFile struct {
	ID, Root, Filename           string
	NewPath, Relative, Identity  string
	PrimaryLocation              string
	CurrentVersion, Hash         string
	Size                         int64
	Managed, Available, Document bool
	Confirmed                    bool
}
type restoreUpload struct {
	ID, Locator string
	NewLocator  string
}
type restoreIndex struct {
	roots   map[string]*restoreRoot
	files   map[string]*restoreFile
	uploads map[string]*restoreUpload
}

func collectIndex(pkg *Package) (restoreIndex, error) {
	idx := restoreIndex{roots: map[string]*restoreRoot{}, files: map[string]*restoreFile{}, uploads: map[string]*restoreUpload{}}
	err := decodeTable(pkg, "storage_roots", func(row map[string]any) error {
		r := &restoreRoot{ID: stringValue(row["id"]), LibraryID: stringValue(row["library_id"]), Source: stringValue(row["storage_source"]), OldPath: stringValue(row["canonical_path"]), Status: stringValue(row["status"]), Revision: intValue(row["configuration_revision"])}
		if !validID(r.ID) {
			return fmt.Errorf("invalid root identity")
		}
		idx.roots[r.ID] = r
		return nil
	})
	if err != nil {
		return idx, err
	}
	err = decodeTable(pkg, "physical_files", func(row map[string]any) error {
		id := stringValue(row["id"])
		if !validID(id) {
			return fmt.Errorf("invalid file identity")
		}
		idx.files[id] = &restoreFile{ID: id, PrimaryLocation: stringValue(row["primary_location_id"]), CurrentVersion: stringValue(row["current_content_version_id"]), Managed: stringValue(row["storage_source"]) == "managed", Available: stringValue(row["availability"]) == "available"}
		return nil
	})
	if err != nil {
		return idx, err
	}
	err = decodeTable(pkg, "content_versions", func(row map[string]any) error {
		file := idx.files[stringValue(row["physical_file_id"])]
		if file != nil && file.CurrentVersion == stringValue(row["id"]) {
			file.Hash = stringValue(row["sha256"])
			file.Size = intValue(row["size_bytes"])
		}
		return nil
	})
	if err != nil {
		return idx, err
	}
	err = decodeTable(pkg, "physical_file_locations", func(row map[string]any) error {
		relative := stringValue(row["relative_path"])
		if relative == "" || strings.Contains(relative, "\\") || !filepath.IsLocal(filepath.FromSlash(relative)) || filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative))) != relative {
			return fmt.Errorf("invalid relative file location")
		}
		file := idx.files[stringValue(row["physical_file_id"])]
		if file == nil {
			return fmt.Errorf("location without file")
		}
		if row["retired_at"] != nil || stringValue(row["id"]) != file.PrimaryLocation {
			return nil
		}
		id := stringValue(row["physical_file_id"])
		if !validID(id) {
			return fmt.Errorf("invalid file identity")
		}
		file.Root = stringValue(row["root_id"])
		file.Filename = stringValue(row["relative_path"])
		return nil
	})
	if err != nil {
		return idx, err
	}
	err = decodeTable(pkg, "documents", func(row map[string]any) error {
		if row["deleted_at"] == nil {
			if file := idx.files[stringValue(row["physical_file_id"])]; file != nil {
				file.Document = true
			}
		}
		return nil
	})
	if err != nil {
		return idx, err
	}
	err = decodeTable(pkg, "upload_items", func(row map[string]any) error {
		status := stringValue(row["status"])
		if status == "staged" || status == "retained" {
			id := stringValue(row["id"])
			if !validID(id) {
				return fmt.Errorf("invalid upload identity")
			}
			idx.uploads[id] = &restoreUpload{ID: id, Locator: stringValue(row["private_temporary_locator"])}
		}
		return nil
	})
	if err != nil {
		return idx, err
	}
	if pkg.Manifest.Kind == "full" {
		for _, file := range idx.files {
			if file.Managed && file.Available && file.Document && pkg.Files["files/"+file.ID] == nil {
				return idx, fmt.Errorf("managed document missing from complete backup")
			}
		}
		for id := range idx.uploads {
			if pkg.Files["uploads/"+id] == nil {
				return idx, fmt.Errorf("private upload missing from complete backup")
			}
		}
	}
	return idx, nil
}

func (s *Service) apply(ctx context.Context, op *Operation) error {
	var already int
	err := s.DB.Reader.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE event_type='backup.restore_committed' AND json_extract(details_json,'$.operation_id')=?", op.ID).Scan(&already)
	if err != nil {
		return err
	}
	if already > 0 {
		op.State = "complete"
		op.Stage = "Restauración aplicada"
		op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return s.writeOperation(*op)
	}
	pkg, err := s.stagedPackage(op.ID)
	if err != nil {
		return err
	}
	defer pkg.Close()
	idx, err := collectIndex(pkg)
	if err != nil {
		return err
	}
	if pkg.Manifest.Kind == "full" {
		if err = s.validateDestination(ctx, op.Destination, pkg); err != nil {
			return err
		}
	}
	op.State = "applying"
	op.Stage = "Preparando documentos aislados"
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err = s.writeOperation(*op); err != nil {
		return err
	}
	if pkg.Manifest.Kind == "full" {
		if err = s.prepareFiles(ctx, op, pkg, &idx); err != nil {
			return err
		}
	}
	op.Stage = "Comprobando referencias físicas"
	_ = s.writeOperation(*op)
	if err = s.prepareReferences(ctx, op, pkg.Manifest.Kind, &idx); err != nil {
		return err
	}
	op.Stage = "Guardando punto lógico de recuperación"
	_ = s.writeOperation(*op)
	if err = s.writeRecoveryPoint(ctx, op.ID); err != nil {
		return err
	}
	op.Stage = "Aplicando datos lógicos"
	_ = s.writeOperation(*op)
	if err = s.importLogical(ctx, op, pkg, &idx); err != nil {
		return err
	}
	op.State = "complete"
	op.Stage = "Restauración aplicada; inicia sesión nuevamente"
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	_ = os.Remove(filepath.Join(s.directory, op.ID+".plain"))
	_ = os.Remove(filepath.Join(s.directory, op.ID+".incoming"))
	return s.writeOperation(*op)
}

func (s *Service) prepareReferences(ctx context.Context, op *Operation, kind string, idx *restoreIndex) error {
	for _, root := range idx.roots {
		if err := ctx.Err(); err != nil {
			return err
		}
		if kind == "full" && root.Source == "managed" {
			root.Verified = root.NewPath != ""
			continue
		}
		pathname := root.OldPath
		if mapped := op.Mappings[root.ID]; mapped != "" {
			pathname = mapped
		}
		if err := safeDirectory(pathname); err != nil {
			continue
		}
		if isWithin(s.Config.StateDirectory, pathname) || isWithin(pathname, s.Config.StateDirectory) || isWithin(s.Config.PrivateUploadDirectory(), pathname) || isWithin(pathname, s.Config.PrivateUploadDirectory()) {
			continue
		}
		canonical, identity, sensitive, err := libraries.InspectRestoreDirectory(pathname)
		if err != nil {
			continue
		}
		root.NewPath, root.Identity, root.Sensitive, root.Verified = canonical, identity, sensitive, true
	}
	for _, file := range idx.files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if kind == "full" && file.Managed {
			file.Confirmed = file.NewPath != ""
			continue
		}
		root := idx.roots[file.Root]
		if !file.Available || root == nil || !root.Verified {
			continue
		}
		identity, err := libraries.VerifyRestoreReference(root.NewPath, root.Identity, file.Filename, file.Hash, file.Size)
		if err != nil {
			continue
		}
		file.Identity, file.Confirmed = identity, true
	}
	return nil
}

func (s *Service) prepareFiles(ctx context.Context, op *Operation, pkg *Package, idx *restoreIndex) error {
	var err error
	destination := filepath.Join(op.Destination, "aidex-restore-"+op.ID)
	if err := os.Mkdir(destination, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	if err := safeDirectory(destination); err != nil {
		return err
	}
	for _, root := range idx.roots {
		if root.Source != "managed" {
			continue
		}
		root.NewPath = filepath.Join(destination, root.ID)
		if err := os.Mkdir(root.NewPath, 0700); err != nil && !os.IsExist(err) {
			return err
		}
		if err := safeDirectory(root.NewPath); err != nil {
			return err
		}
		root.NewPath, root.Identity, root.Sensitive, err = libraries.InspectRestoreDirectory(root.NewPath)
		if err != nil {
			return err
		}
	}
	for _, entry := range pkg.Manifest.Entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if strings.HasPrefix(entry.Name, "files/") {
			id := strings.TrimPrefix(entry.Name, "files/")
			file := idx.files[id]
			if file == nil {
				return fmt.Errorf("physical file has no location")
			}
			root := idx.roots[file.Root]
			if root == nil || root.Source != "managed" {
				return fmt.Errorf("physical file has no managed root")
			}
			ext := filepath.Ext(file.Filename)
			if !extensionPattern.MatchString(ext) {
				ext = ".document"
			}
			file.Relative = id + ext
			file.NewPath = filepath.Join(root.NewPath, file.Relative)
			if err := publishEntry(pkg.Files[entry.Name], file.NewPath, entry); err != nil {
				return err
			}
			file.Identity, err = libraries.InspectRestoreFile(file.NewPath)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(entry.Name, "uploads/") {
			id := strings.TrimPrefix(entry.Name, "uploads/")
			upload := idx.uploads[id]
			if upload == nil {
				return fmt.Errorf("upload entry has no row")
			}
			upload.NewLocator = "restore-" + op.ID + "-" + id + ".document"
			if err := storage.PrepareLocalPrivateDirectory(s.Config.PrivateUploadDirectory()); err != nil {
				return err
			}
			if err := publishEntry(pkg.Files[entry.Name], filepath.Join(s.Config.PrivateUploadDirectory(), upload.NewLocator), entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func publishEntry(entry *zip.File, target string, expected Entry) error {
	if entry == nil {
		return fmt.Errorf("missing archive file")
	}
	folder, name := filepath.Dir(target), filepath.Base(target)
	if err := safeDirectory(folder); err != nil {
		return err
	}
	container, err := os.OpenRoot(folder)
	if err != nil {
		return err
	}
	defer container.Close()
	if info, err := container.Lstat(name); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination collision")
		}
		file, e := container.Open(name)
		if e != nil {
			return e
		}
		after, e := file.Stat()
		if e != nil || !os.SameFile(info, after) {
			file.Close()
			return fmt.Errorf("destination changed")
		}
		sum := sha256.New()
		size, e := io.Copy(sum, io.LimitReader(file, expected.Size+1))
		file.Close()
		if e != nil || size != expected.Size || hex.EncodeToString(sum.Sum(nil)) != expected.SHA256 {
			return fmt.Errorf("destination collision")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	targetFile, err := container.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	size, err := io.Copy(targetFile, io.TeeReader(io.LimitReader(source, expected.Size+1), hash))
	if syncErr := targetFile.Sync(); err == nil {
		err = syncErr
	}
	if closeErr := targetFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil || size != expected.Size || hex.EncodeToString(hash.Sum(nil)) != expected.SHA256 {
		// Do not unlink: after an interruption the name may have been replaced
		// by an existing original. An orphan in the isolated restore directory
		// is safer than removing a file whose identity is no longer ours.
		return fmt.Errorf("file integrity failure")
	}
	return nil
}
