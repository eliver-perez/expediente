package previews

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"gestor-documental/internal/documentformat"
)

// SanitizeOffice rewrites only a disposable archive. Cached formula results and
// visible field results survive, but formulas, instructions, external relationships
// and embedded active references never reach LibreOffice. Raw XML token spans
// preserve namespace prefixes used in OOXML attribute values.
func SanitizeOffice(ctx context.Context, source, destination, format string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = documentformat.OpenOffice(ctx, file, format); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer output.Close()
	writer := zip.NewWriter(output)
	for _, entry := range archive.File {
		if err = ctx.Err(); err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return err
		}
		data, readError := io.ReadAll(io.LimitReader(reader, (32<<20)+1))
		reader.Close()
		if readError != nil {
			return readError
		}
		if len(data) > 32<<20 {
			return Failure("PREVIEW_SIZE_LIMIT")
		}
		if strings.HasSuffix(entry.Name, ".xml") || strings.HasSuffix(entry.Name, ".rels") {
			data, err = sanitizeXML(ctx, data, entry.Name)
			if err != nil {
				return err
			}
		}
		part, err := writer.Create(entry.Name)
		if err != nil {
			return err
		}
		if _, err = part.Write(data); err != nil {
			return err
		}
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return output.Close()
}
func sanitizeXML(ctx context.Context, data []byte, filename string) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	depth, skip := 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		before := decoder.InputOffset()
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, Failure("INVALID_OFFICE_DOCUMENT")
		}
		after := decoder.InputOffset()
		omit := skip > 0
		switch element := token.(type) {
		case xml.Directive:
			return nil, Failure("INVALID_OFFICE_DOCUMENT")
		case xml.StartElement:
			depth++
			if depth > 128 {
				return nil, Failure("PREVIEW_SIZE_LIMIT")
			}
			if skip > 0 {
				skip++
				break
			}
			switch element.Name.Local {
			case "f", "instrText", "fldChar", "fldData", "ddeLink", "externalLink", "externalReferences", "oleObject", "controls", "altChunk", "connection", "queryTable", "externalData", "mailMerge":
				omit = true
			}
			if strings.HasSuffix(filename, ".rels") && element.Name.Local == "Relationship" {
				target, mode := "", ""
				for _, attribute := range element.Attr {
					if attribute.Name.Local == "Target" {
						target = attribute.Value
					}
					if attribute.Name.Local == "TargetMode" {
						mode = attribute.Value
					}
				}
				parsed, parseError := url.Parse(target)
				if mode == "External" {
					omit = true
				} else if parseError != nil || parsed.IsAbs() || parsed.Host != "" || strings.ContainsAny(target, "\\\x00:") {
					return nil, Failure("INVALID_OFFICE_DOCUMENT")
				} else {
					// A relationship's base is the directory containing its source part.
					base := path.Dir(path.Dir(filename))
					resolved := path.Clean(path.Join(base, parsed.Path))
					if strings.HasPrefix(resolved, "../") || resolved == ".." {
						return nil, Failure("INVALID_OFFICE_DOCUMENT")
					}
				}
			}
			for _, attribute := range element.Attr {
				if (attribute.Name.Local == "src" || attribute.Name.Local == "href") && !strings.HasPrefix(attribute.Value, "#") {
					omit = true
				}
			}
			if omit {
				skip = 1
			}
			// Keep the cached visible result, without the executable instruction
			// stored on the simple-field wrapper.
			if element.Name.Local == "fldSimple" {
				omit = true
			}
		case xml.EndElement:
			depth--
			if element.Name.Local == "fldSimple" {
				omit = true
			}
			if skip > 0 {
				skip--
			}
		}
		if !omit {
			output.Write(data[before:after])
		}
	}
	if depth != 0 {
		return nil, Failure("INVALID_OFFICE_DOCUMENT")
	}
	return output.Bytes(), nil
}
