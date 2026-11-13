package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestIndependentWritersReadersAndRollback(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	database, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	held, err := database.Writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Rollback()
	if _, err = held.Exec("INSERT INTO roles VALUES('uncommitted','Uncommitted','global')"); err != nil {
		t.Fatal(err)
	}
	concurrentCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var workers sync.WaitGroup
	failures := make(chan error, 24)
	for n := 0; n < 24; n++ {
		workers.Go(func() {
			failures <- database.Write(concurrentCtx, func(tx *sql.Tx) error {
				_, err := tx.Exec("INSERT INTO audit_events(id,occurred_at,actor_kind,event_type,details_json) VALUES(gen_random_uuid()::text,'2026-11-13T00:00:00.000000Z','system','concurrent','{}')")
				return err
			})
		})
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal("independent writers blocked behind an open transaction", err)
		}
	}
	var count int
	if err = database.Reader.QueryRow("SELECT count(*) FROM audit_events").Scan(&count); err != nil || count != 24 {
		t.Fatal(count, err)
	}
	if err = database.Reader.QueryRow("SELECT count(*) FROM roles WHERE id='uncommitted'").Scan(&count); err != nil || count != 0 {
		t.Fatal("reader observed uncommitted data", count, err)
	}
	if err = held.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err = database.Writer.Exec("INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,credential_version,started_at,last_activity_at,expires_at,observed_ip,user_agent) VALUES('invalid','absent','token','csrf',1,'a','a','z','local','test')"); err == nil {
		t.Fatal("foreign key accepted an absent user")
	}
}

func TestRepeatableReadAndDeferredIntegrity(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	database, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	tx, err := database.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var before, after int
	if err = tx.QueryRow("SELECT count(*) FROM roles").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err = database.Writer.Exec("INSERT INTO roles VALUES('later','Later','global')"); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow("SELECT count(*) FROM roles").Scan(&after); err != nil || after != before {
		t.Fatal("snapshot changed", before, after, err)
	}
	write, err := database.Writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer write.Rollback()
	if _, err = write.Exec("SET CONSTRAINTS ALL DEFERRED"); err != nil {
		t.Fatal(err)
	}
	if _, err = write.Exec("INSERT INTO role_permissions VALUES('absent','permission')"); err != nil {
		t.Fatal("deferred reference checked too soon", err)
	}
	if err = write.Commit(); err == nil {
		t.Fatal("commit accepted a broken deferred reference")
	}
}
