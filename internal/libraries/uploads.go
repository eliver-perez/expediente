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
	"time"
	"unicode/utf8"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type UploadItem struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	Filename   string `json:"original_filename"`
	Status     string `json:"status"`
	Error      string `json:"error_code"`
	Size       int64  `json:"size_bytes"`
}
type UploadBatch struct {
	ID        string       `json:"id"`
	LibraryID string       `json:"library_id"`
	CreatedBy string       `json:"created_by"`
	Created   string       `json:"created_at"`
	Items     []UploadItem `json:"items"`
}

func validClientKey(key string) bool {
	return len(key) >= 16 && len(key) <= 100 && !strings.ContainsAny(key, "\x00\r\n")
}
func (service *Service) requireManaged(ctx context.Context, query storage.Querier, libraryID string) error {
	_, mode, err := settingsFor(ctx, query, libraryID)
	if err != nil {
		return err
	}
	if mode == "linked" {
		return invalid("Convierte la biblioteca a híbrida para admitir cargas.")
	}
	return service.Identity.License.CheckFeatures(ctx, modeCapabilities(mode)...)
}
func (service *Service) CreateBatch(ctx context.Context, principal domain.Principal, libraryID, clientID string, metadata domain.RequestMetadata) (string, error) {
	if !validClientKey(clientID) {
		return "", invalid("Identificador de lote inválido.")
	}
	identifier := domain.NewID()
	err := service.write(ctx, principal, libraryID, "documents.upload", func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.requireManaged(ctx, transaction, libraryID); err != nil {
			return err
		}
		var existing string
		err := transaction.QueryRowContext(ctx, "SELECT id FROM upload_batches WHERE library_id=? AND created_by=? AND client_batch_id=?", libraryID, current.User.ID, clientID).Scan(&existing)
		if err == nil {
			identifier = existing
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO upload_batches VALUES(?,?,?,?,?)", identifier, libraryID, current.User.ID, clientID, now()); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "upload.batch_created", libraryID, "", map[string]any{"batch_id": identifier})
	})
	return identifier, err
}
func (service *Service) Batch(ctx context.Context, principal domain.Principal, identifier string) (UploadBatch, error) {
	batch := UploadBatch{Items: []UploadItem{}}
	err := service.Database.Reader.QueryRowContext(ctx, "SELECT id,library_id,created_by,created_at FROM upload_batches WHERE id=?", identifier).Scan(&batch.ID, &batch.LibraryID, &batch.CreatedBy, &batch.Created)
	if err == sql.ErrNoRows {
		return batch, notFound()
	}
	if err != nil {
		return batch, err
	}
	if err = service.Read(ctx, principal, batch.LibraryID, "documents.read"); err != nil {
		return UploadBatch{}, err
	}
	if batch.CreatedBy != principal.User.ID && service.require(ctx, service.Database.Reader, principal, batch.LibraryID, "documents.review") != nil {
		return UploadBatch{}, notFound()
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,coalesce(document_id,''),original_filename,status,error_code,size_bytes FROM upload_items WHERE batch_id=? ORDER BY created_at,id", identifier)
	if err != nil {
		return batch, err
	}
	defer rows.Close()
	for rows.Next() {
		var item UploadItem
		if err = rows.Scan(&item.ID, &item.DocumentID, &item.Filename, &item.Status, &item.Error, &item.Size); err != nil {
			return batch, err
		}
		batch.Items = append(batch.Items, item)
	}
	return batch, rows.Err()
}
func (service *Service) Batches(ctx context.Context, principal domain.Principal, libraryID, cursor string) (Page[UploadBatch], error) {
	result := Page[UploadBatch]{Items: []UploadBatch{}}
	if err := service.Read(ctx, principal, libraryID, "documents.upload"); err != nil {
		return result, err
	}
	scope := principal.User.ID + ":uploads:" + libraryID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,library_id,created_by,created_at FROM upload_batches WHERE library_id=? AND created_by=? AND (?='' OR created_at<? OR (created_at=? AND id<?)) ORDER BY created_at DESC,id DESC LIMIT 51", libraryID, principal.User.ID, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item UploadBatch
		if err = rows.Scan(&item.ID, &item.LibraryID, &item.CreatedBy, &item.Created); err != nil {
			return result, err
		}
		item.Items = []UploadItem{}
		result.Items = append(result.Items, item)
	}
	if len(result.Items) > 50 {
		result.Items = result.Items[:50]
		last := result.Items[49]
		result.Next = nextListCursor(scope, last.Created, last.ID)
	}
	return result, rows.Err()
}
func (service *Service) prepareUploads(ctx context.Context) error {
	directory := service.Identity.Config.PrivateUploadDirectory()
	roots, err := rootsQuery(ctx, service.Database.Reader, "")
	if err != nil {
		return err
	}
	for _, root := range roots {
		if root.Status != "superseded" && (within(root.Path, directory, root.CaseSensitive) || within(directory, root.Path, root.CaseSensitive)) {
			return invalid("El temporal se superpone con una raíz documental. Revisa la configuración local.")
		}
	}
	if err := storage.PrepareLocalPrivateDirectory(directory); err != nil {
		return domain.Failure("UPLOAD_STORAGE_UNAVAILABLE", "El almacenamiento temporal privado no está disponible.", 503)
	}
	return nil
}
func privateUploadError() error {
	return domain.Failure("UPLOAD_STORAGE_UNAVAILABLE", "No se pudo guardar la carga en almacenamiento privado.", 503)
}
func (service *Service) openUpload(ctx context.Context, fileID string) (*os.File, error) {
	var locator string
	err := service.Database.Reader.QueryRowContext(ctx, "SELECT u.private_temporary_locator FROM upload_items u JOIN documents d ON d.id=u.document_id WHERE d.physical_file_id=? AND u.status IN ('staged','retained') AND d.deleted_at IS NULL", fileID).Scan(&locator)
	if err != nil {
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	if !relativeSafe(locator) || strings.ContainsAny(locator, "/\\") {
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	root, err := os.OpenRoot(service.Identity.Config.PrivateUploadDirectory())
	if err != nil {
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	defer root.Close()
	info, err := root.Lstat(locator)
	if err != nil || !info.Mode().IsRegular() {
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	file, err := root.Open(locator)
	if err != nil {
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(info, after) {
		file.Close()
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	return file, nil
}

// The receiving row is a recovery journal, not a document. Publication creates
// the physical file, document, content version and extraction job atomically.
func (service *Service) Upload(ctx context.Context, principal domain.Principal, batchID, clientID, filename string, reader io.Reader, finish func() error, metadata domain.RequestMetadata) (UploadItem, error) {
	result := UploadItem{}
	if !validClientKey(clientID) || utf8.RuneCountInString(filename) > 250 || strings.ContainsAny(filename, "/\\\x00\r\n") || !strings.EqualFold(filepath.Ext(filename), ".pdf") {
		return result, invalid("Carga un archivo PDF con nombre válido.")
	}
	batch, err := service.Batch(ctx, principal, batchID)
	if err != nil {
		return result, err
	}
	if batch.CreatedBy != principal.User.ID {
		return result, notFound()
	}
	select {
	case service.uploadSlots <- struct{}{}:
		defer func() { <-service.uploadSlots }()
	default:
		return result, domain.Failure("UPLOAD_BUSY", "Hay dos cargas en curso. Vuelve a intentarlo en unos segundos.", 429)
	}
	if err = service.prepareUploads(ctx); err != nil {
		return result, err
	}
	maximum := int64(service.Identity.Config.Indexing.MaximumFileMB) << 20
	available, err := storage.AvailableBytes(service.Identity.Config.PrivateUploadDirectory())
	if err != nil {
		return result, privateUploadError()
	}
	if available < uint64(maximum+(64<<20)) {
		return result, domain.Failure("UPLOAD_SPACE", "No hay espacio suficiente para recibir la carga.", 507)
	}
	identifier := domain.NewID()
	locator := identifier + ".pdf"
	replay := false
	err = service.write(ctx, principal, batch.LibraryID, "documents.upload", func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.requireManaged(ctx, transaction, batch.LibraryID); err != nil {
			return err
		}
		var existingName, existingStatus string
		err := transaction.QueryRowContext(ctx, "SELECT id,original_filename,status,coalesce(document_id,''),size_bytes FROM upload_items WHERE batch_id=? AND client_file_id=?", batchID, clientID).Scan(&result.ID, &existingName, &existingStatus, &result.DocumentID, &result.Size)
		if err == nil {
			if existingName != filename {
				return domain.Failure("IDEMPOTENCY_CONFLICT", "La clave ya pertenece a otro archivo.", 409)
			}
			if existingStatus == "receiving" {
				return domain.Failure("UPLOAD_IN_PROGRESS", "Esta carga está en curso.", 409)
			}
			if existingStatus == "staged" || existingStatus == "retained" {
				result.Filename = filename
				result.Status = existingStatus
				replay = true
				return nil
			}
			// Failed requests keep evidence; retry uses a new receiving journal key.
			return domain.Failure("UPLOAD_RETRY_KEY", "Reintenta el archivo con una nueva clave de carga.", 409)
		}
		if err != sql.ErrNoRows {
			return err
		}
		var count int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM upload_items WHERE batch_id=?", batchID).Scan(&count); err != nil {
			return err
		}
		if count >= 100 {
			return invalid("El lote admite hasta 100 archivos. Crea otro lote.")
		}
		_, err = transaction.ExecContext(ctx, "INSERT INTO upload_items(id,batch_id,library_id,client_file_id,original_filename,private_temporary_locator,status,created_at) VALUES(?,?,?,?,?,?,'receiving',?)", identifier, batchID, batch.LibraryID, clientID, filename, locator, now())
		return err
	})
	if err != nil {
		return result, err
	}
	if replay {
		hash := sha256.New()
		size, err := io.Copy(hash, io.LimitReader(reader, maximum+1))
		if err != nil {
			return result, invalid("Carga incompleta.")
		}
		if finish != nil {
			if err = finish(); err != nil {
				return result, err
			}
		}
		var expected string
		if err = service.Database.Reader.QueryRowContext(ctx, "SELECT sha256 FROM upload_items WHERE id=?", result.ID).Scan(&expected); err != nil {
			return result, err
		}
		if size != result.Size || fmt.Sprintf("%x", hash.Sum(nil)) != expected {
			return result, domain.Failure("IDEMPOTENCY_CONFLICT", "La clave ya pertenece a otro contenido.", 409)
		}
		return result, nil
	}
	committed := false
	// Cleanup only our names; a crash is completed by recoverUploads at startup.
	root, err := os.OpenRoot(service.Identity.Config.PrivateUploadDirectory())
	if err != nil {
		service.failUpload(identifier, "UPLOAD_STORAGE_UNAVAILABLE")
		return result, privateUploadError()
	}
	defer root.Close()
	defer func() {
		if !committed {
			_ = root.Remove(locator + ".part")
			_ = root.Remove(locator)
			service.failUpload(identifier, "UPLOAD_INCOMPLETE")
		}
	}()
	file, err := root.OpenFile(locator+".part", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return result, privateUploadError()
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(reader, maximum+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		return result, invalid("La transferencia no se completó.")
	}
	if size > maximum {
		return result, domain.Failure("FILE_SIZE_LIMIT", "El PDF supera el tamaño máximo configurado.", 413)
	}
	if syncErr != nil || closeErr != nil {
		return result, privateUploadError()
	}
	if finish != nil {
		if err = finish(); err != nil {
			return result, err
		}
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	probe, err := root.Open(locator + ".part")
	if err != nil {
		return result, privateUploadError()
	}
	header := make([]byte, 5)
	_, err = io.ReadFull(probe, header)
	probe.Close()
	if err != nil || string(header) != "%PDF-" {
		return result, invalid("El contenido no es un PDF válido.")
	}
	if _, err = service.Identity.Config.Indexing.ValidatePDF(ctx, filepath.Join(service.Identity.Config.PrivateUploadDirectory(), locator+".part")); err != nil {
		return result, err
	}
	if err = root.Rename(locator+".part", locator); err != nil {
		return result, privateUploadError()
	}
	physical, err := root.Open(locator)
	if err != nil {
		return result, privateUploadError()
	}
	identity, _, identityErr := physicalIdentity(physical)
	info, statErr := physical.Stat()
	physical.Close()
	if identityErr != nil || statErr != nil {
		return result, privateUploadError()
	}
	documentID, fileID, versionID, jobID := domain.NewID(), domain.NewID(), domain.NewID(), domain.NewID()
	digest := fmt.Sprintf("%x", hash.Sum(nil))
	err = service.write(ctx, principal, batch.LibraryID, "documents.upload", func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.requireManaged(ctx, transaction, batch.LibraryID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO physical_files(id,library_id,storage_source,os_identity_key,availability,integrity_status,current_content_version_id,created_at) VALUES(?,?,'managed',?,'staged','verified',?,?)", fileID, batch.LibraryID, identity, versionID, now()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,sha256,observed_change_key,observed_at,change_origin) VALUES(?,?,1,?,?,?,?,?,'upload')", versionID, fileID, size, domain.Timestamp(info.ModTime()), digest, identifier, now()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_by,created_at) VALUES(?,?,?,?,?,?,?)", documentID, batch.LibraryID, fileID, filename, searchKey(filename), current.User.ID, now()); err != nil {
			return err
		}
		result, err := transaction.ExecContext(ctx, "UPDATE upload_items SET document_id=?,status='staged',size_bytes=?,sha256=? WHERE id=? AND status='receiving'", documentID, size, digest, identifier)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return conflict()
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','queued',?,?)", jobID, batch.LibraryID, fileID, versionID, "upload:"+identifier, now(), now()); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "document.uploaded", batch.LibraryID, documentID, map[string]any{"batch_id": batchID, "size_bytes": size, "sha256": digest})
	})
	if err != nil {
		return result, err
	}
	committed = true
	return UploadItem{ID: identifier, DocumentID: documentID, Filename: filename, Status: "staged", Size: size}, nil
}
func (service *Service) failUpload(identifier, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := service.Database.Writer.ExecContext(ctx, "UPDATE upload_items SET status='failed',error_code=? WHERE id=? AND status='receiving'", code, identifier)
	return err
}
func (service *Service) recoverUploads(ctx context.Context) error {
	if err := service.prepareUploads(ctx); err != nil {
		return err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,private_temporary_locator FROM upload_items WHERE status='receiving'")
	if err != nil {
		return err
	}
	type pending struct{ ID, Locator string }
	items := []pending{}
	for rows.Next() {
		var item pending
		if err = rows.Scan(&item.ID, &item.Locator); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(service.Identity.Config.PrivateUploadDirectory())
	if err != nil {
		return privateUploadError()
	}
	defer root.Close()
	for _, item := range items {
		if !relativeSafe(item.Locator) || strings.ContainsAny(item.Locator, "/\\") {
			return privateUploadError()
		}
		for _, name := range []string{item.Locator, item.Locator + ".part"} {
			if err = root.Remove(name); err != nil && !os.IsNotExist(err) {
				return privateUploadError()
			}
		}
		if err := service.failUpload(item.ID, "PROCESS_RESTARTED"); err != nil {
			return err
		}
	}
	return nil
}
