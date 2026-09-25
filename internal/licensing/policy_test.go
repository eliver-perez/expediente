package licensing

import (
	"context"
	"gestor-documental/internal/storage"
	"path/filepath"
	"testing"
)

func TestSecurityRecoveryAlwaysAvailableAndDocumentGateRespectsBuild(t *testing.T) {
	ctx := context.Background()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := storage.Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	client, err := New(database, directory, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []Operation{SecurityAdministration, BackupExport} {
		if err = client.Check(ctx, op); err != nil {
			t.Fatal(err)
		}
	}
	for _, op := range []Operation{ReadDocuments, WriteDocuments} {
		if allowed := client.Check(ctx, op) == nil; allowed != DevelopmentEnabled() {
			t.Fatal("development policy leaked across build boundary")
		}
	}
}
