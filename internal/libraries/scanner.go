package libraries

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

type observation struct {
	Identity, Relative, Hash string
	Strong                   bool
	Size                     int64
	Modified                 string
}

func scanFailure(code string) error {
	return domain.Failure(code, "La indexación requiere atención. Consulta el diagnóstico de la raíz o tarea.", 409)
}
func failureCode(err error) string {
	var failure *domain.Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	if errors.Is(err, context.Canceled) {
		return "INTERRUPTED"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "PROCESS_TIMEOUT"
	}
	return "SOURCE_UNAVAILABLE"
}
func (service *Service) root(ctx context.Context, rootID string) (Root, error) {
	roots, err := rootsQuery(ctx, service.Database.Reader, "")
	if err != nil {
		return Root{}, err
	}
	for _, root := range roots {
		if root.ID == rootID {
			return root, nil
		}
	}
	return Root{}, notFound()
}
func activeRoot(root Root) bool { return root.Status == "active" || root.Status == "inaccessible" }
func (service *Service) Scan(ctx context.Context, rootID string, maximumBytes int64, registerDirectory func(string) error) error {
	if err := licensing.Check(licensing.WriteDocuments); err != nil {
		return err
	}
	root, err := service.root(ctx, rootID)
	if err != nil {
		return err
	}
	if !activeRoot(root) {
		return scanFailure("ROOT_DISABLED")
	}
	directory, err := inspectDirectory(root.Path)
	if err != nil || directory.Identity != root.Identity {
		return service.scanError(ctx, root, scanFailure("ROOT_UNAVAILABLE"))
	}
	container, err := os.OpenRoot(root.Path)
	if err != nil {
		return service.scanError(ctx, root, err)
	}
	defer container.Close()
	scanID := domain.NewID()
	pending := []string{"."}
	err = service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if _, err := transaction.ExecContext(ctx, "UPDATE root_scans SET status='superseded' WHERE root_id=? AND status='running' AND configuration_revision<>?", root.ID, root.Revision); err != nil {
			return err
		}
		var previousID, checkpoint string
		lookup := transaction.QueryRowContext(ctx, "SELECT id,checkpoint_json FROM root_scans WHERE root_id=? AND status='running' AND configuration_revision=?", root.ID, root.Revision).Scan(&previousID, &checkpoint)
		if lookup == nil {
			scanID = previousID
			var decodeError error
			pending, decodeError = decodeStrings(checkpoint)
			return decodeError
		}
		if lookup != sql.ErrNoRows {
			return lookup
		}
		_, err := transaction.ExecContext(ctx, "INSERT INTO root_scans(id,root_id,configuration_revision,status,checkpoint_json,started_at) VALUES(?,?,?,'running',?,?)", scanID, root.ID, root.Revision, encode(pending), now())
		return err
	})
	if err != nil {
		return err
	}
	directoriesSeen := 0
	skippedLinks := false
	var scanIssue error
	for len(pending) > 0 {
		if err = ctx.Err(); err != nil {
			return err
		}
		relativeDirectory := pending[0]
		info, statError := container.Lstat(relativeDirectory)
		if statError != nil {
			return service.scanError(ctx, root, statError)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return service.scanError(ctx, root, scanFailure("DIRECTORY_CHANGED"))
		}
		if registerDirectory != nil {
			_ = registerDirectory(filepath.Join(root.Path, relativeDirectory))
		} // Failure switches the runtime to polling.
		folder, openError := container.Open(relativeDirectory)
		if openError != nil {
			return service.scanError(ctx, root, openError)
		}
		openedInfo, openError := folder.Stat()
		if openError != nil || !os.SameFile(info, openedInfo) {
			folder.Close()
			return service.scanError(ctx, root, scanFailure("DIRECTORY_CHANGED"))
		}
		openedIdentity, _, identityError := physicalIdentity(folder)
		if identityError != nil || volumeKey(openedIdentity) != volumeKey(root.Identity) {
			folder.Close()
			return service.scanError(ctx, root, scanFailure("ROOT_UNAVAILABLE"))
		}
		nextDirectories := []string{}
		for {
			entries, readError := folder.ReadDir(128)
			if readError != nil && readError != io.EOF {
				folder.Close()
				return service.scanError(ctx, root, readError)
			}
			for _, entry := range entries {
				relative := filepath.ToSlash(filepath.Join(relativeDirectory, entry.Name()))
				if entry.Type()&os.ModeSymlink != 0 {
					skippedLinks = true
					continue
				}
				if entry.IsDir() {
					nextDirectories = append(nextDirectories, relative)
					continue
				}
				if !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
					continue
				}
				file, openError := openLinked(root, relative)
				if openError != nil {
					if scanIssue == nil {
						scanIssue = openError
					}
					continue
				}
				observed, observeError := observe(ctx, file, relative, maximumBytes)
				file.Close()
				if observeError != nil {
					if ctx.Err() != nil {
						folder.Close()
						return ctx.Err()
					}
					if scanIssue == nil {
						scanIssue = observeError
					}
					continue
				}
				if volumeKey(observed.Identity) != volumeKey(root.Identity) {
					folder.Close()
					return service.scanError(ctx, root, scanFailure("ROOT_UNAVAILABLE"))
				}
				if err = service.ingest(ctx, root, scanID, observed); err != nil {
					folder.Close()
					return service.scanError(ctx, root, err)
				}
			}
			if readError == io.EOF {
				break
			}
		}
		folder.Close()
		directoriesSeen++
		pending = append(pending[1:], nextDirectories...)
		if len(pending)+directoriesSeen > 100000 {
			return service.scanError(ctx, root, scanFailure("DIRECTORY_LIMIT"))
		}
		if _, err = service.Database.Writer.ExecContext(ctx, "UPDATE root_scans SET checkpoint_json=?,directories_seen=directories_seen+1 WHERE id=? AND status='running'", encode(pending), scanID); err != nil {
			return err
		}
	}
	if scanIssue != nil {
		return service.scanError(ctx, root, scanIssue)
	}
	// Never infer deletions unless the same mounted directory survived a complete scan.
	verified, err := inspectDirectory(root.Path)
	if err != nil || verified.Identity != root.Identity {
		return service.scanError(ctx, root, scanFailure("ROOT_UNAVAILABLE"))
	}
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if err := checkRootRevision(ctx, transaction, root); err != nil {
			return err
		}
		// Retire only locations proven absent; retain the historical row and its OCR.
		if _, err := transaction.ExecContext(ctx, "UPDATE physical_file_locations SET retired_at=? WHERE root_id=? AND retired_at IS NULL AND NOT EXISTS(SELECT 1 FROM scan_file_observations WHERE scan_id=? AND location_id=physical_file_locations.id)", now(), root.ID, scanID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET primary_location_id=coalesce((SELECT location.id FROM physical_file_locations location JOIN storage_roots owner ON owner.id=location.root_id WHERE location.physical_file_id=physical_files.id AND location.retired_at IS NULL AND owner.status<>'superseded' ORDER BY CASE owner.status WHEN 'active' THEN 0 ELSE 1 END,location.id LIMIT 1),primary_location_id) WHERE library_id=?", root.LibraryID); err != nil {
			return err
		}
		missingRows, err := transaction.QueryContext(ctx, "SELECT d.id,coalesce(f.current_content_version_id,'') FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE f.library_id=? AND f.availability<>'missing' AND NOT EXISTS(SELECT 1 FROM physical_file_locations WHERE physical_file_id=f.id AND retired_at IS NULL)", root.LibraryID)
		if err != nil {
			return err
		}
		for missingRows.Next() {
			var documentID, versionID string
			if err = missingRows.Scan(&documentID, &versionID); err != nil {
				missingRows.Close()
				return err
			}
			if err = appendDocumentObservation(ctx, transaction, domain.NewID(), "document.linked_missing", root.LibraryID, documentID, versionID); err != nil {
				missingRows.Close()
				return err
			}
		}
		err = missingRows.Err()
		missingRows.Close()
		if err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET availability='missing' WHERE library_id=? AND NOT EXISTS(SELECT 1 FROM physical_file_locations WHERE physical_file_id=physical_files.id AND retired_at IS NULL)", root.LibraryID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE root_scans SET status='complete',completed_at=?,can_confirm_absence=1 WHERE id=?", now(), scanID); err != nil {
			return err
		}
		diagnostic := ""
		if skippedLinks {
			diagnostic = "INTERNAL_LINK_SKIPPED"
		}
		_, err = transaction.ExecContext(ctx, "UPDATE storage_roots SET status='active',last_verified_at=?,last_scan_at=?,last_error_code=CASE WHEN last_error_code IN ('WATCH_LIMIT','WATCH_EVENTS_LOST') THEN last_error_code ELSE ? END WHERE id=?", now(), now(), diagnostic, root.ID)
		return err
	})
}
func checkRootRevision(ctx context.Context, transaction *sql.Tx, root Root) error {
	if err := licensing.Check(licensing.WriteDocuments); err != nil {
		return err
	}
	var count int
	if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM storage_roots WHERE id=? AND configuration_revision=? AND status IN ('active','inaccessible')", root.ID, root.Revision).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return scanFailure("ROOT_PLAN_STALE")
	}
	return nil
}
func observe(ctx context.Context, file *os.File, relative string, maximumBytes int64) (observation, error) {
	var observed observation
	info, err := file.Stat()
	if err != nil {
		return observed, err
	}
	if info.Size() > maximumBytes {
		return observed, scanFailure("FILE_SIZE_LIMIT")
	}
	if time.Since(info.ModTime()) < 2*time.Second {
		return observed, scanFailure("FILE_UNSTABLE")
	}
	key, strong, err := physicalIdentity(file)
	if err != nil {
		return observed, err
	}
	header := make([]byte, 5)
	if _, err = io.ReadFull(file, header); err != nil || string(header) != "%PDF-" {
		return observed, scanFailure("INVALID_PDF")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return observed, err
	}
	hash := sha256.New()
	buffer := make([]byte, 64*1024)
	var count int64
	for {
		if err = ctx.Err(); err != nil {
			return observed, err
		}
		read, readError := file.Read(buffer)
		if read > 0 {
			count += int64(read)
			if count > maximumBytes {
				return observed, scanFailure("FILE_SIZE_LIMIT")
			}
			hash.Write(buffer[:read])
		}
		if readError == io.EOF {
			break
		}
		if readError != nil {
			return observed, readError
		}
	}
	after, err := file.Stat()
	if err != nil || after.Size() != info.Size() || after.ModTime() != info.ModTime() || count != info.Size() {
		return observed, scanFailure("FILE_UNSTABLE")
	}
	return observation{Identity: key, Strong: strong, Relative: relative, Hash: fmt.Sprintf("%x", hash.Sum(nil)), Size: count, Modified: domain.Timestamp(info.ModTime())}, nil
}
func (service *Service) scanError(ctx context.Context, root Root, cause error) error {
	code := failureCode(cause)
	err := service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if _, err := transaction.ExecContext(ctx, "UPDATE root_scans SET status='failed',can_confirm_absence=0,last_error_code=? WHERE root_id=? AND status='running'", code, root.ID); err != nil {
			return err
		}
		status := "active"
		if code == "ROOT_UNAVAILABLE" || code == "SOURCE_UNAVAILABLE" || code == "DIRECTORY_CHANGED" {
			status = "inaccessible"
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE storage_roots SET status=?,last_error_code=?,last_scan_at=? WHERE id=? AND configuration_revision=? AND status IN ('active','inaccessible')", status, code, now(), root.ID, root.Revision); err != nil {
			return err
		}
		if root.LastError == code {
			return nil
		}
		return record(ctx, transaction, domain.Principal{}, domain.RequestMetadata{RequestID: domain.NewID()}, "storage.scan_failed", root.LibraryID, "", map[string]any{"root_id": root.ID, "error_code": code})
	})
	if err != nil {
		return err
	}
	return cause
}
func (service *Service) ingest(ctx context.Context, root Root, scanID string, observed observation) error {
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if err := checkRootRevision(ctx, transaction, root); err != nil {
			return err
		}
		var fileID, ownerLibrary, versionID, oldHash, availability string
		err := transaction.QueryRowContext(ctx, "SELECT f.id,f.library_id,coalesce(f.current_content_version_id,''),coalesce(v.sha256,''),f.availability FROM physical_files f LEFT JOIN content_versions v ON v.id=f.current_content_version_id WHERE f.os_identity_key=?", observed.Identity).Scan(&fileID, &ownerLibrary, &versionID, &oldHash, &availability)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if ownerLibrary != "" && ownerLibrary != root.LibraryID {
			return scanFailure("FILE_AUTHORIZATION_CONFLICT")
		}
		if !observed.Strong && availability == "missing" {
			return scanFailure("FILE_IDENTITY_AMBIGUOUS")
		}
		newFile := fileID == ""
		documentID := ""
		if newFile {
			fileID = domain.NewID()
			documentID = domain.NewID()
			if _, err = transaction.ExecContext(ctx, "INSERT INTO physical_files(id,library_id,storage_source,os_identity_key,availability,integrity_status,created_at) VALUES(?,?,'linked',?,'available','verified',?)", fileID, root.LibraryID, observed.Identity, now()); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES(?,?,?,?,?,?)", documentID, root.LibraryID, fileID, filepath.Base(observed.Relative), searchKey(filepath.Base(observed.Relative)), now()); err != nil {
				return err
			}
		} else {
			if err = transaction.QueryRowContext(ctx, "SELECT id FROM documents WHERE physical_file_id=?", fileID).Scan(&documentID); err != nil {
				return err
			}
		}
		path := filepath.Join(root.Path, filepath.FromSlash(observed.Relative))
		key := comparison(path, root.CaseSensitive)
		var locationID, locationFile string
		err = transaction.QueryRowContext(ctx, "SELECT id,physical_file_id FROM physical_file_locations WHERE comparison_key=? AND retired_at IS NULL", key).Scan(&locationID, &locationFile)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if locationID != "" && locationFile != fileID {
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_file_locations SET retired_at=? WHERE id=?", now(), locationID); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_files SET availability='missing' WHERE id=? AND primary_location_id=?", locationFile, locationID); err != nil {
				return err
			}
			locationID = ""
		}
		if locationID == "" {
			locationID = domain.NewID()
			if _, err = transaction.ExecContext(ctx, "INSERT INTO physical_file_locations VALUES(?,?,?,'linked',?,?,?,?,?,?,?,NULL)", locationID, fileID, root.LibraryID, root.ID, observed.Relative, path, key, root.Revision, now(), now()); err != nil {
				return err
			}
		} else {
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_file_locations SET last_seen_at=? WHERE id=?", now(), locationID); err != nil {
				return err
			}
		}
		observationResult, err := transaction.ExecContext(ctx, "INSERT INTO scan_file_observations VALUES(?,?,?) ON CONFLICT DO NOTHING", scanID, locationID, now())
		if err != nil {
			return err
		}
		observationsAdded, err := observationResult.RowsAffected()
		if err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE root_scans SET files_seen=files_seen+? WHERE id=?", observationsAdded, scanID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE physical_files SET availability='available',primary_location_id=? WHERE id=?", locationID, fileID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE documents SET original_filename=?,filename_search_key=? WHERE id=?", filepath.Base(observed.Relative), searchKey(filepath.Base(observed.Relative)), documentID); err != nil {
			return err
		}
		if oldHash == observed.Hash {
			// A root/configuration pause can cancel extraction without changing bytes.
			// Resume that work, but leave terminal extraction failures for explicit retry.
			var resume int
			if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE f.id=? AND f.extraction_freshness<>'current' AND d.deleted_at IS NULL AND (SELECT status FROM jobs WHERE physical_file_id=f.id AND target_version=f.current_content_version_id AND job_type='extract' ORDER BY created_at DESC,id DESC LIMIT 1)='cancelled'", fileID).Scan(&resume); err != nil {
				return err
			}
			if resume > 0 {
				jobID := domain.NewID()
				if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','queued',?,?)", jobID, root.LibraryID, fileID, versionID, jobID, now(), now()); err != nil {
					return err
				}
			}
			if availability == "missing" {
				return appendDocumentObservation(ctx, transaction, domain.NewID(), "document.linked_reappeared", root.LibraryID, documentID, versionID)
			}
			return nil
		}
		versionID = domain.NewID()
		var generation int
		if err = transaction.QueryRowContext(ctx, "SELECT coalesce(max(generation),0)+1 FROM content_versions WHERE physical_file_id=?", fileID).Scan(&generation); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,sha256,observed_change_key,observed_at,change_origin) VALUES(?,?,?,?,?,?,?,?,'scan')", versionID, fileID, generation, observed.Size, observed.Modified, observed.Hash, versionID, now()); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE physical_files SET current_content_version_id=?,integrity_status='verified',extraction_freshness=CASE WHEN indexed_extraction_id IS NULL THEN 'none' ELSE 'stale' END WHERE id=?", versionID, fileID); err != nil {
			return err
		}
		jobID := domain.NewID()
		if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','queued',?,?)", jobID, root.LibraryID, fileID, versionID, "extract:"+versionID, now(), now()); err != nil {
			return err
		}
		eventID := domain.NewID()
		eventType := "document.discovered"
		if !newFile {
			eventType = "document.linked_changed"
		}
		if err = appendDocumentObservation(ctx, transaction, eventID, eventType, root.LibraryID, documentID, versionID); err != nil {
			return err
		}
		if !newFile {
			_, err = transaction.ExecContext(ctx, "INSERT INTO document_notifications VALUES(?,?,?,?,'linked_changed',?,?)", domain.NewID(), documentID, versionID, eventID, versionID, now())
			if err != nil {
				return err
			}
		}
		return nil
	})
}
