package backups

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type Operation struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	State        string            `json:"state"`
	Stage        string            `json:"stage"`
	Actor        string            `json:"actor"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	Bytes        int64             `json:"bytes"`
	Error        string            `json:"error,omitempty"`
	Manifest     *Manifest         `json:"manifest,omitempty"`
	Destination  string            `json:"destination,omitempty"`
	RecoveryUser string            `json:"recovery_user,omitempty"`
	Mappings     map[string]string `json:"mappings,omitempty"`
}
type Service struct {
	DB        *storage.Database
	Config    config.Config
	directory string
	mu        sync.Mutex
	active    string
	cancel    context.CancelFunc
}

func New(db *storage.Database, cfg config.Config) (*Service, error) {
	directory := filepath.Join(cfg.StateDirectory, "backups")
	if err := storage.PrepareLocalPrivateDirectory(directory); err != nil {
		return nil, err
	}
	svc := &Service{DB: db, Config: cfg, directory: directory}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".backup-") && strings.HasSuffix(entry.Name(), ".zip") {
			_ = os.Remove(filepath.Join(directory, entry.Name()))
			continue
		}
		if strings.HasPrefix(entry.Name(), ".recovery-") && (strings.HasSuffix(entry.Name(), ".zip") || strings.HasSuffix(entry.Name(), ".aidexbak")) {
			_ = os.Remove(filepath.Join(directory, entry.Name()))
			continue
		}
		if strings.HasPrefix(entry.Name(), ".inspect-db-") && entry.IsDir() {
			_ = os.RemoveAll(filepath.Join(directory, entry.Name()))
			continue
		}
		if strings.HasPrefix(entry.Name(), ".operation-") && strings.HasSuffix(entry.Name(), ".tmp") {
			_ = os.Remove(filepath.Join(directory, entry.Name()))
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			var op Operation
			if svc.readOperation(strings.TrimSuffix(entry.Name(), ".json"), &op) == nil && op.State == "running" {
				op.State = "interrupted"
				op.Stage = "Interrupted by service restart"
				op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
				_ = svc.writeOperation(op)
				_ = os.Remove(filepath.Join(directory, op.ID+".aidexbak"))
			}
		}
	}
	return svc, nil
}
func (s *Service) opPath(id string) string { return filepath.Join(s.directory, id+".json") }
func validID(id string) bool {
	if len(id) < 8 || len(id) > 80 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}
func (s *Service) writeOperation(op Operation) error {
	if !validID(op.ID) {
		return fmt.Errorf("invalid operation id")
	}
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.directory, ".operation-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.opPath(op.ID))
}
func (s *Service) readOperation(id string, target *Operation) error {
	if !validID(id) {
		return fmt.Errorf("invalid operation id")
	}
	data, err := os.ReadFile(s.opPath(id))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, target); err != nil {
		return err
	}
	if target.ID != id {
		return fmt.Errorf("operation identity mismatch")
	}
	return nil
}
func (s *Service) Get(id string) (Operation, error) {
	var op Operation
	err := s.readOperation(id, &op)
	return op, err
}
func (s *Service) List() ([]Operation, error) {
	out, err := s.allOperations()
	if err != nil {
		return nil, err
	}
	if len(out) > 100 {
		out = out[:100]
	}
	return out, nil
}
func (s *Service) allOperations() ([]Operation, error) {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, err
	}
	out := []Operation{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var op Operation
		if s.readOperation(strings.TrimSuffix(entry.Name(), ".json"), &op) == nil {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}
func (s *Service) Path(id string) (string, error) {
	op, err := s.Get(id)
	if err != nil {
		return "", err
	}
	if op.Kind != "basic" && op.Kind != "full" || op.State != "complete" {
		return "", fmt.Errorf("backup not available")
	}
	path := filepath.Join(s.directory, id+".aidexbak")
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("invalid backup file")
	}
	return path, nil
}
func (s *Service) Start(kind, password, actor string) (Operation, error) {
	if kind != "basic" && kind != "full" {
		return Operation{}, fmt.Errorf("invalid backup kind")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != "" {
		return Operation{}, fmt.Errorf("another backup is running")
	}
	id := domain.NewID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	op := Operation{ID: id, Kind: kind, State: "running", Stage: "Validando datos", Actor: actor, CreatedAt: now, UpdatedAt: now}
	if err := s.writeOperation(op); err != nil {
		return Operation{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Hour)
	s.active = id
	s.cancel = cancel
	go func() {
		defer cancel()
		err := s.generate(ctx, op, password)
		s.mu.Lock()
		s.active = ""
		s.cancel = nil
		s.mu.Unlock()
		if err != nil {
			op.State = "failed"
			op.Stage = "No se generó el respaldo"
			if errors.Is(err, context.Canceled) {
				op.State = "cancelled"
				op.Stage = "Cancelado"
			}
			op.Error = err.Error()
			op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			_ = s.writeOperation(op)
			_ = s.audit(context.Background(), actor, "backup.generation_failed", map[string]any{"operation_id": id, "kind": kind, "error_code": "BACKUP_GENERATION_FAILED"})
		} else {
			_ = s.audit(context.Background(), actor, "backup.generation_completed", map[string]any{"operation_id": id, "kind": kind})
		}
	}()
	return op, nil
}
func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != id || s.cancel == nil {
		return fmt.Errorf("operation is not running")
	}
	s.cancel()
	return nil
}
func (s *Service) progress(op *Operation, stage string, bytes int64) {
	op.Stage = stage
	op.Bytes = bytes
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	_ = s.writeOperation(*op)
}

func (s *Service) generate(ctx context.Context, op Operation, password string) error {
	plain, err := os.CreateTemp(s.directory, ".backup-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(plain.Name())
	if err = plain.Chmod(0600); err != nil {
		plain.Close()
		return err
	}
	archive := zip.NewWriter(plain)
	manifest := Manifest{Product: "AIDEX", Format: 1, Schema: 15, Build: buildinfo.DisplayVersion(), Kind: op.Kind, CreatedAt: op.CreatedAt, Tables: map[string]int64{}, Entries: []Entry{}}
	tx, err := s.DB.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		plain.Close()
		return err
	}
	defer tx.Rollback()
	for _, query := range []string{"SELECT count(*) FROM upload_items WHERE status='receiving'", "SELECT count(*) FROM materializations WHERE state IN ('copying','verified','published')"} {
		var active int
		if err = tx.QueryRowContext(ctx, query).Scan(&active); err != nil {
			archive.Close()
			plain.Close()
			return err
		}
		if active > 0 {
			archive.Close()
			plain.Close()
			return fmt.Errorf("hay cargas o publicaciones en curso; vuelve a intentar al finalizar")
		}
	}
	var schema int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&schema); err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	// The archive still uses format/schema 15. EULA/deletion execution state is
	// installation-local, the library description is a portable column, and
	// migration 19 adds indexes only. Import accepts older library headers.
	if schema < 15 || schema > 19 {
		archive.Close()
		plain.Close()
		return fmt.Errorf("schema version incompatible: %d", schema)
	}
	for _, table := range tables {
		if err = ctx.Err(); err != nil {
			archive.Close()
			plain.Close()
			return err
		}
		s.progress(&op, "Exportando "+table, 0)
		count, entry, e := exportTable(ctx, tx, archive, table)
		if e != nil {
			archive.Close()
			plain.Close()
			return e
		}
		manifest.Tables[table] = count
		manifest.Entries = append(manifest.Entries, entry)
	}
	manifest.Documents = manifest.Tables["documents"]
	rootRows, err := tx.QueryContext(ctx, "SELECT id,library_id,storage_source,canonical_path,status FROM storage_roots ORDER BY id")
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	for rootRows.Next() {
		var root RootSummary
		if err = rootRows.Scan(&root.ID, &root.LibraryID, &root.Source, &root.Path, &root.Status); err != nil {
			rootRows.Close()
			archive.Close()
			plain.Close()
			return err
		}
		manifest.Roots = append(manifest.Roots, root)
	}
	err = rootRows.Err()
	rootRows.Close()
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	if op.Kind == "full" {
		if err = s.exportFiles(ctx, tx, archive, &manifest, &op); err != nil {
			archive.Close()
			plain.Close()
			return err
		}
	}
	raw, err := encodeManifest(manifest)
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	_, err = addZipEntry(archive, "manifest.json", raw)
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	if err = archive.Close(); err != nil {
		plain.Close()
		return err
	}
	if err = plain.Sync(); err != nil {
		plain.Close()
		return err
	}
	if err = plain.Close(); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	s.progress(&op, "Sellando paquete", 0)
	final := filepath.Join(s.directory, op.ID+".aidexbak")
	if err = seal(plain.Name(), final, password); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		os.Remove(final)
		return err
	}
	op.State = "complete"
	op.Stage = "Listo para descargar"
	op.Manifest = &manifest
	info, _ := os.Stat(final)
	if info != nil {
		op.Bytes = info.Size()
	}
	op.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return s.writeOperation(op)
}

func exportTable(ctx context.Context, tx *sql.Tx, archive *zip.Writer, table string) (int64, Entry, error) {
	if !tableSet[table] {
		return 0, Entry{}, fmt.Errorf("unlisted table")
	}
	columns, err := tableColumns(ctx, tx, table)
	if err != nil {
		return 0, Entry{}, err
	}
	query := "SELECT * FROM \"" + table + "\""
	if table == "audit_events" {
		query += " WHERE event_type NOT LIKE 'license.%'"
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return 0, Entry{}, err
	}
	defer rows.Close()
	name := "tables/" + table + ".jsonl"
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0600)
	header.SetModTime(time.Unix(0, 0))
	target, err := archive.CreateHeader(header)
	if err != nil {
		return 0, Entry{}, err
	}
	sum := sha256.New()
	counter := &countWriter{writer: io.MultiWriter(target, sum)}
	encoder := json.NewEncoder(counter)
	if err = encoder.Encode(columns); err != nil {
		return 0, Entry{}, err
	}
	var count int64
	for rows.Next() {
		if err = ctx.Err(); err != nil {
			return 0, Entry{}, err
		}
		items := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range items {
			ptrs[i] = &items[i]
		}
		if err = rows.Scan(ptrs...); err != nil {
			return 0, Entry{}, err
		}
		for i, item := range items {
			if b, ok := item.([]byte); ok {
				items[i] = string(b)
			}
		}
		if err = encoder.Encode(items); err != nil {
			return 0, Entry{}, err
		}
		count++
		if count > 5000000 {
			return 0, Entry{}, fmt.Errorf("table row limit exceeded")
		}
	}
	if err = rows.Err(); err != nil {
		return 0, Entry{}, err
	}
	return count, Entry{Name: name, Size: counter.n, SHA256: hex.EncodeToString(sum.Sum(nil))}, nil
}

type countWriter struct {
	writer io.Writer
	n      int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.n += int64(n)
	return n, err
}
func tableColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info(\""+table+"\")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if err = rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("missing table %s", table)
	}
	return out, nil
}

type physical struct {
	name, root, relative, expected string
	size                           int64
}

func (s *Service) exportFiles(ctx context.Context, tx *sql.Tx, archive *zip.Writer, m *Manifest, op *Operation) error {
	targets := []physical{}
	rows, err := tx.QueryContext(ctx, `SELECT f.id,r.canonical_path,l.relative_path,coalesce(v.sha256,''),coalesce(v.size_bytes,0) FROM physical_files f JOIN physical_file_locations l ON l.id=f.primary_location_id JOIN storage_roots r ON r.id=l.root_id LEFT JOIN content_versions v ON v.id=f.current_content_version_id WHERE f.storage_source='managed' AND f.availability='available' AND l.retired_at IS NULL AND EXISTS(SELECT 1 FROM documents d WHERE d.physical_file_id=f.id AND d.deleted_at IS NULL) ORDER BY f.id`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, root, relative, digest string
		var size int64
		if err = rows.Scan(&id, &root, &relative, &digest, &size); err != nil {
			rows.Close()
			return err
		}
		if !safeRelative(relative) {
			rows.Close()
			return fmt.Errorf("unsafe source document path")
		}
		if len(digest) != 64 {
			rows.Close()
			return fmt.Errorf("managed document has no registered SHA-256")
		}
		targets = append(targets, physical{name: "files/" + id, root: root, relative: relative, expected: digest, size: size})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	rows, err = tx.QueryContext(ctx, `SELECT id,private_temporary_locator,sha256,size_bytes FROM upload_items WHERE status IN ('staged','retained') ORDER BY id`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, locator, digest string
		var size int64
		if err = rows.Scan(&id, &locator, &digest, &size); err != nil {
			rows.Close()
			return err
		}
		if !safeRelative(locator) || strings.ContainsAny(locator, "/\\") {
			rows.Close()
			return fmt.Errorf("unsafe private upload locator")
		}
		if len(digest) != 64 {
			rows.Close()
			return fmt.Errorf("private upload has no registered SHA-256")
		}
		targets = append(targets, physical{name: "uploads/" + id, root: s.Config.PrivateUploadDirectory(), relative: locator, expected: digest, size: size})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for i, target := range targets {
		if err = ctx.Err(); err != nil {
			return err
		}
		s.progress(op, fmt.Sprintf("Copiando documento %d de %d", i+1, len(targets)), int64(i))
		entry, e := copyPhysical(ctx, archive, target)
		if e != nil {
			return fmt.Errorf("no se pudo leer un archivo necesario: %w", e)
		}
		m.Entries = append(m.Entries, entry)
		m.PhysicalFiles++
	}
	return nil
}
func safeRelative(value string) bool {
	return value != "" && !strings.Contains(value, "\\") && path.Clean(value) == value && filepath.IsLocal(filepath.FromSlash(value))
}
func copyPhysical(ctx context.Context, archive *zip.Writer, target physical) (Entry, error) {
	if err := safeDirectory(target.root); err != nil {
		return Entry{}, err
	}
	container, err := os.OpenRoot(target.root)
	if err != nil {
		return Entry{}, err
	}
	defer container.Close()
	component := ""
	for _, part := range strings.Split(target.relative, "/") {
		component = filepath.Join(component, part)
		info, e := container.Lstat(component)
		if e != nil {
			return Entry{}, e
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return Entry{}, fmt.Errorf("symbolic link or special source file")
		}
	}
	before, err := container.Lstat(filepath.FromSlash(target.relative))
	if err != nil {
		return Entry{}, err
	}
	file, err := container.Open(filepath.FromSlash(target.relative))
	if err != nil {
		return Entry{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return Entry{}, err
	}
	if !before.Mode().IsRegular() || !os.SameFile(before, opened) {
		return Entry{}, fmt.Errorf("not a regular file")
	}
	reader := &contextReader{ctx: ctx, source: file}
	entry, err := addZipEntry(archive, target.name, reader)
	if err != nil {
		return Entry{}, err
	}
	after, err := file.Stat()
	if err != nil {
		return Entry{}, err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return Entry{}, fmt.Errorf("file changed during backup")
	}
	if target.expected != "" && entry.SHA256 != target.expected {
		return Entry{}, fmt.Errorf("file hash differs from registered version")
	}
	if entry.Size != target.size {
		return Entry{}, fmt.Errorf("file size differs from registered version")
	}
	return entry, nil
}

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(p)
}
