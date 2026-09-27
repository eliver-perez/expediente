//go:build development

package libraries

import (
	"context"
	"fmt"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/extraction"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"
)

func TestNativeAndOCRWorkersBoundedAndRestartable(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("controlled shell wrapper used only on Unix; native Windows validation separate")
	}
	tesseract, err := exec.LookPath("tesseract")
	if err != nil {
		t.Skip("OCR tools unavailable")
	}
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Concurrency")
	release := filepath.Join(directory, "release-ocr")
	wrapper := filepath.Join(directory, "ocr-wrapper")
	script := fmt.Sprintf("#!/bin/sh\nwhile [ ! -f %q ]; do sleep 0.05; done\nexec %q \"$@\"\n", release, tesseract)
	requireNoError(t, os.WriteFile(wrapper, []byte(script), 0700))
	service.Identity.Config.Indexing.Tesseract = wrapper
	requireNoError(t, service.ConfigureProcessing(ctx, principal, extraction.Concurrency{Mode: "manual", Native: 2, OCR: 1, Total: 3}, 0, domain.RequestMetadata{}))
	source := filepath.Join(directory, "source")
	for i := 0; i < 6; i++ {
		fixture := "native.pdf"
		if i < 2 {
			fixture = "scanned.pdf"
		}
		copyFixture(t, filepath.Join(source, fmt.Sprintf("%02d.pdf", i)), fixture)
	}
	root := addRoot(t, service, principal, library, source)
	scan(t, service, root)
	runtime, err := service.Start(ctx)
	requireNoError(t, err)
	closed := false
	t.Cleanup(func() {
		if !closed {
			runtime.Close()
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	simultaneous := false
	for time.Now().Before(deadline) {
		report, err := service.Processing(ctx, principal, library)
		requireNoError(t, err)
		if report.Performance.OCRActive > 1 || report.Performance.NativeActive+report.Performance.OCRActive > 3 {
			t.Fatal("limits exceeded", report.Performance)
		}
		if report.Stats.Native == 4 && report.Performance.OCRActive == 1 && report.Performance.WaitingOCR >= 1 {
			simultaneous = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !simultaneous {
		t.Fatal("native documents did not complete while OCR was blocked")
	}
	runtime.Close()
	closed = true
	requireNoError(t, os.WriteFile(release, []byte("ready"), 0600))
	restarted := New(service.Identity)
	configuration, err := restarted.ProcessingConfiguration(ctx)
	requireNoError(t, err)
	if configuration.Effective.Total != 3 || configuration.Effective.Native != 2 {
		t.Fatal("manual settings lost on restart")
	}
	runtime, err = restarted.Start(ctx)
	requireNoError(t, err)
	closed = false
	deadline = time.Now().Add(20 * time.Second)
	finished := false
	for time.Now().Before(deadline) {
		report, err := restarted.Processing(ctx, principal, library)
		requireNoError(t, err)
		if report.Stats.Processed == 6 && report.Stats.Native == 4 && report.Stats.OCR == 2 {
			finished = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !finished {
		t.Fatal("OCR queue lost on restart")
	}
	runtime.Close()
	closed = true
	var jobs, files, maximum int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*),count(DISTINCT physical_file_id),max(attempt_count) FROM jobs WHERE job_type='extract'").Scan(&jobs, &files, &maximum))
	if jobs != 6 || files != 6 || maximum > 2 {
		t.Fatal("duplicate jobs or attempts", jobs, files, maximum)
	}
	var integrity string
	requireNoError(t, service.Database.Reader.QueryRow("PRAGMA integrity_check").Scan(&integrity))
	if integrity != "ok" {
		t.Fatal(integrity)
	}
}
