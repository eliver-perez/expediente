package storage

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteWindowsDriveIsNotURIAuthority(t *testing.T) {
	// This reproduces the reported Windows failure even when running on macOS/Linux.
	broken := url.URL{Scheme: "file", Path: "C:/ProgramData/AIBID-Test/state/documental.db"}
	connection, err := sql.Open("sqlite", broken.String())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.Ping(); err == nil || !strings.Contains(err.Error(), "invalid uri authority: C:") {
		t.Fatalf("original failure not reproduced: %v", err)
	}
	for _, path := range []string{"C:/ProgramData/AIBID-Test/state/documental.db", "d:/Biblioteca ñ/100% #1/documental.db", "/var/lib/AIBID #1/what?.db"} {
		uri := sqliteFileURL(path)
		uri.RawQuery = url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)", "foreign_keys(1)"}}.Encode()
		parsed, err := url.Parse(uri.String())
		if err != nil {
			t.Fatal(err)
		}
		expected := path
		if path[0] != '/' {
			expected = "/" + path
		}
		if parsed.Host != "" || parsed.Path != expected || parsed.Fragment != "" || parsed.Query().Get("mode") != "ro" || len(parsed.Query()["_pragma"]) != 2 {
			t.Fatalf("path/options lost in %s", uri.String())
		}
	}
}

func TestSQLiteSpecialFilenameRoundTripAndReadOnlyConnection(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(parent, "Biblioteca ñ 100% #1")
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Writer.Exec("CREATE TABLE uri_regression(value TEXT); INSERT INTO uri_regression VALUES ('preserved')"); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Reader.Exec("INSERT INTO uri_regression VALUES ('should not write')"); err == nil {
		database.Close()
		t.Fatal("reader became writable")
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "documental.db")); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var value, journal string
	if err := reopened.Reader.QueryRow("SELECT value FROM uri_regression").Scan(&value); err != nil || value != "preserved" {
		t.Fatalf("data after reopen: %s %v", value, err)
	}
	if err := reopened.Reader.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
		t.Fatalf("WAL: %s %v", journal, err)
	}
}
