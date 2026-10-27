package backups

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

func (s *Service) Inspect(ctx context.Context, source io.Reader, password, actor string) (Operation, error) {
	id := domain.NewID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	op := Operation{ID: id, Kind: "restore", State: "inspecting", Stage: "Copiando paquete privado", Actor: actor, CreatedAt: now, UpdatedAt: now}
	incoming := filepath.Join(s.directory, id+".incoming")
	file, err := os.OpenFile(incoming, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Operation{}, err
	}
	size, err := io.Copy(file, io.LimitReader(&contextReader{ctx: ctx, source: source}, maxPackage+1))
	if syncErr := file.Sync(); err == nil {
		err = syncErr
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil || size > maxPackage {
		os.Remove(incoming)
		return Operation{}, fmt.Errorf("no se pudo recibir el paquete o excede el límite")
	}
	plain := filepath.Join(s.directory, id+".plain")
	packagePath, err := openPackage(incoming, plain, password)
	if err != nil {
		os.Remove(incoming)
		return Operation{}, err
	}
	archive, err := inspectZip(packagePath)
	if err != nil {
		os.Remove(incoming)
		os.Remove(plain)
		return Operation{}, err
	}
	defer archive.Close()
	if err = validateInventory(archive); err != nil {
		os.Remove(incoming)
		os.Remove(plain)
		return Operation{}, err
	}
	if err = s.validateLogical(ctx, archive); err != nil {
		os.Remove(incoming)
		os.Remove(plain)
		return Operation{}, fmt.Errorf("logical backup validation failed: %w", err)
	}
	op.State = "inspected"
	op.Stage = "Verificado; pendiente de confirmación"
	op.Manifest = &archive.Manifest
	op.Bytes = size
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err = s.writeOperation(op); err != nil {
		os.Remove(incoming)
		os.Remove(plain)
		return Operation{}, err
	}
	return op, nil
}

// Inspect in a disposable SQLite database to catch malformed rows, constraints,
// broken references and schema drift before a restore is scheduled. This
// scratch database has a blank license state and never enters the archive.
func (s *Service) validateLogical(ctx context.Context, pkg *Package) error {
	idx, err := collectIndex(pkg)
	if err != nil {
		return err
	}
	if len(pkg.Manifest.Roots) != len(idx.roots) {
		return fmt.Errorf("root inventory mismatch")
	}
	for _, entry := range pkg.Manifest.Roots {
		root := idx.roots[entry.ID]
		if root == nil || root.LibraryID != entry.LibraryID || root.Source != entry.Source || root.OldPath != entry.Path || root.Status != entry.Status {
			return fmt.Errorf("root inventory mismatch")
		}
	}
	directory, err := os.MkdirTemp(s.directory, ".inspect-db-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	db, err := storage.Open(ctx, directory)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err = db.Writer.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	transaction, err := db.Writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	for i := len(tables) - 1; i >= 0; i-- {
		if tables[i] == "audit_events" {
			continue
		}
		if _, err = transaction.ExecContext(ctx, "DELETE FROM \""+tables[i]+"\""); err != nil {
			return err
		}
	}
	for _, table := range tables {
		columns, e := tableColumns(ctx, transaction, table)
		if e != nil {
			return e
		}
		if e = importTable(ctx, transaction, pkg, pkg.Files["tables/"+table+".jsonl"], table, columns, &idx); e != nil {
			return fmt.Errorf("%s: %w", table, e)
		}
	}
	rows, err := transaction.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Close()
		return fmt.Errorf("broken database references")
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var result string
	if err = transaction.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite integrity check failed")
	}
	return nil
}

func validateInventory(pkg *Package) error {
	for _, table := range tables {
		name := "tables/" + table + ".jsonl"
		if pkg.Files[name] == nil {
			return fmt.Errorf("missing required table: %s", table)
		}
		if _, ok := pkg.Manifest.Tables[table]; !ok {
			return fmt.Errorf("missing table count: %s", table)
		}
	}
	for _, entry := range pkg.Manifest.Entries {
		if strings.HasPrefix(entry.Name, "tables/") {
			table := strings.TrimSuffix(strings.TrimPrefix(entry.Name, "tables/"), ".jsonl")
			if !strings.HasSuffix(entry.Name, ".jsonl") || !tableSet[table] {
				return fmt.Errorf("unexpected database component")
			}
		} else if strings.HasPrefix(entry.Name, "files/") || strings.HasPrefix(entry.Name, "uploads/") {
			if pkg.Manifest.Kind != "full" {
				return fmt.Errorf("physical file in basic backup")
			}
			id := strings.TrimPrefix(strings.TrimPrefix(entry.Name, "files/"), "uploads/")
			if !validID(id) || strings.Contains(id, "/") {
				return fmt.Errorf("invalid physical file identity")
			}
		} else {
			return fmt.Errorf("unauthorized archive component")
		}
	}
	if pkg.Manifest.Kind == "basic" && pkg.Manifest.PhysicalFiles != 0 {
		return fmt.Errorf("basic backup claims physical files")
	}
	return nil
}

func (s *Service) stagedPackage(id string) (*Package, error) {
	op, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if op.Kind != "restore" || (op.State != "inspected" && op.State != "pending_restart" && op.State != "applying" && op.State != "attention") {
		return nil, fmt.Errorf("restore package not ready")
	}
	pathname := filepath.Join(s.directory, id+".plain")
	if _, err = os.Stat(pathname); os.IsNotExist(err) {
		pathname = filepath.Join(s.directory, id+".incoming")
	}
	pkg, err := inspectZip(pathname)
	if err != nil {
		return nil, err
	}
	if err = validateInventory(pkg); err != nil {
		pkg.Close()
		return nil, err
	}
	return pkg, nil
}

func (s *Service) CancelInspection(id string) error {
	op, err := s.Get(id)
	if err != nil {
		return err
	}
	if op.Kind != "restore" || op.State != "inspected" {
		return fmt.Errorf("cannot cancel this operation")
	}
	for _, suffix := range []string{".incoming", ".plain"} {
		_ = os.Remove(filepath.Join(s.directory, id+suffix))
	}
	op.State = "cancelled"
	op.Stage = "Vista previa descartada"
	op.Manifest = nil
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return s.writeOperation(op)
}
