package previews

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSanitizeRemovesActiveAndExternalContentPreservingXML(t *testing.T) {
	ctx := context.Background()
	relationships := []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="web" TargetMode="External" Target="https://example.invalid/private"/><Relationship Id="image" Target="media/image.png"/></Relationships>`)
	cleaned, err := sanitizeXML(ctx, relationships, "word/_rels/document.xml.rels")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(cleaned, []byte("example.invalid")) || !bytes.Contains(cleaned, []byte("media/image.png")) {
		t.Fatal(string(cleaned))
	}
	document := []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:instrText>DDE malicious</w:instrText><w:t>Visible result</w:t><w:fldSimple w:instr="INCLUDETEXT malicious"><w:r><w:t>Cached field</w:t></w:r></w:fldSimple></w:p></w:document>`)
	cleaned, err = sanitizeXML(ctx, document, "word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(cleaned, []byte("DDE")) || bytes.Contains(cleaned, []byte("INCLUDETEXT")) || !bytes.Contains(cleaned, []byte("<w:t>Cached field</w:t>")) || !bytes.Contains(cleaned, []byte("<w:t>Visible result</w:t>")) {
		t.Fatal(string(cleaned))
	}
	cleaned, err = sanitizeXML(ctx, []byte(`<worksheet><c><f>WEBSERVICE("http://example.invalid")</f><v>123</v></c></worksheet>`), "xl/worksheets/sheet1.xml")
	if err != nil || bytes.Contains(cleaned, []byte("WEBSERVICE")) || !bytes.Contains(cleaned, []byte("<v>123</v>")) {
		t.Fatal(string(cleaned), err)
	}
	for _, bad := range []string{`<!DOCTYPE x><x/>`, `<Relationships><Relationship Target="file:///etc/passwd"/></Relationships>`, `<Relationships><Relationship Target="../../../../secret"/></Relationships>`} {
		if _, err = sanitizeXML(ctx, []byte(bad), "word/_rels/document.xml.rels"); err == nil {
			t.Fatal("unsafe XML accepted")
		}
	}
}
func TestSanitizeOfficeOriginalUnchanged(t *testing.T) {
	for _, format := range []string{"docx", "xlsx"} {
		source := filepath.Join("../../testdata/documents", "sample."+format)
		before, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(t.TempDir(), "safe."+format)
		if err = SanitizeOffice(context.Background(), source, destination, format); err != nil {
			t.Fatal(err)
		}
		after, _ := os.ReadFile(source)
		if !bytes.Equal(before, after) {
			t.Fatal("original modified")
		}
		archive, err := zip.OpenReader(destination)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range archive.File {
			if !strings.HasSuffix(entry.Name, ".xml") {
				continue
			}
			reader, _ := entry.Open()
			data, _ := io.ReadAll(reader)
			reader.Close()
			if bytes.Contains(data, []byte("<f>")) {
				t.Fatal("formula survived")
			}
		}
		archive.Close()
	}
}
func TestConversionDeadlineKillsProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix controlled program")
	}
	directory := t.TempDir()
	program := filepath.Join(directory, "soffice")
	if err := os.WriteFile(program, []byte("#!/bin/sh\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(directory, "source.docx")
	data, _ := os.ReadFile("../../testdata/documents/sample.docx")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	before := time.Now()
	err := Convert(ctx, source, "docx", filepath.Join(directory, "preview.pdf"), program)
	if err == nil || time.Since(before) > 3*time.Second {
		t.Fatal("unbounded conversion", err)
	}
}

// Opt-in real converter test. Documents remain private and only counts are logged.
func TestLocalLibreOffice(t *testing.T) {
	program := os.Getenv("AIBID_TEST_LIBREOFFICE")
	if program == "" {
		t.Skip("no local converter requested")
	}
	directory := os.Getenv("AIBID_LOCAL_DOCUMENTS")
	synthetic := directory == ""
	if directory == "" {
		directory = "../../testdata/documents"
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		format := strings.TrimPrefix(strings.ToLower(filepath.Ext(entry.Name())), ".")
		if format != "docx" && format != "xlsx" {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join(directory, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			temporary := t.TempDir()
			source := filepath.Join(temporary, "source."+format)
			if err = os.WriteFile(source, original, 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			output := filepath.Join(temporary, "preview.pdf")
			if err = Convert(ctx, source, format, output, program); err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(output)
			if !bytes.HasPrefix(data, []byte("%PDF-")) {
				t.Fatal("not PDF")
			}
			if _, err = exec.LookPath("pdfinfo"); err == nil {
				if _, err = exec.Command("pdfinfo", output).Output(); err != nil {
					t.Fatal("invalid PDF", err)
				}
			}
			if _, err = exec.LookPath("pdftotext"); err == nil {
				text, extractError := exec.Command("pdftotext", output, "-").Output()
				if extractError != nil || len(bytes.TrimSpace(text)) == 0 {
					t.Fatal("preview has no readable document text", extractError)
				}
				// The small XLSX fixtures contain just two cells; Calc can clip the
				// label to its column width. Check their actual content, not an
				// arbitrary minimum which rejects a valid, short spreadsheet.
				if synthetic && format == "xlsx" && (!bytes.Contains(text, []byte("Material")) || !bytes.Contains(text, []byte("12"))) {
					t.Fatal("preview lost the fixture label or numeric cell")
				}
				t.Logf("readable preview text: %d bytes", len(text))
			}
			after, _ := os.ReadFile(filepath.Join(directory, entry.Name()))
			if sha256.Sum256(original) != sha256.Sum256(after) {
				t.Fatal("original changed")
			}
			t.Logf("PDF generated: %d bytes; original unchanged", len(data))
		})
	}
}

func TestConversionStopsGrowingOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix controlled program")
	}
	directory := t.TempDir()
	program := filepath.Join(directory, "soffice")
	if err := os.WriteFile(program, []byte("#!/bin/sh\ndd if=/dev/zero of=safe.pdf bs=1 count=0 seek=67108865 2>/dev/null\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(directory, "source.docx")
	data, err := os.ReadFile("../../testdata/documents/sample.docx")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	before := time.Now()
	err = Convert(ctx, source, "docx", filepath.Join(directory, "preview.pdf"), program)
	if err == nil || err.Error() != "PREVIEW_SIZE_LIMIT" || time.Since(before) > 3*time.Second {
		t.Fatal("growing output not stopped", err)
	}
}
