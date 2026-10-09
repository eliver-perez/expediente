//go:build development

package libraries

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/previews"
)

func TestOutcomeWriteBackoffStopsOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempted, stopped := make(chan struct{}), make(chan bool, 1)
	go func() {
		stopped <- retryWorkerWrite(ctx, diagnostics.Context{Operation: "extract"}, func(context.Context) error {
			close(attempted)
			return errors.New("storage unavailable")
		})
	}()
	<-attempted
	cancel()
	select {
	case result := <-stopped:
		if result {
			t.Fatal("failed write reported success")
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown stuck in journal retry")
	}
}

func awaitOutcome(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("worker did not recover its outcome after storage became writable")
}

func TestWorkerRecoversFailedOutcomeWriteWithoutRepeatingExtraction(t *testing.T) {
	s, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, s, admin, "managed")
	allowContentFormats(t, s, admin)
	item := uploadFormat(t, s, admin, library, "Informe ñ #100%.txt", []byte("Contenido para comprobar recuperación"))
	_, err := s.Database.Writer.Exec(`CREATE TRIGGER qa_deny_finish BEFORE UPDATE OF status ON jobs WHEN OLD.status='running' AND NEW.status='succeeded' BEGIN SELECT RAISE(ABORT,'SECRET-QA-SQL-PATH'); END`)
	requireNoError(t, err)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	runtime, err := s.Start(ctx)
	requireNoError(t, err)
	defer runtime.Close()
	awaitOutcome(t, func() bool {
		var count int
		requireNoError(t, s.Database.Reader.QueryRow(`SELECT count(*) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=? AND f.indexed_extraction_id IS NOT NULL`, item.DocumentID).Scan(&count))
		return count == 1
	})
	// The extraction has published, but its terminal-state transaction is denied.
	time.Sleep(300 * time.Millisecond)
	_, err = s.Database.Writer.Exec("DROP TRIGGER qa_deny_finish")
	requireNoError(t, err)
	awaitOutcome(t, func() bool {
		var count int
		requireNoError(t, s.Database.Reader.QueryRow(`SELECT count(*) FROM jobs WHERE job_type='extract' AND status='succeeded'`).Scan(&count))
		return count == 1
	})
	runtime.Close()
	var attempts, runs int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM job_attempts").Scan(&attempts))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM extraction_runs").Scan(&runs))
	if attempts != 1 || runs != 1 {
		t.Fatal("repeated document work", attempts, runs)
	}
	if !strings.Contains(logs.String(), "WORKER_STATE_WRITE_FAILED") || strings.Contains(logs.String(), "SECRET-QA") || strings.Contains(logs.String(), "Informe") {
		t.Fatal("missing or unsafe storage-failure diagnostic")
	}
}

func TestPreviewRecoversFailedErrorWriteWithoutRepeatingConversion(t *testing.T) {
	s, admin, _, documentID := previewFixture(t)
	ctx := context.Background()
	var conversions atomic.Int32
	s.previewConvert = func(context.Context, string, string, string, string) error {
		conversions.Add(1)
		return previews.Failure("PREVIEW_CONVERSION_FAILED")
	}
	_, err := s.Database.Writer.Exec(`CREATE TRIGGER qa_deny_preview_finish BEFORE UPDATE OF status ON preview_cache WHEN OLD.status='generating' AND NEW.status='error' BEGIN SELECT RAISE(ABORT,'SECRET-QA-PREVIEW'); END`)
	requireNoError(t, err)
	runtime, err := s.Start(ctx)
	requireNoError(t, err)
	defer runtime.Close()
	_, err = s.RequestPreview(ctx, admin, documentID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	awaitOutcome(t, func() bool { return conversions.Load() == 1 })
	time.Sleep(300 * time.Millisecond)
	_, err = s.Database.Writer.Exec("DROP TRIGGER qa_deny_preview_finish")
	requireNoError(t, err)
	awaitOutcome(t, func() bool {
		preview, err := s.Preview(ctx, admin, documentID)
		requireNoError(t, err)
		return preview.State == "error"
	})
	if conversions.Load() != 1 {
		t.Fatal("converter repeated")
	}
}
