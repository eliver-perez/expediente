package httpapi

import (
	"database/sql"
	"net/http"
	"runtime"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
)

func (s *Server) networkConfiguration(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if s.network == nil {
		s.fail(w, r, domain.Failure("NETWORK_UNAVAILABLE", "El servicio de red no está disponible en este proceso.", 503))
		return
	}
	if r.Method == http.MethodPut {
		var input struct {
			Mode     string `json:"mode"`
			Port     int    `json:"port"`
			Revision string `json:"revision"`
		}
		if err := decodeJSON(w, r, &input); err != nil {
			s.fail(w, r, err)
			return
		}
		if input.Mode != "local" && input.Mode != "lan" {
			s.fail(w, r, domain.Failure("INVALID_REQUEST", "Selecciona acceso local o desde la red.", 422))
			return
		}
		current := s.network.Diagnose(r.Context())
		if input.Port == 0 {
			input.Port = current.Port
		}
		if input.Port < 1 || input.Port > 65535 {
			s.fail(w, r, domain.Failure("INVALID_REQUEST", "El puerto debe estar entre 1 y 65535.", 422))
			return
		}
		if runtime.GOOS == "windows" && input.Port != current.Port && (current.Mode == "lan" || input.Mode == "lan") {
			s.fail(w, r, domain.Failure("NETWORK_ADMIN_REQUIRED", "Para cambiar el puerto de acceso LAN y su regla de firewall en Windows, ejecuta «Configurar servidor AIDEX» como administrador desde Inicio.", 409))
			return
		}
		if runtime.GOOS == "windows" && input.Mode == "lan" && current.Mode != "lan" && current.Firewall.State != "managed" {
			s.fail(w, r, domain.Failure("NETWORK_ADMIN_REQUIRED", "No se pudo confirmar una regla de firewall válida para este puerto. Ejecuta «Configurar servidor AIDEX» desde Inicio antes de habilitar la red local.", 409))
			return
		}
		// Recheck the administrator before a system mutation; record the request
		// even when binding or persisting later fails. No credentials enter audit.
		err := s.identity.AuthorizedWrite(r.Context(), p, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
			return audit.Append(r.Context(), tx, time.Now(), audit.Event{Type: "system.network_change_requested", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata(r), Details: map[string]any{"mode": input.Mode, "port": input.Port}})
		})
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if err = s.network.ApplyPort(input.Mode, input.Port, input.Revision); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	writeJSON(w, 200, s.network.Diagnose(r.Context()))
}
