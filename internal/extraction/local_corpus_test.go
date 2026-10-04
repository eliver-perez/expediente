package extraction

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Optional local regression corpus. Documents stay outside version control and
// installers; the extractor reads only test-directory copies. No text is logged.
func TestLocalDocumentCorpus(t *testing.T) {
	directory := os.Getenv("AIBID_LOCAL_DOCUMENTS")
	if directory == "" {
		t.Skip("no local corpus requested")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	tested := 0
	for _, entry := range entries {
		format := strings.TrimPrefix(strings.ToLower(filepath.Ext(entry.Name())), ".")
		if entry.IsDir() || format != "docx" && format != "xlsx" && format != "txt" && format != "csv" {
			continue
		}
		tested++
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(directory, entry.Name())
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := extractBytes(t, format, original)
			if err != nil {
				t.Fatal(err)
			}
			total := 0
			for _, unit := range result.Units {
				total += len(strings.TrimSpace(unit.Text))
			}
			if total == 0 {
				t.Fatal("local document yielded no text")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(original, after) {
				t.Fatal("source changed", err)
			}
			t.Logf("extractor=%s/%s units=%d text_bytes=%d summary=%v warnings=%v original=unchanged", result.Extractor.ID, result.Extractor.Version, len(result.Units), total, result.Summary, result.Warnings)
		})
	}
	if tested == 0 {
		t.Fatal("empty corpus")
	}
}
