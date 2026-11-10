package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gestor-documental/internal/domain"
)

func (s *Server) backupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/backups", s.protected("", true, s.backupList))
	mux.HandleFunc("POST /api/v1/backups", s.protected("backup.manage", true, s.backupCreate))
	mux.HandleFunc("GET /api/v1/backups/{id}", s.protected("backup.manage", true, s.backupGet))
	mux.HandleFunc("POST /api/v1/backups/{id}/cancel", s.protected("backup.manage", true, s.backupCancel))
	mux.HandleFunc("GET /api/v1/backups/{id}/download", s.protected("backup.manage", true, s.backupDownload))
	mux.HandleFunc("POST /api/v1/restores/inspect", s.protected("restore.manage", true, s.restoreInspect))
	mux.HandleFunc("POST /api/v1/restores/{id}/confirm", s.protected("restore.manage", true, s.restoreConfirm))
	mux.HandleFunc("POST /api/v1/restores/{id}/retry", s.protected("restore.manage", true, s.restoreRetry))
	mux.HandleFunc("POST /api/v1/restores/{id}/discard", s.protected("restore.manage", true, s.restoreDiscard))
}

func (s *Server) backupList(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if !p.Can("backup.manage") && !p.Can("restore.manage") {
		s.fail(w, r, domain.Failure("FORBIDDEN", "No tienes permiso para consultar respaldos.", 403))
		return
	}
	if s.backups == nil {
		s.fail(w, r, domain.Failure("BACKUP_UNAVAILABLE", "Respaldos no disponibles.", 503))
		return
	}
	items, err := s.backups.List()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) backupGet(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	op, err := s.backups.Get(r.PathValue("id"))
	if err != nil {
		s.fail(w, r, domain.Failure("BACKUP_NOT_FOUND", "No se encontró la operación.", 404))
		return
	}
	writeJSON(w, 200, op)
}
func (s *Server) backupCreate(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	var input struct {
		Kind     string `json:"kind"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, r, err)
		return
	}
	if len(input.Password) > 4096 {
		s.fail(w, r, domain.Failure("INVALID_REQUEST", "La contraseña es demasiado larga.", 422))
		return
	}
	op, err := s.backups.Start(input.Kind, input.Password, p.User.ID)
	if err != nil {
		s.fail(w, r, domain.Failure("BACKUP_CONFLICT", err.Error(), 409))
		return
	}
	_ = s.backups.RecordEvent(r.Context(), p.User.ID, "backup.generation_started", map[string]any{"operation_id": op.ID, "kind": op.Kind})
	writeJSON(w, 202, op)
}
func (s *Server) backupCancel(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if err := s.backups.Cancel(r.PathValue("id")); err != nil {
		s.fail(w, r, domain.Failure("BACKUP_CONFLICT", err.Error(), 409))
		return
	}
	writeJSON(w, 202, map[string]string{"status": "cancelling"})
}
func (s *Server) backupDownload(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	pathname, err := s.backups.Path(r.PathValue("id"))
	if err != nil {
		s.fail(w, r, domain.Failure("BACKUP_NOT_FOUND", "El respaldo no está disponible.", 404))
		return
	}
	file, err := os.Open(pathname)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !info.Mode().IsRegular() {
		s.fail(w, r, domain.Failure("BACKUP_NOT_FOUND", "El respaldo no está disponible.", 404))
		return
	}
	_ = s.backups.RecordEvent(r.Context(), p.User.ID, "backup.downloaded", map[string]any{"operation_id": r.PathValue("id")})
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.aidexbak\"", filepath.Base(r.PathValue("id"))))
	http.ServeContent(w, r, "", info.ModTime(), file)
}
func (s *Server) restoreInspect(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if s.backups == nil {
		s.fail(w, r, domain.Failure("BACKUP_UNAVAILABLE", "Restauración no disponible.", 503))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<30)
	multipart, err := r.MultipartReader()
	if err != nil {
		s.fail(w, r, domain.Failure("INVALID_REQUEST", "Adjunta un archivo de respaldo.", 400))
		return
	}
	password := ""
	part, err := multipart.NextPart()
	if err != nil {
		s.fail(w, r, domain.Failure("INVALID_REQUEST", "Falta el archivo.", 400))
		return
	}
	if part.FormName() == "password" {
		raw, e := io.ReadAll(io.LimitReader(part, 4097))
		part.Close()
		if e != nil || len(raw) > 4096 {
			s.fail(w, r, domain.Failure("INVALID_REQUEST", "Contraseña no válida.", 422))
			return
		}
		password = string(raw)
		part, err = multipart.NextPart()
		if err != nil {
			s.fail(w, r, domain.Failure("INVALID_REQUEST", "Falta el archivo.", 400))
			return
		}
	}
	if part.FormName() != "file" {
		s.fail(w, r, domain.Failure("INVALID_REQUEST", "Falta el archivo.", 400))
		return
	}
	op, err := s.backups.Inspect(r.Context(), part, password, p.User.ID)
	part.Close()
	if err != nil {
		s.fail(w, r, domain.Failure("BACKUP_INVALID", err.Error(), 422))
		return
	}
	_ = s.backups.RecordEvent(r.Context(), p.User.ID, "backup.inspected", map[string]any{"operation_id": op.ID, "kind": op.Manifest.Kind})
	writeJSON(w, 200, op)
}
func (s *Server) restoreConfirm(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	var input struct {
		Destination  string            `json:"destination"`
		Confirmation string            `json:"confirmation"`
		Mappings     map[string]string `json:"mappings"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, r, err)
		return
	}
	if strings.TrimSpace(input.Confirmation) != "RESTAURAR" {
		s.fail(w, r, domain.Failure("RESTORE_CONFIRMATION_REQUIRED", "Confirma expresamente la restauración.", 422))
		return
	}
	op, err := s.backups.ScheduleRestoreWithMappings(r.Context(), r.PathValue("id"), strings.TrimSpace(input.Destination), input.Mappings, p.User.ID, s.identity.License)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, 202, op)
}
func (s *Server) restoreDiscard(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	op, err := s.backups.Get(r.PathValue("id"))
	if err != nil || op.Actor != p.User.ID {
		s.fail(w, r, domain.Failure("BACKUP_NOT_FOUND", "No se encontró la vista previa.", 404))
		return
	}
	if err = s.backups.CancelInspection(op.ID); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "cancelled"})
}
func (s *Server) restoreRetry(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	op, err := s.backups.Get(r.PathValue("id"))
	if err != nil || op.State != "attention" || op.Actor != p.User.ID {
		s.fail(w, r, domain.Failure("BACKUP_NOT_FOUND", "No se encontró una restauración reintentable.", 404))
		return
	}
	result, err := s.backups.ScheduleRestoreWithMappings(r.Context(), op.ID, op.Destination, op.Mappings, p.User.ID, s.identity.License)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, 202, result)
}
