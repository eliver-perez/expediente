//go:build development

package libraries

import (
	"bytes"
	"context"
	"fmt"
	"gestor-documental/internal/domain"
	"testing"
)

func TestUploadHistoryCountsPaginationAndIsolation(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	lib := managedLibrary(t, s, p, "managed")
	allowContentFormats(t, s, p)
	empty, err := s.CreateBatch(ctx, p, lib, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	page, err := s.BatchItems(ctx, p, empty, "")
	requireNoError(t, err)
	if len(page.Items) != 0 || page.Next != "" {
		t.Fatal(page)
	}
	batch, err := s.CreateBatch(ctx, p, lib, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	for n := 0; n < 26; n++ {
		_, err = s.Upload(ctx, p, batch, domain.NewID(), fmt.Sprintf("archivo-%d.txt", n), bytes.NewBufferString("Documento de prueba para el historial"), nil, domain.RequestMetadata{})
		requireNoError(t, err)
	}
	// Interrupted uploads have no document/version/job; their diagnostics must still render.
	_, err = s.Database.Writer.Exec(`INSERT INTO upload_items(id,batch_id,library_id,client_file_id,private_temporary_locator,original_filename,status,size_bytes,error_code,created_at) VALUES(?,?,?,?,'','interrumpido.pdf','failed',0,'PROCESS_RESTARTED',?)`, domain.NewID(), empty, lib, domain.NewID(), now())
	requireNoError(t, err)
	failed, err := s.BatchItems(ctx, p, empty, "")
	requireNoError(t, err)
	if len(failed.Items) != 1 || failed.Items[0].CanView || failed.Items[0].Error != "PROCESS_RESTARTED" {
		t.Fatal(failed)
	}
	for n := 0; n < 49; n++ {
		_, err = s.CreateBatch(ctx, p, lib, domain.NewID(), domain.RequestMetadata{})
		requireNoError(t, err)
	}
	batches, err := s.Batches(ctx, p, lib, "")
	requireNoError(t, err)
	if len(batches.Items) != 50 || batches.Next == "" {
		t.Fatal("unbounded history", batches)
	}
	next, err := s.Batches(ctx, p, lib, batches.Next)
	requireNoError(t, err)
	if len(next.Items) != 1 || next.Next != "" {
		t.Fatal(next)
	}
	found := false
	for _, b := range append(batches.Items, next.Items...) {
		if b.ID == batch {
			found = true
			if b.Count != 26 {
				t.Fatal(b)
			}
		}
	}
	if !found {
		t.Fatal("missing batch")
	}
	page, err = s.BatchItems(ctx, p, batch, "")
	requireNoError(t, err)
	if len(page.Items) != 25 || page.Next == "" {
		t.Fatal(page)
	}
	for _, item := range page.Items {
		if item.Format != "txt" || item.Status != "staged" || !item.CanView || item.Size <= 0 || item.ProcessingState != "pending" {
			t.Fatal(item)
		}
	}
	last, err := s.BatchItems(ctx, p, batch, page.Next)
	requireNoError(t, err)
	if len(last.Items) != 1 || last.Next != "" {
		t.Fatal(last)
	}
	if _, err = s.BatchItems(ctx, p, empty, page.Next); err == nil {
		t.Fatal("cursor crossed batches")
	}
	if _, err = s.BatchItems(ctx, domain.Principal{}, batch, ""); err == nil {
		t.Fatal("anonymous history")
	}
	for n := 0; n < 26; n++ {
		job, err := s.claim(ctx)
		requireNoError(t, err)
		requireNoError(t, s.Extract(ctx, job, s.Identity.Config.Indexing))
		requireNoError(t, s.finish(ctx, job, nil))
	}
	page, err = s.BatchItems(ctx, p, batch, "")
	requireNoError(t, err)
	if page.Items[0].ProcessingState != "completed" {
		t.Fatal(page.Items[0])
	}
}
func TestUploadHistoryPrivateBatches(t *testing.T) {
	f := workflowSetup(t, false)
	s := f.service
	ctx := context.Background()
	batch, err := s.CreateBatch(ctx, f.author, f.library, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	if _, err = s.BatchItems(ctx, member(t, s, f.author, f.library, "history-reader", "library_reader"), batch, ""); err == nil {
		t.Fatal("private batch visible to reader")
	}
}
