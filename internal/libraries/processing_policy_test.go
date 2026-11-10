//go:build development

package libraries

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

func TestPauseRetryBudgetsAndManualRecovery(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "managed")
	allowContentFormats(t, service, admin)
	item := uploadFormat(t, service, admin, library, "Texto.txt", []byte("Contenido consultable"))
	policy, err := service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.Paused = true
	policy.MaximumAttempts = 2
	policy.RetryDelaySeconds = 7
	policy.TimeoutSeconds = 30
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	for _, lane := range []string{"all", "scan", "content"} {
		if _, err = service.claimLane(ctx, lane); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal("pause let a claim pass", err)
		}
	}
	if _, err = service.Document(ctx, admin, item.DocumentID); err != nil {
		t.Fatal("pause blocked reads", err)
	}
	restarted := New(service.Identity)
	policy, err = restarted.ProcessingPolicy(ctx)
	requireNoError(t, err)
	if !policy.Paused {
		t.Fatal("pause lost on restart")
	}
	policy.Paused = false
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	job, err := service.claim(ctx)
	requireNoError(t, err)
	if job.MaximumAttempts != 2 || job.RetryDelaySeconds != 7 || job.TimeoutSeconds != 30 {
		t.Fatal(job)
	}
	// A running job keeps its budget; only subsequent new jobs use new settings.
	policy, err = service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.MaximumAttempts = 3
	policy.RetryDelaySeconds = 11
	policy.TimeoutSeconds = 60
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	before := time.Now()
	requireNoError(t, service.finish(ctx, job, context.DeadlineExceeded))
	saved, err := service.Job(ctx, admin, job.ID)
	requireNoError(t, err)
	available, _ := time.Parse(domain.TimeLayout, saved.AvailableAt)
	if saved.Status != "retry_wait" || saved.ErrorClass != "timeout" || available.Sub(before) < 6*time.Second || available.Sub(before) > 9*time.Second {
		t.Fatal(saved)
	}
	if _, err = service.claim(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("backoff ignored", err)
	}
	_, err = service.Database.Writer.Exec("UPDATE jobs SET available_at=? WHERE id=?", now(), job.ID)
	requireNoError(t, err)
	job, err = service.claim(ctx)
	requireNoError(t, err)
	requireNoError(t, service.finish(ctx, job, context.DeadlineExceeded))
	saved, err = service.Job(ctx, admin, job.ID)
	requireNoError(t, err)
	if saved.Status != "failed" || saved.Attempts != 2 {
		t.Fatal(saved)
	}
	successor, err := service.Retry(ctx, admin, job.ID, "Extractor reparado", domain.RequestMetadata{})
	requireNoError(t, err)
	job, err = service.claim(ctx)
	requireNoError(t, err)
	if job.ID != successor || job.Attempts != 1 || job.MaximumAttempts != 3 || job.RetryDelaySeconds != 11 || job.TimeoutSeconds != 60 {
		t.Fatal(job)
	}
	requireNoError(t, service.Extract(ctx, job, service.Identity.Config.Indexing))
	requireNoError(t, service.finish(ctx, job, nil))
	if query(t, service, admin, library, "consultable").Count != 1 {
		t.Fatal("retry failed to publish")
	}
}

func TestPauseAllowsActiveDocumentToFinishAndRemainSearchable(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "managed")
	allowContentFormats(t, service, admin)
	uploadFormat(t, service, admin, library, "Activo.txt", []byte("Consulta durante pausa"))
	active, err := service.claim(ctx)
	requireNoError(t, err)
	uploadFormat(t, service, admin, library, "Pendiente.txt", []byte("Trabajo siguiente"))
	policy, err := service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.Paused = true
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	requireNoError(t, service.Extract(ctx, active, service.Identity.Config.Indexing))
	requireNoError(t, service.finish(ctx, active, nil))
	if query(t, service, admin, library, "Consulta").Count != 1 {
		t.Fatal("pause prevented active work or search")
	}
	if _, err = service.claim(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("pending document started while paused", err)
	}
}

func TestProcessingPolicyAuthorizationAndRevision(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	policy, err := service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	if err = service.ConfigureProcessingPolicy(ctx, domain.Principal{}, policy, domain.RequestMetadata{}); err == nil {
		t.Fatal("anonymous configuration accepted")
	}
	policy.MaximumAttempts = 6
	if err = service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}); err == nil {
		t.Fatal("invalid budget accepted")
	}
	policy.MaximumAttempts = 1
	policy.Paused = true
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	if err = service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}); err == nil {
		t.Fatal("stale revision accepted")
	}
}

func TestWorkerTimeoutDoesNotBlockOtherDocuments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("controlled Unix program")
	}
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		t.Skip("PDF tools missing")
	}
	service, admin, directory := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "managed")
	allowContentFormats(t, service, admin)
	pdf, err := os.ReadFile("../../testdata/documents/native.pdf")
	requireNoError(t, err)
	failed := uploadFormat(t, service, admin, library, "Lento.pdf", pdf)
	good := uploadFormat(t, service, admin, library, "Sigue.txt", []byte("Cola continúa sin bloqueo"))
	policy, err := service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.MaximumAttempts = 1
	policy.TimeoutSeconds = 10
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	wrapper := filepath.Join(directory, "blocked-pdfinfo")
	requireNoError(t, os.WriteFile(wrapper, []byte("#!/bin/sh\nexec sleep 60\n"), 0700))
	service.Identity.Config.Indexing.PDFInfo = wrapper
	workers, err := service.Start(ctx)
	requireNoError(t, err)
	defer workers.Close()
	deadline := time.Now().Add(16 * time.Second)
	for time.Now().Before(deadline) {
		document, err := service.Document(ctx, admin, failed.DocumentID)
		requireNoError(t, err)
		if document.Processing != nil && document.Processing.Error == "PROCESS_TIMEOUT" {
			healthy, err := service.Document(ctx, admin, good.DocumentID)
			requireNoError(t, err)
			if healthy.Freshness != "current" {
				t.Fatal("other document blocked", healthy)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("blocked extractor did not time out")
}

func TestPanicIsContainedAndDoesNotExposeContent(t *testing.T) {
	err := isolatedOperation(func() error { panic("private document contents") })
	if failureCode(err) != "PROCESSING_INTERNAL_ERROR" {
		t.Fatal(err)
	}
	requireNoError(t, isolatedOperation(func() error { return nil }))
}
