package extraction

import (
	"context"
	"encoding/xml"
	"net/url"
	"path"
	"strings"

	"gestor-documental/internal/documentformat"
)

const wordNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
const sheetNS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
const relationNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
const packageNS = "http://schemas.openxmlformats.org/package/2006/relationships"

func sameNamespace(actual, expected string) bool {
	strict := strings.Replace(strings.Replace(expected, "http://schemas.openxmlformats.org/", "http://purl.oclc.org/ooxml/", 1), "/2006/", "/", 1)
	return actual == expected || actual == strict
}
func attribute(element xml.StartElement, name string) string {
	for _, attr := range element.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

type relationship struct {
	ID, Kind, Target string
	External         bool
}

// Relationships resolve only inside the already validated ZIP. URL schemes,
// traversal outside its root and ambiguous targets never become filesystem or
// network operations. External relationships are reported but never opened.
func officeRelationships(ctx context.Context, archive *documentformat.OfficePackage, source string) ([]relationship, error) {
	filename := path.Join(path.Dir(source), "_rels", path.Base(source)+".rels")
	result := []relationship{}
	if !archive.Has(filename) {
		return result, nil
	}
	seen := map[string]bool{}
	err := archive.XML(ctx, filename, "Relationships", packageNS, func(token xml.Token) error {
		element, ok := token.(xml.StartElement)
		if !ok || !sameNamespace(element.Name.Space, packageNS) || element.Name.Local != "Relationship" {
			return nil
		}
		item := relationship{ID: attribute(element, "Id"), Kind: attribute(element, "Type"), Target: attribute(element, "Target"), External: attribute(element, "TargetMode") == "External"}
		if item.ID == "" || seen[item.ID] {
			return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
		}
		seen[item.ID] = true
		if !item.External {
			parsed, err := url.Parse(item.Target)
			if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(item.Target, "\\\x00") {
				return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
			}
			target := parsed.Path
			if strings.HasPrefix(target, "/") {
				target = strings.TrimPrefix(target, "/")
			} else {
				target = path.Join(path.Dir(source), target)
			}
			target = path.Clean(target)
			if target == ".." || strings.HasPrefix(target, "../") || strings.ContainsAny(target, "\\:\x00") {
				return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
			}
			item.Target = target
		}
		result = append(result, item)
		return nil
	})
	return result, err
}
func relationKind(value, name string) bool {
	return value == relationNS+"/"+name || value == "http://purl.oclc.org/ooxml/officeDocument/relationships/"+name
}
