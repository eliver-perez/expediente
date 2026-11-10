package extraction

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"gestor-documental/internal/documentformat"
)

type wordParagraph struct {
	text                              strings.Builder
	number, section, table, row, cell int
	style                             string
}
type wordTable struct{ number, row, cell int }

func extractDOCX(ctx context.Context, file *os.File, options Options, result *Result) error {
	archive, err := documentformat.OpenOffice(ctx, file, "docx")
	if err != nil {
		return err
	}
	relations, err := officeRelationships(ctx, archive, "word/document.xml")
	if err != nil {
		return err
	}
	type part struct{ name, root, label string }
	parts := []part{{"word/document.xml", "document", "Documento"}}
	seen := map[string]bool{"word/document.xml": true}
	for _, rel := range relations {
		if rel.External {
			result.warn("EXTERNAL_REFERENCES_IGNORED")
			continue
		}
		for _, candidate := range []struct{ kind, root, label string }{{"header", "hdr", "Encabezado"}, {"footer", "ftr", "Pie de página"}, {"footnotes", "footnotes", "Notas al pie"}, {"endnotes", "endnotes", "Notas finales"}} {
			if relationKind(rel.Kind, candidate.kind) && !seen[rel.Target] {
				parts = append(parts, part{rel.Target, candidate.root, candidate.label})
				seen[rel.Target] = true
			}
		}
	}
	paragraphs, tables, sections := 0, 0, 0
	for _, part := range parts {
		stack := []*wordParagraph{}
		tableStack := []wordTable{}
		section, textDepth, deleted := 1, 0, 0
		err := archive.XML(ctx, part.name, part.root, wordNS, func(token xml.Token) error {
			switch element := token.(type) {
			case xml.StartElement:
				if !sameNamespace(element.Name.Space, wordNS) {
					return nil
				}
				if element.Name.Local == "del" {
					deleted++
					result.warn("TRACKED_DELETIONS_IGNORED")
				}
				if deleted > 0 {
					return nil
				}
				switch element.Name.Local {
				case "tbl":
					tables++
					tableStack = append(tableStack, wordTable{number: tables})
				case "tr":
					if len(tableStack) > 0 {
						current := &tableStack[len(tableStack)-1]
						current.row++
						current.cell = 0
					}
				case "tc":
					if len(tableStack) > 0 {
						tableStack[len(tableStack)-1].cell++
					}
				case "p":
					paragraphs++
					current := &wordParagraph{number: paragraphs, section: section}
					if len(tableStack) > 0 {
						table := tableStack[len(tableStack)-1]
						current.table, current.row, current.cell = table.number, table.row, table.cell
					}
					stack = append(stack, current)
				case "pStyle":
					if len(stack) > 0 {
						stack[len(stack)-1].style = attribute(element, "val")
					}
				case "t":
					textDepth++
				case "tab", "br", "cr":
					if len(stack) > 0 {
						if element.Name.Local == "tab" {
							stack[len(stack)-1].text.WriteByte('\t')
						} else {
							stack[len(stack)-1].text.WriteByte('\n')
						}
					}
				case "instrText":
					result.warn("FIELD_INSTRUCTIONS_IGNORED")
				case "altChunk":
					result.warn("EMBEDDED_TEXT_PART_IGNORED")
				}
			case xml.CharData:
				if deleted == 0 && textDepth > 0 && len(stack) > 0 {
					current := stack[len(stack)-1]
					current.text.Write(element)
					return textLimit(&current.text)
				}
			case xml.EndElement:
				if !sameNamespace(element.Name.Space, wordNS) {
					return nil
				}
				if element.Name.Local == "del" {
					deleted--
					return nil
				}
				if deleted > 0 {
					return nil
				}
				switch element.Name.Local {
				case "t":
					textDepth--
				case "tbl":
					if len(tableStack) > 0 {
						tableStack = tableStack[:len(tableStack)-1]
					}
				case "sectPr":
					if part.root == "document" {
						sections++
						section++
					}
				case "p":
					if len(stack) == 0 {
						return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
					}
					current := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					label := fmt.Sprintf("%s · párrafo %d", part.label, current.number)
					details := map[string]any{"part": part.name, "paragraph": current.number, "section": current.section}
					if current.table > 0 {
						label += fmt.Sprintf(" · tabla %d, fila %d, celda %d", current.table, current.row, current.cell)
						details["table"], details["row"], details["cell"] = current.table, current.row, current.cell
					}
					if current.style != "" {
						details["paragraph_style"] = current.style
					}
					if strings.TrimSpace(current.text.String()) != "" {
						if err := result.add("paragraph", label, current.text.String(), details); err != nil {
							return err
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	result.Summary = map[string]any{"paragraphs": paragraphs, "tables": tables, "section_properties": sections, "parts": len(parts)}
	return nil
}
