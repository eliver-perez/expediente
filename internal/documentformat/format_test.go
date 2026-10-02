package documentformat

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"gestor-documental/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func officeFixture(t *testing.T, format string, extra map[string]string) []byte {
	t.Helper()
	main, kind, body := "word/document.xml", "wordprocessingml.document", `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Documento de prueba</w:t></w:r></w:p></w:body></w:document>`
	if format == "xlsx" {
		main, kind, body = "xl/workbook.xml", "spreadsheetml.sheet", `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheets/></workbook>`
	}
	entries := map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/` + main + `" ContentType="application/vnd.openxmlformats-officedocument.` + kind + `.main+xml"/></Types>`,
		"_rels/.rels":         `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="` + main + `"/></Relationships>`, main: body,
	}
	for name, value := range extra {
		entries[name] = value
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, value := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func detectBytes(t *testing.T, name string, data []byte) (Detection, error) {
	t.Helper()
	file, err := os.Create(filepath.Join(t.TempDir(), "input"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		t.Fatal(err)
	}
	return Detect(context.Background(), file, name)
}

func TestRecognizedFormatsAndRealContent(t *testing.T) {
	for _, item := range []struct {
		name   string
		data   []byte
		format string
	}{
		{"source.PDF", []byte("%PDF-1.7\n"), "pdf"},
		{"source.docx", officeFixture(t, "docx", nil), "docx"},
		{"source.xlsx", officeFixture(t, "xlsx", nil), "xlsx"},
		{"source.txt", []byte("Biblioteca, documentos y expedientes\nTexto UTF-8 con acentos."), "txt"},
		{"source.csv", []byte("Nombre;Cantidad\n\"Material; A\";2\nOtro;3"), "csv"},
	} {
		t.Run(item.format, func(t *testing.T) {
			detected, err := detectBytes(t, item.name, item.data)
			if err != nil || detected.Format != item.format || detected.Mismatch {
				t.Fatalf("%+v %v", detected, err)
			}
			format, _ := Lookup(item.format)
			if detected.MIME != format.MIME {
				t.Fatal(detected)
			}
		})
	}
}

func TestDisguisedAndUnsafeFilesRejected(t *testing.T) {
	for _, item := range []struct {
		name string
		data []byte
		code string
	}{
		{"safe.pdf", []byte("MZ executable"), "FILE_TYPE_BLOCKED"},
		{"safe.txt", []byte("\x7fELF binary"), "FILE_TYPE_BLOCKED"},
		{"safe.txt", []byte("\xef\xbb\xbf#!/bin/sh\necho unsafe"), "FILE_TYPE_BLOCKED"},
		{"safe.txt", []byte("<!doctype html><script>alert(1)</script>"), "FILE_TYPE_BLOCKED"},
		{"safe.txt", []byte("\x00\x01binary"), "FILE_TYPE_UNKNOWN"},
		{"safe.txt", []byte("\xff\xfe\x00\x00"), "FILE_TYPE_UNKNOWN"},
		{"safe.exe", []byte("Ordinary text"), "FILE_TYPE_BLOCKED"},
		{"safe.zip", officeFixture(t, "docx", nil), "FILE_TYPE_BLOCKED"},
		{"safe.pdf", []byte("ordinary text"), "INVALID_PDF"},
		{"safe.docx", []byte("%PDF-1.7"), "FILE_TYPE_MISMATCH"},
		{"safe.csv", []byte("not a CSV\njust a paragraph"), "FILE_TYPE_MISMATCH"},
		{"safe.docx", officeFixture(t, "xlsx", nil), "FILE_TYPE_MISMATCH"},
		{"safe.docx", officeFixture(t, "docx", map[string]string{"../escape.txt": "bad"}), "INVALID_OFFICE_DOCUMENT"},
		{"safe.docx", officeFixture(t, "docx", map[string]string{"word/vbaProject.bin": "macro"}), "FILE_TYPE_BLOCKED"},
		{"safe.docx", officeFixture(t, "docx", map[string]string{"word/embeddings/hidden.exe": "bad"}), "FILE_TYPE_BLOCKED"},
		{"safe.docx", officeFixture(t, "docx", map[string]string{"word/document.xml": `<!DOCTYPE x [<!ENTITY a SYSTEM "file:///etc/passwd">]><x/>`}), "INVALID_OFFICE_DOCUMENT"},
		{"safe.docx", officeFixture(t, "docx", map[string]string{"payload.txt": strings.Repeat("a", 4<<20)}), "DOCUMENT_COMPLEXITY_LIMIT"},
	} {
		t.Run(item.name+"/"+item.code, func(t *testing.T) {
			_, err := detectBytes(t, item.name, item.data)
			var failure *domain.Error
			if !errors.As(err, &failure) || failure.Code != item.code {
				t.Fatalf("want %s, got %v", item.code, err)
			}
		})
	}
}

func TestOfficeRequiresPackageRelationshipsAndMainXML(t *testing.T) {
	for _, extra := range []map[string]string{
		{"_rels/.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`},
		{"word/document.xml": `<document/>`},
		{"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><broken>`},
		{"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`},
	} {
		if _, err := detectBytes(t, "invalid.docx", officeFixture(t, "docx", extra)); err == nil {
			t.Fatal("invalid package admitted")
		}
	}
}

func TestPolicyHierarchyAndIndexingCapability(t *testing.T) {
	base := Default(256)
	base.Index = []string{"pdf", "docx", "xlsx", "txt", "csv"}
	store := []string{"pdf"}
	limit := 10
	resolved := Resolve(base, Overrides{Store: &store, MaximumFileMB: &limit})
	if err := resolved.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(resolved.Index) != 1 || resolved.Index[0] != "pdf" || resolved.MaximumIndexMB != 10 || base.MaximumIndexMB != 256 || len(base.Index) != 5 {
		t.Fatalf("inheritance mutated defaults: %+v %+v", base, resolved)
	}
	explicitIndex := []string{"docx"}
	if err := Resolve(base, Overrides{Store: &store, Index: &explicitIndex}).Validate(); err == nil {
		t.Fatal("indexing disallowed storage accepted")
	}
	base.Store = append(base.Store, "exe")
	if base.Validate() == nil {
		t.Fatal("forbidden global format accepted")
	}
	base = Default(256)
	unknown, err := detectBytes(t, "notes.data", []byte("texto conservado"))
	if err != nil {
		t.Fatal(err)
	}
	if base.Admit(unknown, 10) == nil {
		t.Fatal("unknown extension accepted by default")
	}
	base.Unknown = "store_text"
	if err = base.Admit(unknown, 10); err != nil {
		t.Fatal(err)
	}
	if base.IndexReason(unknown, 10) != "unknown_extension" {
		t.Fatal("unknown extension indexed")
	}
	base.Index = append(base.Index, "docx")
	if base.IndexReason(Detection{Format: "docx"}, 100) != "extractor_pending" {
		t.Fatal("unimplemented extractor claimed")
	}
	base.NonIndexable = "reject"
	if base.Admit(Detection{Format: "txt"}, 100) == nil {
		t.Fatal("store-only policy bypassed")
	}
	base = Default(1)
	if base.Admit(Detection{Format: "pdf"}, 2<<20) == nil {
		t.Fatal("size limit bypassed")
	}
}
