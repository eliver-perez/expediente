package extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Diagnose uses the same restricted environment as extraction, including bundled languages.
func (options Options) Diagnose(ctx context.Context) ([]string, error) {
	directory, err := os.MkdirTemp("", "aibid-doctor-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	results := []string{}
	for _, program := range []string{options.PDFInfo, options.PDFText, options.PDFRender} {
		// Poppler prints -v to stderr; exercise -h and rely on the exit status.
		if _, err := options.run(ctx, 10, 32768, program, directory, "-h"); err != nil {
			return results, fmt.Errorf("%s: %w", program, err)
		}
		results = append(results, program+": OK")
	}
	languages, err := options.run(ctx, 10, 32768, options.Tesseract, directory, "--list-langs")
	if err != nil {
		return results, fmt.Errorf("%s: %w", options.Tesseract, err)
	}
	for _, required := range []string{"spa", "eng"} {
		found := false
		for _, installed := range strings.Fields(string(languages)) {
			if installed == required {
				found = true
			}
		}
		if !found {
			return results, fmt.Errorf("OCR language missing: %s", required)
		}
	}
	return append(results, options.Tesseract+": spa, eng OK"), nil
}

// DiagnoseSamples validates real extraction using the two shipped, non-sensitive PDFs.
func (options Options) DiagnoseSamples(ctx context.Context, sourceDirectory string) error {
	directory, err := os.MkdirTemp("", "aibid-pdf-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	for _, sample := range []struct{ name, method, text string }{{"native.pdf", "native", "PR-008"}, {"scanned.pdf", "ocr", "ESCANEADO"}} {
		contents, err := os.ReadFile(filepath.Join(sourceDirectory, sample.name))
		if err != nil {
			return err
		}
		path := filepath.Join(directory, sample.name)
		if err := os.WriteFile(path, contents, 0600); err != nil {
			return err
		}
		pages, err := options.Extract(ctx, path, "spa")
		if err != nil {
			return fmt.Errorf("%s: %w", sample.name, err)
		}
		if len(pages) != 1 || pages[0].Method != sample.method || !strings.Contains(pages[0].Text, sample.text) {
			return fmt.Errorf("%s: unexpected extraction result", sample.name)
		}
	}
	return nil
}
