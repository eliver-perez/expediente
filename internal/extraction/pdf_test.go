package extraction

import (
	"context"
	"fmt"
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
	if _, err := Defaults().Diagnose(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := Defaults().DiagnoseSamples(context.Background(), "../../testdata/documents"); err != nil {
		t.Fatal(err)
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

func TestNativeBatchesPreservePageBoundaries(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("PDF tools unavailable")
	}
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"}
	kids := ""
	for number := 1; number <= 34; number++ {
		pageID := len(objects) + 1
		kids += fmt.Sprintf("%d 0 R ", pageID)
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", pageID+1))
		commands := ""
		if number%17 != 0 {
			commands = fmt.Sprintf("BT /F1 14 Tf 48 730 Td (Page %d native extraction with sufficient readable characters.) Tj ET", number)
		}
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(commands), commands))
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count 34 >>", kids)
	var pdf strings.Builder
	pdf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	start := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), start)
	path := filepath.Join(t.TempDir(), "batches.pdf")
	if err := os.WriteFile(path, []byte(pdf.String()), 0600); err != nil {
		t.Fatal(err)
	}
	options := Defaults()
	updates := 0
	options.Progress = func(operation string, completed, total int) {
		updates++
		if total != 34 || completed > total {
			t.Fatal("invalid native progress")
		}
	}
	pages, err := options.Prepare(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 34 || updates != 4 {
		t.Fatal("batch or page count changed", len(pages), updates)
	}
	for index, page := range pages {
		number := index + 1
		if page.Number != number {
			t.Fatal("wrong page number")
		}
		if number%17 == 0 {
			if page.Method != "pending_ocr" {
				t.Fatal("blank page lost")
			}
		} else if page.Method != "native" || !strings.Contains(page.Text, fmt.Sprintf("Page %d native", number)) {
			t.Fatal("text assigned to incorrect page", page)
		}
	}
}
