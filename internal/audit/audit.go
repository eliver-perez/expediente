package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"gestor-documental/internal/domain"
)

// Details are constructed by each use case; never pass request bodies or credentials here.
type Event struct {
	ID          string
	LibraryID   string
	DocumentID  string
	Type        string
	ActorUserID string
	SystemActor bool
	SessionID   string
	Metadata    domain.RequestMetadata
	Details     map[string]any
}

func Append(ctx context.Context, transaction *sql.Tx, now time.Time, event Event) error {
	actorKind := "user"
	if event.ActorUserID == "" {
		actorKind = "anonymous"
	}
	if event.SystemActor || event.Metadata.ObservedIP == "local" {
		actorKind = "system"
	}
	actorID := any(nil)
	if actorKind == "user" {
		actorID = event.ActorUserID
	}
	if event.Details == nil {
		event.Details = map[string]any{}
	}
	details, err := json.Marshal(event.Details)
	if err != nil {
		return err
	}
	if event.ID == "" {
		event.ID = domain.NewID()
	}
	_, err = transaction.ExecContext(ctx, `INSERT INTO audit_events
		(id, occurred_at, actor_kind, actor_user_id, session_id, observed_ip, request_id, event_type, details_json,library_id,document_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?,?,?)`, event.ID, domain.Timestamp(now), actorKind,
		actorID, nullable(event.SessionID), nullable(event.Metadata.ObservedIP), event.Metadata.RequestID, event.Type, string(details), nullable(event.LibraryID), nullable(event.DocumentID))
	return err
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
