package licensing

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestTransientFailuresKeepSignedStateAndGraceAfterRestart(t *testing.T) {
	for _, kind := range []string{"subscription", "perpetual"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			client, key := testClient(t)
			client.Options.ServerURL = "https://license.example"
			client.Options.RefreshHours = 24
			claims := testClaims(client)
			claims.Type = kind
			expires := client.Now().Add(time.Hour)
			if kind == "subscription" {
				value := expires.Format(time.RFC3339)
				claims.ExpiresAt = &value
				claims.GraceDays = 15
			}
			importClaims(t, client, key, claims)
			contact := client.Now().UTC()
			if err := client.Database.Write(ctx, func(tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, "UPDATE license_state SET last_contact_at=?", contact.Format(time.RFC3339))
				return err
			}); err != nil {
				t.Fatal(err)
			}
			client.recordContact(ctx, remoteFailure("TEMPORARY_UNAVAILABLE"))
			restarted, err := New(client.Database, client.stateDirectory, client.Options)
			if err != nil {
				t.Fatal(err)
			}
			restarted.Now = func() time.Time { return expires.Add(14 * 24 * time.Hour) }
			state, err := restarted.Status(ctx)
			if err != nil || !state.WriteAllowed || state.Revision != claims.Revision || state.LastContact != contact.Format(time.RFC3339) || state.LastError != "TEMPORARY_UNAVAILABLE" {
				t.Fatal("outage replaced valid state", state, err)
			}
			if state.NextValidation != contact.Add(24*time.Hour).Format(time.RFC3339) {
				t.Fatal("incorrect next validation", state.NextValidation)
			}
			if kind == "subscription" && state.State != "grace" || kind == "perpetual" && state.State != "active" {
				t.Fatal(state.State)
			}
			restarted.Now = func() time.Time { return expires.Add(15 * 24 * time.Hour) }
			state, err = restarted.Status(ctx)
			if err != nil || !state.ReadAllowed {
				t.Fatal(state, err)
			}
			if kind == "subscription" && (state.WriteAllowed || state.State != "expired") {
				t.Fatal("grace extended by outage")
			}
			if kind == "perpetual" && !state.WriteAllowed {
				t.Fatal("perpetual license expired offline")
			}
			if err = restarted.Check(ctx, SecurityAdministration); err != nil {
				t.Fatal("recovery blocked", err)
			}
		})
	}
}
