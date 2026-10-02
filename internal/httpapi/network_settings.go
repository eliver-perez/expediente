package httpapi

import (
	"database/sql"
	"net/http"
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
		// Recheck the administrator before a system mutation; record the request
		// even when binding or persisting later fails. No credentials enter audit.
		err := s.identity.AuthorizedWrite(r.Context(), p, "system.configure", func(tx *sql.Tx, current domain.Principal) error {
			return audit.Append(r.Context(), tx, time.Now(), audit.Event{Type: "system.network_change_requested", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata(r), Details: map[string]any{"mode": input.Mode}})
		})
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if err = s.network.Apply(input.Mode, input.Revision); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	writeJSON(w, 200, s.network.Diagnose(r.Context()))
}
