package libraries

import (
	"context"
	"database/sql"
	"gestor-documental/internal/domain"
	"strings"
	"time"
)

// These endpoints query retained rows, not a client-side window of 100 records.
type TableFilter struct {
	Query, State, Kind, From, Until, Sort, Direction string
	Page, Size                                       int
}
type TablePage[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Size  int `json:"page_size"`
}

func (f TableFilter) validate() error {
	if f.Page < 1 || f.Page > 1000000 || (f.Size != 10 && f.Size != 25 && f.Size != 50 && f.Size != 100) || len(f.Query) > 240 || len(f.State) > 80 || len(f.Kind) > 80 {
		return invalid("Filtros o paginación inválidos.")
	}
	if f.Direction != "asc" && f.Direction != "desc" {
		return invalid("Orden inválido.")
	}
	for _, value := range []string{f.From, f.Until} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return invalid("Fecha inválida.")
			}
		}
	}
	if f.From != "" && f.Until != "" && f.From > f.Until {
		return invalid("El inicio debe ser anterior al fin.")
	}
	return nil
}
func likeLiteral(value string) string {
	return "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value) + "%"
}
func (s *Service) FileSkips(ctx context.Context, p domain.Principal, library string, f TableFilter) (TablePage[ScanError], error) {
	result := TablePage[ScanError]{Items: []ScanError{}, Page: f.Page, Size: f.Size}
	if err := s.Read(ctx, p, library, "indexing.run"); err != nil {
		return result, err
	}
	if err := s.require(ctx, s.Database.Reader, p, library, "storage.view_paths"); err != nil {
		return result, err
	}
	if err := f.validate(); err != nil {
		return result, err
	}
	column := map[string]string{"date": "e.observed_at", "path": "e.relative_path", "reason": "e.error_code"}[f.Sort]
	if column == "" {
		return result, invalid("Orden inválido.")
	}
	base := ` FROM document_file_skips e JOIN storage_roots r ON r.id=e.root_id WHERE r.library_id=? AND r.status<>'superseded' AND e.relative_path LIKE ? ESCAPE '\' AND (?='' OR e.error_code=?)`
	args := []any{library, likeLiteral(f.Query), f.State, f.State}
	tx, err := s.Database.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, "SELECT count(*)"+base, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	result.Page = boundedPage(f.Page, f.Size, result.Total)
	rows, err := tx.QueryContext(ctx, "SELECT e.root_id,e.relative_path,e.error_code,e.observed_at"+base+" ORDER BY "+column+" "+f.Direction+",e.root_id,e.relative_path LIMIT ? OFFSET ?", append(args, f.Size, (result.Page-1)*f.Size)...)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item ScanError
		if err = rows.Scan(&item.RootID, &item.Path, &item.Code, &item.At); err != nil {
			rows.Close()
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func boundedPage(page, size, total int) int {
	last := (total + size - 1) / size
	if last < 1 {
		last = 1
	}
	if page > last {
		return last
	}
	return page
}
func (s *Service) JobHistory(ctx context.Context, p domain.Principal, library string, f TableFilter) (TablePage[Job], error) {
	result := TablePage[Job]{Items: []Job{}, Page: f.Page, Size: f.Size}
	if err := s.Read(ctx, p, library, "indexing.run"); err != nil {
		return result, err
	}
	if err := f.validate(); err != nil {
		return result, err
	}
	column := map[string]string{"date": "j.created_at", "name": "coalesce(d.original_filename,'')", "state": "j.status", "kind": "j.job_type"}[f.Sort]
	if column == "" {
		return result, invalid("Orden inválido.")
	}
	paths := s.require(ctx, s.Database.Reader, p, library, "storage.view_paths") == nil
	base := ` FROM jobs j LEFT JOIN documents d ON d.physical_file_id=j.physical_file_id LEFT JOIN physical_files f ON f.id=j.physical_file_id LEFT JOIN physical_file_locations l ON l.id=f.primary_location_id LEFT JOIN storage_roots r ON r.id=CASE WHEN j.job_type IN ('scan','verify_managed') THEN j.target_version ELSE l.root_id END WHERE j.library_id=? AND (j.physical_file_id IS NULL OR (` + visibleDocumentSQL + `)) AND (?='' OR j.status=?) AND (?='' OR j.job_type=?) AND (?='' OR j.created_at>=?) AND (?='' OR substr(j.created_at,1,10)<=?) AND (coalesce(d.original_filename,'') LIKE ? ESCAPE '\' OR j.id LIKE ? ESCAPE '\' OR j.last_error_code LIKE ? ESCAPE '\'`
	args := []any{library, p.User.ID, p.User.ID, f.State, f.State, f.Kind, f.Kind, f.From, f.From, f.Until, f.Until, likeLiteral(f.Query), likeLiteral(f.Query), likeLiteral(f.Query)}
	if paths {
		base += ` OR coalesce(r.canonical_path,'') LIKE ? ESCAPE '\' OR coalesce(l.relative_path,'') LIKE ? ESCAPE '\'`
		args = append(args, likeLiteral(f.Query), likeLiteral(f.Query))
	}
	base += ")"
	tx, err := s.Database.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, "SELECT count(*)"+base, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	result.Page = boundedPage(f.Page, f.Size, result.Total)
	rows, err := tx.QueryContext(ctx, "SELECT j.id,j.library_id,coalesce(j.physical_file_id,''),j.job_type,j.status,j.attempt_count,coalesce(j.last_error_code,''),j.created_at"+base+" ORDER BY "+column+" "+f.Direction+",j.id "+f.Direction+" LIMIT ? OFFSET ?", append(args, f.Size, (result.Page-1)*f.Size)...)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var j Job
		if err = rows.Scan(&j.ID, &j.LibraryID, &j.FileID, &j.Kind, &j.Status, &j.Attempts, &j.Error, &j.Created); err != nil {
			rows.Close()
			return result, err
		}
		result.Items = append(result.Items, j)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	for i := range result.Items {
		if err = s.enrichJob(ctx, &result.Items[i]); err != nil {
			return result, err
		}
		if !paths {
			result.Items[i].RootPath = ""
		}
	}
	return result, nil
}
