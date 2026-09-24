package extraction

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealNativeAndSpanishOCR(t *testing.T) {
	for _, program := range []string{"pdfinfo", "pdftotext", "pdftoppm", "tesseract"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Skip("PDF/OCR tools not installed")
		}
	}
	for _, fixture := range []struct{ Name, Method, Contains string }{{"native.pdf", "native", "PR-008"}, {"scanned.pdf", "ocr", "ESCANEADO"}} {
		t.Run(fixture.Name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join("../../testdata/documents", fixture.Name))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "source.pdf")
			if err = os.WriteFile(path, contents, 0600); err != nil {
				t.Fatal(err)
			}
			pages, err := Defaults().Extract(context.Background(), path, "spa")
			if err != nil {
				t.Fatal(err)
			}
			if len(pages) != 1 || pages[0].Method != fixture.Method || !strings.Contains(pages[0].Text, fixture.Contains) {
				t.Fatalf("unexpected extraction: %+v", pages)
			}
		})
	}
}
