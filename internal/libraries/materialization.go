package libraries

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type Materialization struct {
	ID                    string `json:"id"`
	DocumentID            string `json:"document_id"`
	LibraryID             string `json:"library_id"`
	FileID                string `json:"-"`
	VersionID             string `json:"-"`
	Revision              int64  `json:"-"`
	RootID                string `json:"-"`
	RootRevision          int64  `json:"-"`
	ConfigurationRevision int64  `json:"configuration_revision"`
	Relative              string `json:"-"`
	Comparison            string `json:"-"`
	Partial               string `json:"-"`
	Hash                  string `json:"-"`
	Size                  int64  `json:"-"`
	State                 string `json:"state"`
	Identity              string `json:"-"`
	ApprovedBy            string `json:"approved_by"`
	DecisionKind          string `json:"decision_kind"`
	ReviewID              string `json:"review_id"`
	Error                 string `json:"error_code"`
}

const materializationColumns = "id,document_id,library_id,physical_file_id,content_version_id,document_revision,target_root_id,root_revision,library_configuration_revision,target_relative_path,target_comparison_key,partial_relative_path,expected_sha256,expected_size_bytes,state,published_identity,approved_by,decision_kind,coalesce(review_id,''),error_code"

func scanMaterialization(row rowScanner) (Materialization, error) {
	var operation Materialization
	err := row.Scan(&operation.ID, &operation.DocumentID, &operation.LibraryID, &operation.FileID, &operation.VersionID, &operation.Revision, &operation.RootID, &operation.RootRevision, &operation.ConfigurationRevision, &operation.Relative, &operation.Comparison, &operation.Partial, &operation.Hash, &operation.Size, &operation.State, &operation.Identity, &operation.ApprovedBy, &operation.DecisionKind, &operation.ReviewID, &operation.Error)
	return operation, err
}
func hashFile(ctx context.Context, file *os.File, maximum int64) (string, int64, error) {
	before, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	if !before.Mode().IsRegular() || before.Size() > maximum {
		return "", 0, scanFailure("FILE_SIZE_LIMIT")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	digest := sha256.New()
	buffer := make([]byte, 128<<10)
	var size int64
	for {
		if err = ctx.Err(); err != nil {
			return "", 0, err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			size += int64(count)
			if size > maximum {
				return "", size, scanFailure("FILE_SIZE_LIMIT")
			}
			_, _ = digest.Write(buffer[:count])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", 0, readErr
		}
	}
	after, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	if size != before.Size() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return "", 0, scanFailure("FILE_UNSTABLE")
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), size, nil
}
func (service *Service) verifyDecisionContent(ctx context.Context, document Document) error {
	var file *os.File
	var err error
	if document.Availability == "staged" {
		file, err = service.openUpload(ctx, document.FileID)
	} else if document.Availability == "available" {
		var root Root
		root, err = service.root(ctx, document.RootID)
		if err == nil {
			file, err = openLinked(root, document.RelativePath)
		}
	} else {
		return scanFailure("DOCUMENT_UNAVAILABLE")
	}
	if err != nil {
		return scanFailure("DOCUMENT_UNAVAILABLE")
	}
	defer file.Close()
	digest, _, err := hashFile(ctx, file, int64(service.Identity.Config.Indexing.MaximumFileMB)<<20)
	if err != nil {
		return err
	}
	identity, _, err := physicalIdentity(file)
	if err != nil {
		return err
	}
	if digest != document.Hash || identity != document.Identity {
		return domain.Failure("CONTENT_CHANGED", "El archivo cambió. Verifica la biblioteca y vuelve a revisar el documento.", 409)
	}
	if document.Source == "managed" && document.Integrity == "changed" {
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		directory := filepath.Join(service.Identity.Config.StateDirectory, "extraction")
		if err = storage.PreparePrivateDirectory(directory); err != nil {
			return err
		}
		snapshot, err := os.CreateTemp(directory, "decision-*.pdf")
		if err != nil {
			return err
		}
		defer os.Remove(snapshot.Name())
		if err = storage.ProtectPrivatePath(snapshot.Name(), false); err != nil {
			snapshot.Close()
			return err
		}
		hash := sha256.New()
		_, copyErr := copyDocument(ctx, io.MultiWriter(snapshot, hash), io.LimitReader(file, (int64(service.Identity.Config.Indexing.MaximumFileMB)<<20)+1))
		closeErr := snapshot.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if fmt.Sprintf("%x", hash.Sum(nil)) != document.Hash {
			return scanFailure("FILE_UNSTABLE")
		}
		if _, err = service.Identity.Config.Indexing.ValidatePDF(ctx, snapshot.Name()); err != nil {
			return err
		}
	}

	return nil
}
func (service *Service) beginApproval(ctx context.Context, transaction *sql.Tx, principal domain.Principal, document Document, settings Settings, kind, reviewID string, metadata domain.RequestMetadata) (string, error) {
	if document.Source != "managed" || document.Availability != "staged" {
		if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET integrity_status='verified' WHERE id=?", document.FileID); err != nil {
			return "", err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE documents SET approval_status='approved',approved_content_version_id=(SELECT current_content_version_id FROM physical_files WHERE id=?),approved_by=?,approved_at=?,revision=revision+1 WHERE id=?", document.FileID, principal.User.ID, now(), document.ID); err != nil {
			return "", err
		}
		if reviewID != "" {
			if _, err := transaction.ExecContext(ctx, "UPDATE review_requests SET status='approved',decided_by=?,decided_at=? WHERE id=?", principal.User.ID, now(), reviewID); err != nil {
				return "", err
			}
		}
		return "", workflowEvent(ctx, transaction, principal, document, metadata, "document.approved", map[string]any{"decision_kind": kind, "review_id": reviewID})
	}
	if settings.ManagedRootID == "" {
		return "", domain.Failure("MANAGED_ROOT_REQUIRED", "Selecciona un destino administrado en Configuración.", 422)
	}
	root, err := scanRoot(transaction.QueryRowContext(ctx, "SELECT "+rootColumns+" FROM storage_roots WHERE id=? AND library_id=? AND storage_source='managed' AND status='active'", settings.ManagedRootID, document.LibraryID))
	if err == sql.ErrNoRows {
		return "", invalid("El destino administrado no está activo.")
	}
	if err != nil {
		return "", err
	}
	operationID := domain.NewID()
	var configurationRevision int64
	if err = transaction.QueryRowContext(ctx, "SELECT current_configuration_revision FROM libraries WHERE id=?", document.LibraryID).Scan(&configurationRevision); err != nil {
		return "", err
	}
	if _, err = transaction.ExecContext(ctx, "INSERT OR IGNORE INTO library_configuration_versions SELECT id,current_configuration_revision,mode,settings_json,?,? FROM libraries WHERE id=?", principal.User.ID, now(), document.LibraryID); err != nil {
		return "", err
	}
	if _, err = transaction.ExecContext(ctx, "INSERT INTO library_document_sequences VALUES(?,1) ON CONFLICT DO NOTHING", document.LibraryID); err != nil {
		return "", err
	}
	var sequence int64
	if err = transaction.QueryRowContext(ctx, "SELECT next_number FROM library_document_sequences WHERE library_id=?", document.LibraryID).Scan(&sequence); err != nil {
		return "", err
	}
	if _, err = transaction.ExecContext(ctx, "UPDATE library_document_sequences SET next_number=next_number+1 WHERE library_id=?", document.LibraryID); err != nil {
		return "", err
	}
	input := NamingInput{Identifier: document.CaseIdentifier, Category: document.CategoryName, DocumentType: document.TypeName, OriginalFilename: document.Filename}
	if document.CaseID != "" {
		if err = transaction.QueryRowContext(ctx, "SELECT coalesce(exercise,'') FROM cases WHERE id=?", document.CaseID).Scan(&input.Exercise); err != nil {
			return "", err
		}
	}
	relative := renderNamingSequence(settings, input, document.ID, sequence)
	partial := filepath.ToSlash(filepath.Join(filepath.Dir(relative), ".aibid-"+operationID+".part"))
	_, err = transaction.ExecContext(ctx, `INSERT INTO materializations(id,document_id,library_id,physical_file_id,content_version_id,document_revision,target_root_id,root_revision,library_configuration_revision,target_relative_path,target_comparison_key,partial_relative_path,expected_sha256,expected_size_bytes,state,approved_by,decision_kind,review_id,created_at,updated_at)
 SELECT ?,d.id,d.library_id,f.id,v.id,d.revision,?,?,?,?,?,?,v.sha256,v.size_bytes,'planned',?,?,?,?,? FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN content_versions v ON v.id=f.current_content_version_id WHERE d.id=?`, operationID, root.ID, root.Revision, configurationRevision, relative, comparison(filepath.Join(root.Path, filepath.FromSlash(relative)), root.CaseSensitive), partial, principal.User.ID, kind, nullableID(reviewID), now(), now(), document.ID)
	if err != nil {
		return "", err
	}
	if _, err = transaction.ExecContext(ctx, "UPDATE documents SET approval_status='materializing',revision=revision+1 WHERE id=?", document.ID); err != nil {
		return "", err
	}
	if _, err = transaction.ExecContext(ctx, "UPDATE upload_items SET retain_until=NULL WHERE document_id=?", document.ID); err != nil {
		return "", err
	}
	jobID := domain.NewID()
	if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'materialize',?,?,'{}','queued',?,?)", jobID, document.LibraryID, document.FileID, operationID, "materialize:"+operationID, now(), now()); err != nil {
		return "", err
	}
	return operationID, workflowEvent(ctx, transaction, principal, document, metadata, "document.materialization_requested", map[string]any{"operation_id": operationID, "decision_kind": kind, "configuration_revision": configurationRevision})
}

// Each child directory is opened through a verified handle; no symlink component
// or mount outside the registered volume is accepted, including a swapped parent.
func managedDirectory(root Root, relative string, create bool) (*os.Root, string, error) {
	if !relativeSafe(relative) {
		return nil, "", scanFailure("STORAGE_PATH_INVALID")
	}
	container, err := os.OpenRoot(root.Path)
	if err != nil {
		return nil, "", scanFailure("ROOT_UNAVAILABLE")
	}
	file, err := container.Open(".")
	if err != nil {
		container.Close()
		return nil, "", err
	}
	identity, _, identityErr := physicalIdentity(file)
	file.Close()
	if identityErr != nil || identity != root.Identity {
		container.Close()
		return nil, "", scanFailure("ROOT_UNAVAILABLE")
	}
	components := strings.Split(relative, "/")
	for _, component := range components[:len(components)-1] {
		if create {
			if err = container.Mkdir(component, 0700); err != nil && !os.IsExist(err) {
				container.Close()
				return nil, "", scanFailure("STORAGE_UNAVAILABLE")
			}
			if err = syncManagedDirectory(container); err != nil {
				container.Close()
				return nil, "", err
			}
		}
		before, err := container.Lstat(component)
		if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
			container.Close()
			return nil, "", scanFailure("STORAGE_PATH_INVALID")
		}
		child, err := container.OpenRoot(component)
		if err != nil {
			container.Close()
			return nil, "", scanFailure("STORAGE_UNAVAILABLE")
		}
		directory, err := child.Open(".")
		if err != nil {
			child.Close()
			container.Close()
			return nil, "", err
		}
		after, statErr := directory.Stat()
		childIdentity, _, identityErr := physicalIdentity(directory)
		directory.Close()
		if statErr != nil || identityErr != nil || !os.SameFile(before, after) || volumeKey(childIdentity) != volumeKey(root.Identity) {
			child.Close()
			container.Close()
			return nil, "", scanFailure("STORAGE_PATH_INVALID")
		}
		container.Close()
		container = child
	}
	return container, components[len(components)-1], nil
}
func materializationFile(container *os.Root, name string) (*os.File, error) {
	before, err := container.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, scanFailure("STORAGE_PATH_INVALID")
	}
	file, err := container.Open(name)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		file.Close()
		return nil, scanFailure("STORAGE_PATH_INVALID")
	}
	return file, nil
}
func (service *Service) materializationState(ctx context.Context, job Job, state, identity string) error {
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if err := service.checkMaterializationLease(ctx, transaction, job); err != nil {
			return err
		}
		_, err := transaction.ExecContext(ctx, "UPDATE materializations SET state=?,published_identity=CASE WHEN ?='' THEN published_identity ELSE ? END,error_code='',updated_at=? WHERE id=?", state, identity, identity, now(), job.Version)
		return err
	})
}
func (service *Service) checkMaterializationLease(ctx context.Context, query storage.Querier, job Job) error {
	if err := service.jobLicense(ctx, query, job); err != nil {
		return err
	}
	var count int
	if err := query.QueryRowContext(ctx, "SELECT count(*) FROM jobs WHERE id=? AND status='running' AND fencing_token=?", job.ID, job.Fence).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return scanFailure("VERSION_SUPERSEDED")
	}
	return nil
}
func (service *Service) Materialize(ctx context.Context, job Job) error {
	if err := service.checkMaterializationLease(ctx, service.Database.Reader, job); err != nil {
		return err
	}
	operation, err := scanMaterialization(service.Database.Reader.QueryRowContext(ctx, "SELECT "+materializationColumns+" FROM materializations WHERE id=?", job.Version))
	if err != nil {
		return err
	}
	if operation.State == "cleaned" {
		return nil
	}
	root, err := service.root(ctx, operation.RootID)
	if err != nil {
		return err
	}
	if root.Source != "managed" || root.Status != "active" || root.Revision != operation.RootRevision {
		return scanFailure("ROOT_UNAVAILABLE")
	}
	container, filename, err := managedDirectory(root, operation.Relative, true)
	if err != nil {
		return err
	}
	defer container.Close()
	partialName := filepath.Base(operation.Partial)
	if operation.State == "committed" {
		return service.cleanupMaterialization(ctx, job, operation, container, filename, partialName)
	}
	if err = service.checkMaterializationLease(ctx, service.Database.Reader, job); err != nil {
		return err
	}
	partial, partialErr := materializationFile(container, partialName)
	if partialErr != nil && !os.IsNotExist(partialErr) {
		return scanFailure("STORAGE_UNAVAILABLE")
	}
	final, finalErr := materializationFile(container, filename)
	if finalErr == nil {
		defer final.Close()
		identity, _, err := physicalIdentity(final)
		if err != nil {
			if partial != nil {
				partial.Close()
			}
			return err
		}
		owned := operation.Identity != "" && operation.Identity == identity
		if partial != nil {
			partialInfo, _ := partial.Stat()
			finalInfo, _ := final.Stat()
			owned = owned || partialInfo != nil && finalInfo != nil && os.SameFile(partialInfo, finalInfo)
			partial.Close()
		}
		if !owned {
			return scanFailure("STORAGE_COLLISION")
		}
		digest, size, err := hashFile(ctx, final, operation.Size)
		if err != nil || digest != operation.Hash || size != operation.Size {
			return scanFailure("STORAGE_INTEGRITY_FAILED")
		}
		if err = service.materializationState(ctx, job, "published", identity); err != nil {
			return err
		}
	} else {
		if !os.IsNotExist(finalErr) {
			if partial != nil {
				partial.Close()
			}
			return scanFailure("STORAGE_UNAVAILABLE")
		}
		if operation.Identity != "" {
			if partial != nil {
				partial.Close()
			}
			return scanFailure("STORAGE_INTEGRITY_FAILED")
		}
		// A partial file is owned by its journal name, but only verified bytes can publish.
		verified := false
		if partial != nil {
			digest, size, hashErr := hashFile(ctx, partial, operation.Size)
			verified = hashErr == nil && digest == operation.Hash && size == operation.Size
			partial.Close()
			if !verified {
				if err = container.Remove(partialName); err != nil {
					return scanFailure("STORAGE_UNAVAILABLE")
				}
			}
		}
		if !verified {
			available, err := storage.AvailableBytes(root.Path)
			if err != nil || available < uint64(operation.Size+(64<<20)) {
				return scanFailure("STORAGE_SPACE")
			}
			if err = service.materializationState(ctx, job, "copying", ""); err != nil {
				return err
			}
			source, err := service.openUpload(ctx, operation.FileID)
			if err != nil {
				return err
			}
			target, err := container.OpenFile(partialName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				source.Close()
				return scanFailure("STORAGE_UNAVAILABLE")
			}
			digest := sha256.New()
			size, copyErr := copyDocument(ctx, io.MultiWriter(target, digest), io.LimitReader(source, operation.Size+1))
			syncErr := target.Sync()
			closeErr := target.Close()
			source.Close()
			if copyErr != nil {
				return copyErr
			}
			if syncErr != nil || closeErr != nil {
				return scanFailure("STORAGE_UNAVAILABLE")
			}
			if size != operation.Size || fmt.Sprintf("%x", digest.Sum(nil)) != operation.Hash {
				return scanFailure("STORAGE_INTEGRITY_FAILED")
			}
		}
		if err = service.materializationState(ctx, job, "verified", ""); err != nil {
			return err
		}
		if err = service.materializationCheckpoint("before_publish"); err != nil {
			return err
		}
		// Link is atomic and refuses replacement; unsupported destination filesystems fail closed.
		if err = container.Link(partialName, filename); err != nil {
			return scanFailure("STORAGE_PUBLICATION_FAILED")
		}
		if err = syncManagedDirectory(container); err != nil {
			return err
		}
		if err = service.materializationCheckpoint("after_publish"); err != nil {
			return err
		}
		final, err = materializationFile(container, filename)
		if err != nil {
			return err
		}
		identity, _, identityErr := physicalIdentity(final)
		final.Close()
		if identityErr != nil {
			return identityErr
		}
		if err = service.materializationState(ctx, job, "published", identity); err != nil {
			return err
		}
	}
	if err = service.materializationCheckpoint("before_commit"); err != nil {
		return err
	}
	// Reopen through the original absolute root immediately before commit. A renamed
	// parent cannot make a detached directory appear to be the recorded final path.
	published, err := openLinked(root, operation.Relative)
	if err != nil {
		return scanFailure("STORAGE_UNAVAILABLE")
	}
	defer published.Close()
	digest, size, err := hashFile(ctx, published, operation.Size)
	if err != nil || digest != operation.Hash || size != operation.Size {
		return scanFailure("STORAGE_INTEGRITY_FAILED")
	}
	identity, _, err := physicalIdentity(published)
	if err != nil {
		return err
	}
	if err = service.commitMaterialization(ctx, job, operation, root, identity); err != nil {
		return err
	}
	if err = service.materializationCheckpoint("after_commit"); err != nil {
		return err
	}
	operation.Identity = identity
	return service.cleanupMaterialization(ctx, job, operation, container, filename, partialName)
}
func copyDocument(ctx context.Context, target io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			written, err := target.Write(buffer[:count])
			total += int64(written)
			if err != nil {
				return total, err
			}
			if written != count {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
func (service *Service) materializationCheckpoint(point string) error {
	if service.materializationFault != nil {
		return service.materializationFault(point)
	}
	return nil
}
func (service *Service) commitMaterialization(ctx context.Context, job Job, operation Materialization, root Root, identity string) error {
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if err := service.checkMaterializationLease(ctx, transaction, job); err != nil {
			return err
		}
		if err := service.checkRootRevision(ctx, transaction, root); err != nil {
			return err
		}
		_, mode, err := settingsFor(ctx, transaction, operation.LibraryID)
		if err != nil {
			return err
		}
		if err = service.Identity.License.CheckFeatures(ctx, modeCapabilities(mode)...); err != nil {
			return err
		}
		principal := domain.Principal{User: domain.User{ID: operation.ApprovedBy}}
		permission := "documents.finalize"
		if operation.DecisionKind == "review" {
			permission = "documents.approve"
			if err := service.Identity.License.CheckFeatures(ctx, "review_workflow"); err != nil {
				return err
			}
		}
		if err := service.require(ctx, transaction, principal, operation.LibraryID, permission); err != nil {
			return err
		}
		var enabled bool
		if err := transaction.QueryRowContext(ctx, "SELECT disabled_at IS NULL FROM users WHERE id=?", operation.ApprovedBy).Scan(&enabled); err != nil {
			return err
		}
		if !enabled {
			return notFound()
		}
		if operation.ReviewID != "" {
			var selfReview bool
			if err := transaction.QueryRowContext(ctx, "SELECT d.created_by=? OR r.requested_by=? FROM review_requests r JOIN documents d ON d.id=r.document_id WHERE r.id=? AND r.status='pending'", operation.ApprovedBy, operation.ApprovedBy, operation.ReviewID).Scan(&selfReview); err != nil {
				return err
			}
			if selfReview {
				if err := service.require(ctx, transaction, principal, operation.LibraryID, "documents.approve_own"); err != nil {
					return err
				}
			}
		}
		var version string
		var revision int64
		var status, caseID string
		if err := transaction.QueryRowContext(ctx, "SELECT d.revision,d.approval_status,f.current_content_version_id,coalesce(d.case_id,'') FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=? AND d.deleted_at IS NULL", operation.DocumentID).Scan(&revision, &status, &version, &caseID); err != nil {
			return err
		}
		if revision != operation.Revision+1 || status != "materializing" || version != operation.VersionID {
			return conflict()
		}
		if caseID != "" {
			if err := service.Identity.License.CheckFeatures(ctx, "expedientes"); err != nil {
				return err
			}
		}
		locationID := domain.NewID()
		path := filepath.Join(root.Path, filepath.FromSlash(operation.Relative))
		if _, err := transaction.ExecContext(ctx, "INSERT INTO physical_file_locations VALUES(?,?,?,'managed',?,?,?,?,?,?,?,NULL)", locationID, operation.FileID, operation.LibraryID, root.ID, operation.Relative, path, operation.Comparison, root.Revision, now(), now()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET primary_location_id=?,os_identity_key=?,availability='available',integrity_status='verified' WHERE id=?", locationID, identity, operation.FileID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE documents SET approval_status='approved',approved_content_version_id=?,approved_by=?,approved_at=?,revision=revision+1 WHERE id=?", operation.VersionID, operation.ApprovedBy, now(), operation.DocumentID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE upload_items SET status='cleaned',retain_until=NULL WHERE document_id=?", operation.DocumentID); err != nil {
			return err
		}
		if operation.ReviewID != "" {
			if _, err := transaction.ExecContext(ctx, "UPDATE review_requests SET status='approved',decided_by=?,decided_at=? WHERE id=? AND status='pending'", operation.ApprovedBy, now(), operation.ReviewID); err != nil {
				return err
			}
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE materializations SET state='committed',published_identity=?,error_code='',updated_at=? WHERE id=?", identity, now(), operation.ID); err != nil {
			return err
		}
		return workflowEvent(ctx, transaction, principal, Document{ID: operation.DocumentID, LibraryID: operation.LibraryID, CaseID: caseID}, domain.RequestMetadata{}, "document.approved", map[string]any{"decision_kind": operation.DecisionKind, "operation_id": operation.ID, "content_version_id": operation.VersionID, "configuration_revision": operation.ConfigurationRevision})
	})
}
func (service *Service) cleanupMaterialization(ctx context.Context, job Job, operation Materialization, container *os.Root, filename, partialName string) error {
	if err := service.checkMaterializationLease(ctx, service.Database.Reader, job); err != nil {
		return err
	}
	if err := service.materializationCheckpoint("before_cleanup"); err != nil {
		return err
	}
	root, err := service.root(ctx, operation.RootID)
	if err != nil {
		return err
	}
	final, err := openLinked(root, operation.Relative)
	if err != nil {
		return scanFailure("STORAGE_UNAVAILABLE")
	}
	digest, size, err := hashFile(ctx, final, operation.Size)
	identity, _, identityErr := physicalIdentity(final)
	final.Close()
	if err != nil || identityErr != nil || digest != operation.Hash || size != operation.Size || identity != operation.Identity {
		return scanFailure("STORAGE_INTEGRITY_FAILED")
	}
	var locator string
	if err = service.Database.Reader.QueryRowContext(ctx, "SELECT private_temporary_locator FROM upload_items WHERE document_id=?", operation.DocumentID).Scan(&locator); err != nil {
		return err
	}
	if !relativeSafe(locator) || strings.ContainsAny(locator, "/\\") {
		return scanFailure("STORAGE_PATH_INVALID")
	}
	temporary, err := os.OpenRoot(service.Identity.Config.PrivateUploadDirectory())
	if err != nil {
		return privateUploadError()
	}
	defer temporary.Close()
	if err = temporary.Remove(locator); err != nil && !os.IsNotExist(err) {
		return privateUploadError()
	}
	if err = container.Remove(partialName); err != nil && !os.IsNotExist(err) {
		return scanFailure("STORAGE_UNAVAILABLE")
	}
	if err = syncManagedDirectory(container); err != nil {
		return err
	}
	return service.materializationState(ctx, job, "cleaned", identity)
}
