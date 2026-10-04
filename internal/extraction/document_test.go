package extraction

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
)

func officeDocument(t *testing.T, format string, changed map[string]string) []byte {
	t.Helper()
	source, err := zip.OpenReader("../../testdata/documents/sample." + format)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	entries := map[string][]byte{}
	for _, item := range source.File {
		reader, err := item.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[item.Name] = contents
	}
	for name, value := range changed {
		entries[name] = []byte(value)
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, contents := range entries {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = writer.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
func extractBytes(t *testing.T, format string, contents []byte) (Result, error) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "source."+format)
	if err := os.WriteFile(file, contents, 0600); err != nil {
		t.Fatal(err)
	}
	options := Defaults()
	options.PDFInfo = "not-an-installed-tool"
	options.Tesseract = "not-an-installed-tool"
	result, err := PrepareDocument(context.Background(), file, format, options)
	if err == nil {
		err = CompleteDocument(context.Background(), file, "spa", options, &result)
	}
	after, readErr := os.ReadFile(file)
	if readErr != nil || !bytes.Equal(after, contents) {
		t.Fatal("extractor changed original", readErr)
	}
	return result, err
}
func TestDocumentRegistryMatchesAdmissionCatalog(t *testing.T) {
	for _, format := range documentformat.Formats() {
		descriptor, ok := ExtractorFor(format.ID)
		if !ok || !format.ExtractorAvailable || descriptor.Version == "" || descriptor.ID == "" {
			t.Fatal(format, descriptor)
		}
	}
}
func TestDOCXContextAndStaticContent(t *testing.T) {
	contents := officeDocument(t, "docx", map[string]string{
		"word/document.xml":            `<w:document xmlns:w="` + wordNS + `"><w:body><w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Obra </w:t></w:r><w:r><w:t>pública</w:t></w:r><w:del><w:r><w:delText>Texto eliminado</w:delText></w:r></w:del></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>Cemento</w:t><w:tab/><w:t>gris</w:t></w:r></w:p></w:tc></w:tr></w:tbl><w:p><w:r><w:instrText>DDE cmd /c unsafe</w:instrText><w:t>Resultado guardado</w:t></w:r></w:p><w:sectPr/></w:body></w:document>`,
		"word/_rels/document.xml.rels": `<Relationships xmlns="` + packageNS + `"><Relationship Id="header1" Type="` + relationNS + `/header" Target="header1.xml"/><Relationship Id="link1" Type="` + relationNS + `/hyperlink" TargetMode="External" Target="https://example.invalid/never-fetched"/></Relationships>`,
		"word/header1.xml":             `<w:hdr xmlns:w="` + wordNS + `"><w:p><w:r><w:t>Encabezado sintético</w:t></w:r></w:p></w:hdr>`,
	})
	result, err := extractBytes(t, "docx", contents)
	if err != nil {
		t.Fatal(err)
	}
	if result.Extractor.ID != "docx-openxml" || result.Extractor.Version != "2" || len(result.Units) != 4 {
		t.Fatalf("%+v", result)
	}
	if result.Units[0].Text != "Obra pública" || result.Units[0].Context["paragraph_style"] != "Heading1" || result.Units[1].Text != "Cemento\tgris" || result.Units[1].Context["table"] != 1 || result.Units[3].Context["part"] != "word/header1.xml" {
		t.Fatalf("%+v", result.Units)
	}
	if result.Summary["paragraphs"] != 4 || result.Summary["tables"] != 1 || result.Summary["section_properties"] != 1 || len(result.Warnings) != 3 {
		t.Fatalf("%+v", result)
	}
}

func TestDOCXLogoAndBlankParagraphsDoNotHideBodyText(t *testing.T) {
	contents := officeDocument(t, "docx", map[string]string{
		"word/document.xml": `<w:document xmlns:w="` + wordNS + `"><w:body><w:p><w:r><w:drawing/></w:r></w:p><w:p/><w:p><w:r><w:t>Comisión municipal: cotización de software</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>Instalación</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`,
	})
	result, err := extractBytes(t, "docx", contents)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Units) != 2 || result.Units[0].Context["paragraph"] != 3 || !strings.Contains(result.Units[0].Text, "cotización") || result.Summary["paragraphs"] != 4 {
		t.Fatalf("%+v", result)
	}
}
func TestXLSXSharedStringsCellsAndCachedFormulas(t *testing.T) {
	contents := officeDocument(t, "xlsx", map[string]string{
		"xl/workbook.xml":            `<workbook xmlns="` + sheetNS + `" xmlns:r="` + relationNS + `"><sheets><sheet name="Presupuesto 2026" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="` + packageNS + `"><Relationship Id="rId1" Type="` + relationNS + `/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="` + relationNS + `/sharedStrings" Target="sharedStrings.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<sst xmlns="` + sheetNS + `"><si><r><t>Cemento </t></r><r><t>gris</t></r><rPh><t>ignored-phonetic</t></rPh></si></sst>`,
		"xl/worksheets/sheet1.xml":   `<worksheet xmlns="` + sheetNS + `"><sheetData><row r="14"><c r="B14" t="s"><v>0</v></c><c r="C14"><f>1+1</f><v>2</v></c><c r="D14"><f>DDE_UNSAFE_NEVER_EXECUTED()</f></c><c r="E14" t="b"><v>1</v></c><c r="F14" s="1"><v>45000</v></c></row></sheetData></worksheet>`,
	})
	result, err := extractBytes(t, "xlsx", contents)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Units) != 5 || result.Units[0].Label != "Hoja Presupuesto 2026 · celda B14" || !strings.Contains(result.Units[0].Text, "Cemento gris") || !strings.HasSuffix(result.Units[1].Text, "\n2") || strings.Contains(result.Units[2].Text, "DDE_UNSAFE") {
		t.Fatalf("%+v", result.Units)
	}
	if result.Summary["cells"] != 5 || result.Summary["formula_cells"] != 2 || result.Summary["formulas_without_cached_value"] != 1 || len(result.Warnings) != 3 {
		t.Fatalf("%+v", result)
	}
}
func TestXLSXFormattedEmptySheetKeepsSearchableName(t *testing.T) {
	contents := officeDocument(t, "xlsx", map[string]string{
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="` + sheetNS + `"><sheetData><row r="14"><c r="B14" s="1"/></row></sheetData></worksheet>`,
	})
	result, err := extractBytes(t, "xlsx", contents)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Units) != 1 || result.Units[0].Kind != "sheet" || result.Units[0].Text == "" || result.Summary["cells"] != 0 || result.Summary["sheets"] != 1 {
		t.Fatalf("empty sheet name lost or fictitious cell recorded: %+v", result)
	}
}

func utf16Bytes(text string, order binary.ByteOrder) []byte {
	data := make([]byte, 2)
	order.PutUint16(data, 0xfeff)
	for _, word := range utf16.Encode([]rune(text)) {
		var bytes [2]byte
		order.PutUint16(bytes[:], word)
		data = append(data, bytes[:]...)
	}
	return data
}
func TestUnicodeTextAndCSVPreserveContext(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		result, err := extractBytes(t, "txt", utf16Bytes("Línea pública 🙂\r\nFin", order))
		if err != nil {
			t.Fatal(err)
		}
		if result.Summary["lines"] != 2 || len(result.Units) != 1 || result.Units[0].Text != "Línea pública 🙂\nFin\n" {
			t.Fatalf("%+v", result)
		}
	}
	result, err := extractBytes(t, "csv", []byte("\xef\xbb\xbfMaterial;Cantidad\n\"Cemento;\n gris\";12\nArena;8\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Units) != 3 || result.Units[1].Context["source_line"] != 2 || result.Units[2].Context["source_line"] != 4 || !strings.Contains(result.Units[1].Text, "Cemento;\n gris") || result.Summary["delimiter"] != ";" {
		t.Fatalf("%+v", result)
	}
}
func TestDocumentRejectsMalformedPartsAndActiveInputs(t *testing.T) {
	for _, item := range []struct {
		format string
		data   []byte
	}{
		{"txt", utf16Bytes("#!/bin/sh\nnever execute", binary.LittleEndian)},
		{"txt", []byte{0xff, 0xfe, 0x00, 0xd8}},
		{"txt", []byte("binary\x00data")},
		{"csv", []byte("A,B\n1,2,3\n")},
		{"xlsx", officeDocument(t, "xlsx", map[string]string{"xl/worksheets/sheet1.xml": `<!DOCTYPE worksheet [<!ENTITY x SYSTEM "file:///private/secret">]><worksheet xmlns="` + sheetNS + `"/>`})},
		{"xlsx", officeDocument(t, "xlsx", map[string]string{"xl/worksheets/sheet1.xml": `<worksheet xmlns="` + sheetNS + `"><sheetData><row r="1"><c r="A1" t="s"><v>9999</v></c></row></sheetData></worksheet>`})},
		{"xlsx", officeDocument(t, "xlsx", map[string]string{"xl/_rels/workbook.xml.rels": `<Relationships xmlns="` + packageNS + `"><Relationship Id="rId1" Type="` + relationNS + `/worksheet" Target="../../../outside.xml"/></Relationships>`})},
	} {
		if _, err := extractBytes(t, item.format, item.data); err == nil {
			t.Fatal("unsafe input accepted", item.format)
		}
	}
	file := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(file, []byte("Texto"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PrepareDocument(ctx, file, "txt", Defaults()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err := extractBytes(t, "txt", []byte(strings.Repeat("x", (4<<20)+1)))
	var failure *domain.Error
	if !errors.As(err, &failure) {
		t.Fatal("unbounded line accepted", err)
	}
}
