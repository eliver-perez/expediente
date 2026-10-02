package documentformat

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"io"
	"os"
	"path"
	"strings"
)

type cancellableReader struct {
	context context.Context
	reader  io.Reader
}

func (reader cancellableReader) Read(buffer []byte) (int, error) {
	if err := reader.context.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

// Inspect only bounded package/XML structures, without unpacking files or
// resolving relationships. Macro packages and embedded executable payloads are
// intentionally outside the initial format allowlist.
func detectOffice(ctx context.Context, file *os.File, size int64) (string, error) {
	invalid := func() (string, error) { return "", Failure("INVALID_OFFICE_DOCUMENT") }
	footer := make([]byte, min(size, 65557))
	if _, err := file.ReadAt(footer, size-int64(len(footer))); err != nil {
		return invalid()
	}
	end := bytes.LastIndex(footer, []byte("PK\x05\x06"))
	if end < 0 || len(footer)-end < 22 {
		return invalid()
	}
	centralSize := binary.LittleEndian.Uint32(footer[end+12:])
	centralOffset := binary.LittleEndian.Uint32(footer[end+16:])
	endOffset := size - int64(len(footer)) + int64(end)
	if binary.LittleEndian.Uint16(footer[end+4:]) != 0 || binary.LittleEndian.Uint16(footer[end+6:]) != 0 || binary.LittleEndian.Uint16(footer[end+8:]) != binary.LittleEndian.Uint16(footer[end+10:]) || int64(centralOffset)+int64(centralSize) != endOffset || int(binary.LittleEndian.Uint16(footer[end+20:])) != len(footer)-end-22 {
		return invalid() // No split, ZIP64, appended or ambiguous directories.
	}
	// Bound the central directory before archive/zip allocates its entry list.
	if binary.LittleEndian.Uint16(footer[end+10:]) > 10000 || binary.LittleEndian.Uint32(footer[end+12:]) > 4<<20 {
		return "", Failure("DOCUMENT_COMPLEXITY_LIMIT")
	}
	archive, err := zip.NewReader(file, size)
	if err != nil {
		return invalid()
	}
	if len(archive.File) > 10000 {
		return "", Failure("DOCUMENT_COMPLEXITY_LIMIT")
	}
	entries := map[string]*zip.File{}
	var expanded uint64
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		name := strings.ToLower(entry.Name)
		clean := strings.TrimSuffix(entry.Name, "/")
		if path.IsAbs(clean) || path.Clean(clean) != clean || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(clean, "\\:\x00") || entry.Mode()&os.ModeSymlink != 0 || entries[name] != nil || entry.Flags&1 != 0 {
			return invalid()
		}
		if strings.Contains(name, "vbaproject") || strings.Contains(name, "activex/") || strings.Contains(name, "embeddings/") || !entry.FileInfo().IsDir() && BlockedExtension(entry.Name) {
			return "", Failure("FILE_TYPE_BLOCKED")
		}
		if entry.UncompressedSize64 > 64<<20 || expanded > 512<<20 || entry.UncompressedSize64 > 200*max(entry.CompressedSize64, 1) {
			return "", Failure("DOCUMENT_COMPLEXITY_LIMIT")
		}
		expanded += entry.UncompressedSize64
		entries[name] = entry
	}
	if expanded > 512<<20 {
		return "", Failure("DOCUMENT_COMPLEXITY_LIMIT")
	}
	// Package identifiers are case-sensitive even though duplicates were checked
	// case-insensitively to avoid ambiguous Office interpretations.
	part := func(name string) *zip.File {
		entry := entries[name]
		if entry != nil && entry.Name == name {
			return entry
		}
		return nil
	}
	typesEntry := entries["[content_types].xml"]
	if typesEntry == nil {
		return "", Failure("FILE_TYPE_BLOCKED")
	}
	if typesEntry == nil || typesEntry.Name != "[Content_Types].xml" || part("_rels/.rels") == nil {
		return invalid()
	}
	mainPart, format := "", ""
	err = inspectXML(ctx, typesEntry, "Types", "http://schemas.openxmlformats.org/package/2006/content-types", func(element xml.StartElement) error {
		if element.Name.Local != "Override" && element.Name.Local != "Default" {
			return nil
		}
		var contentType, partName string
		for _, attr := range element.Attr {
			if attr.Name.Local == "ContentType" {
				contentType = attr.Value
			}
			if attr.Name.Local == "PartName" {
				partName = attr.Value
			}
		}
		lower := strings.ToLower(contentType)
		if strings.Contains(lower, "macroenabled") || strings.Contains(lower, "vba") || strings.Contains(lower, "activex") || strings.Contains(lower, "oleobject") {
			return Failure("FILE_TYPE_BLOCKED")
		}
		candidate := ""
		if contentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml" && partName == "/word/document.xml" {
			candidate = "docx"
		}
		if contentType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml" && partName == "/xl/workbook.xml" {
			candidate = "xlsx"
		}
		if candidate != "" {
			if format != "" {
				return Failure("INVALID_OFFICE_DOCUMENT")
			}
			format = candidate
			mainPart = strings.TrimPrefix(partName, "/")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if format == "" || part(mainPart) == nil {
		return invalid()
	}
	related := 0
	err = inspectXML(ctx, part("_rels/.rels"), "Relationships", "http://schemas.openxmlformats.org/package/2006/relationships", func(element xml.StartElement) error {
		if element.Name.Local != "Relationship" {
			return nil
		}
		var target, kind, mode string
		for _, attr := range element.Attr {
			switch attr.Name.Local {
			case "Target":
				target = attr.Value
			case "Type":
				kind = attr.Value
			case "TargetMode":
				mode = attr.Value
			}
		}
		if kind == "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" || kind == "http://purl.oclc.org/ooxml/officeDocument/relationships/officeDocument" {
			if strings.TrimPrefix(target, "/") != mainPart || mode == "External" {
				return Failure("INVALID_OFFICE_DOCUMENT")
			}
			related++
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if related != 1 {
		return invalid()
	}
	root, namespace := "document", "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	if format == "xlsx" {
		root, namespace = "workbook", "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	}
	err = inspectXML(ctx, part(mainPart), root, namespace, nil)
	if err != nil {
		return "", err
	}
	return format, nil
}

func inspectXML(ctx context.Context, entry *zip.File, root, namespace string, visit func(xml.StartElement) error) error {
	if entry.UncompressedSize64 > 32<<20 {
		return Failure("DOCUMENT_COMPLEXITY_LIMIT")
	}
	stream, err := entry.Open()
	if err != nil {
		return Failure("INVALID_OFFICE_DOCUMENT")
	}
	defer stream.Close()
	decoder := xml.NewDecoder(cancellableReader{ctx, io.LimitReader(stream, (32<<20)+1)})
	depth, roots := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return Failure("INVALID_OFFICE_DOCUMENT")
		}
		switch element := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
				strict := strings.Replace(namespace, "http://schemas.openxmlformats.org/", "http://purl.oclc.org/ooxml/", 1)
				strict = strings.Replace(strict, "/2006/", "/", 1)
				if roots != 1 || element.Name.Local != root || element.Name.Space != namespace && element.Name.Space != strict {
					return Failure("INVALID_OFFICE_DOCUMENT")
				}
			}
			depth++
			if depth > 128 {
				return Failure("DOCUMENT_COMPLEXITY_LIMIT")
			}
			if visit != nil {
				if err := visit(element); err != nil {
					return err
				}
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			return Failure("INVALID_OFFICE_DOCUMENT") // No DTD/entities.
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(element)) > 0 {
				return Failure("INVALID_OFFICE_DOCUMENT")
			}
		}
	}
	if roots != 1 || depth != 0 {
		return Failure("INVALID_OFFICE_DOCUMENT")
	}
	return nil
}
