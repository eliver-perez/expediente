package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"runtime"
	"time"

	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type DiagnosticEvent struct {
	ID          string              `json:"id"`
	First       string              `json:"occurred_at"`
	Last        string              `json:"last_occurred_at"`
	Module      string              `json:"module"`
	Code        string              `json:"code"`
	Severity    string              `json:"severity"`
	Status      string              `json:"status"`
	Message     string              `json:"message"`
	Context     diagnostics.Context `json:"context"`
	Occurrences int                 `json:"occurrences"`
	Revision    int64               `json:"revision"`
}
type DiagnosticFilter struct{ From, To, Module, Severity, Status string }

func (f DiagnosticFilter) validate() error {
	for _, date := range []string{f.From, f.To} {
		if date != "" {
			if _, err := time.Parse(time.RFC3339Nano, date); err != nil {
				return invalid("La fecha del filtro no es válida.")
			}
		}
	}
	if f.From != "" && f.To != "" {
		a, _ := time.Parse(time.RFC3339Nano, f.From)
		b, _ := time.Parse(time.RFC3339Nano, f.To)
		if !a.Before(b) {
			return invalid("El inicio debe ser anterior al fin del período.")
		}
	}
	if f.Module != "" && diagnostics.Module(f.Module) != f.Module {
		return invalid("Módulo inválido.")
	}
	switch f.Severity {
	case "", "info", "warning", "error", "critical":
	default:
		return invalid("Severidad inválida.")
	}
	switch f.Status {
	case "", "open", "reviewed":
	default:
		return invalid("Estado inválido.")
	}
	return nil
}

const diagnosticColumns = "id,occurred_at,last_occurred_at,module,code,severity,status,context_json,occurrences,revision"

func readDiagnostic(row rowScanner) (DiagnosticEvent, error) {
	var event DiagnosticEvent
	var raw string
	err := row.Scan(&event.ID, &event.First, &event.Last, &event.Module, &event.Code, &event.Severity, &event.Status, &raw, &event.Occurrences, &event.Revision)
	if err != nil {
		return event, err
	}
	if _, known := diagnostics.Messages[event.Code]; !known {
		event.Code = "INTERNAL_ERROR"
	}
	event.Message = diagnostics.Messages[event.Code]
	event.Module = diagnostics.Module(event.Module)
	if err = json.Unmarshal([]byte(raw), &event.Context); err != nil {
		event.Context = diagnostics.Context{}
	}
	event.Context = event.Context.Safe()
	return event, nil
}
func systemAccess(p domain.Principal) error {
	if !p.Can("system.configure") {
		return domain.Failure("FORBIDDEN", "No tienes permiso para administrar el sistema.", 403)
	}
	return nil
}
func (s *Service) DiagnosticEvents(ctx context.Context, p domain.Principal, f DiagnosticFilter, cursor string) (Page[DiagnosticEvent], error) {
	result := Page[DiagnosticEvent]{Items: []DiagnosticEvent{}}
	if err := systemAccess(p); err != nil {
		return result, err
	}
	if err := f.validate(); err != nil {
		return result, err
	}
	scope := p.User.ID + ":diagnostics:" + encode(f)
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	from, to := "", ""
	if f.From != "" {
		t, _ := time.Parse(time.RFC3339Nano, f.From)
		from = domain.Timestamp(t)
	}
	if f.To != "" {
		t, _ := time.Parse(time.RFC3339Nano, f.To)
		to = domain.Timestamp(t)
	}
	rows, err := s.Database.Reader.QueryContext(ctx, `SELECT `+diagnosticColumns+` FROM diagnostic_events WHERE (?='' OR last_occurred_at>=?) AND (?='' OR last_occurred_at<?) AND (?='' OR module=?) AND (?='' OR severity=?) AND (?='' OR status=?) AND (?='' OR last_occurred_at<? OR (last_occurred_at=? AND id<?)) ORDER BY last_occurred_at DESC,id DESC LIMIT 51`, from, from, to, to, f.Module, f.Module, f.Severity, f.Severity, f.Status, f.Status, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		event, err := readDiagnostic(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, event)
	}
	if len(result.Items) > 50 {
		result.Items = result.Items[:50]
		last := result.Items[49]
		result.Next = nextListCursor(scope, last.Last, last.ID)
	}
	return result, rows.Err()
}
func (s *Service) DiagnosticPolicy(ctx context.Context, p domain.Principal) (diagnostics.Policy, error) {
	if err := systemAccess(p); err != nil {
		return diagnostics.Policy{}, err
	}
	return diagnostics.ReadPolicy(ctx, s.Database.Reader)
}
func (s *Service) ConfigureDiagnostics(ctx context.Context, p domain.Principal, input diagnostics.Policy, m domain.RequestMetadata) error {
	if input.Days < 1 || input.Days > 365 || input.Maximum < 100 || input.Maximum > 50000 {
		return invalid("Usa de 1 a 365 días y de 100 a 50000 registros.")
	}
	return s.Identity.AuthorizedWrite(ctx, p, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
		result, err := tx.ExecContext(ctx, "UPDATE diagnostic_policy SET retention_days=?,maximum_events=?,revision=revision+1 WHERE singleton=1 AND revision=?", input.Days, input.Maximum, input.Revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("REVISION_CONFLICT", "La retención cambió. Actualiza antes de guardar.", 409)
		}
		if err = diagnostics.PruneTx(ctx, tx, time.Now()); err != nil {
			return err
		}
		return record(ctx, tx, current, m, "system.diagnostics_configured", "", "", map[string]any{"retention_days": input.Days, "maximum_events": input.Maximum})
	})
}
func (s *Service) ReviewDiagnostic(ctx context.Context, p domain.Principal, id, status string, revision int64, m domain.RequestMetadata) error {
	if status != "open" && status != "reviewed" {
		return invalid("Estado inválido.")
	}
	return s.Identity.AuthorizedWrite(ctx, p, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
		result, err := tx.ExecContext(ctx, "UPDATE diagnostic_events SET status=?,revision=revision+1 WHERE id=? AND revision=?", status, id, revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("REVISION_CONFLICT", "El registro cambió o ya venció su retención. Actualiza el listado.", 409)
		}
		return record(ctx, tx, current, m, "system.diagnostic_reviewed", "", "", map[string]any{"diagnostic_id": id, "status": status})
	})
}

type SupportDiagnostic struct {
	Version      string          `json:"version"`
	OS           string          `json:"operating_system"`
	Architecture string          `json:"architecture"`
	Event        DiagnosticEvent `json:"event"`
}

func (s *Service) SupportDiagnostic(ctx context.Context, p domain.Principal, id string) (SupportDiagnostic, error) {
	result := SupportDiagnostic{Version: buildinfo.DisplayVersion(), OS: runtime.GOOS, Architecture: runtime.GOARCH}
	if err := systemAccess(p); err != nil {
		return result, err
	}
	event, err := readDiagnostic(s.Database.Reader.QueryRowContext(ctx, "SELECT "+diagnosticColumns+" FROM diagnostic_events WHERE id=?", id))
	if err == sql.ErrNoRows {
		return result, notFound()
	}
	result.Event = event
	return result, err
}
func recentDiagnostics(ctx context.Context, q storage.Querier) ([]DiagnosticEvent, error) {
	events := []DiagnosticEvent{}
	rows, err := q.QueryContext(ctx, "SELECT "+diagnosticColumns+" FROM diagnostic_events WHERE severity IN ('error','critical') ORDER BY last_occurred_at DESC,id DESC LIMIT 5")
	if err != nil {
		return events, err
	}
	defer rows.Close()
	for rows.Next() {
		event, err := readDiagnostic(rows)
		if err != nil {
			return events, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
