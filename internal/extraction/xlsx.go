package extraction

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gestor-documental/internal/documentformat"
)

type spreadsheetCell struct {
	address, kind     string
	value, inline     strings.Builder
	formula, hasValue bool
}

func extractXLSX(ctx context.Context, file *os.File, options Options, result *Result) error {
	archive, err := documentformat.OpenOffice(ctx, file, "xlsx")
	if err != nil {
		return err
	}
	relations, err := officeRelationships(ctx, archive, "xl/workbook.xml")
	if err != nil {
		return err
	}
	byID := map[string]relationship{}
	shared := []string{}
	sharedPart := ""
	for _, rel := range relations {
		byID[rel.ID] = rel
		if rel.External {
			result.warn("EXTERNAL_REFERENCES_IGNORED")
			continue
		}
		if relationKind(rel.Kind, "sharedStrings") {
			if sharedPart != "" {
				return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
			}
			sharedPart = rel.Target
		}
	}
	if sharedPart != "" {
		shared, err = sharedStrings(ctx, archive, sharedPart)
		if err != nil {
			return err
		}
	}
	type sheet struct{ name, path string }
	sheets := []sheet{}
	seenNames, seenPaths := map[string]bool{}, map[string]bool{}
	err = archive.XML(ctx, "xl/workbook.xml", "workbook", sheetNS, func(token xml.Token) error {
		element, ok := token.(xml.StartElement)
		if !ok || !sameNamespace(element.Name.Space, sheetNS) || element.Name.Local != "sheet" {
			return nil
		}
		name, id := attribute(element, "name"), ""
		for _, attr := range element.Attr {
			if attr.Name.Local == "id" && sameNamespace(attr.Name.Space, relationNS) {
				id = attr.Value
			}
		}
		rel, exists := byID[id]
		if !exists || rel.External || name == "" || len(name) > 256 || seenNames[name] || seenPaths[rel.Target] {
			return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
		}
		if !relationKind(rel.Kind, "worksheet") {
			result.warn("NON_WORKSHEET_SHEET_IGNORED")
			return nil
		}
		if len(sheets) >= 1024 {
			return documentformat.Failure("DOCUMENT_COMPLEXITY_LIMIT")
		}
		sheets = append(sheets, sheet{name, rel.Target})
		seenNames[name] = true
		seenPaths[rel.Target] = true
		return nil
	})
	if err != nil {
		return err
	}
	cells, formulas, missing := 0, 0, 0
	for _, sheet := range sheets {
		previousUnits := len(result.Units)
		row, column := 0, 0
		var cell *spreadsheetCell
		inValue, inText, phonetic := false, false, 0
		lastRow := 0
		err = archive.XML(ctx, sheet.path, "worksheet", sheetNS, func(token xml.Token) error {
			switch element := token.(type) {
			case xml.StartElement:
				if !sameNamespace(element.Name.Space, sheetNS) {
					return nil
				}
				switch element.Name.Local {
				case "row":
					row = lastRow + 1
					if value := attribute(element, "r"); value != "" {
						parsed, err := strconv.Atoi(value)
						if err != nil {
							return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
						}
						row = parsed
					}
					if row <= lastRow || row > 1048576 {
						return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
					}
					lastRow = row
					column = 0
				case "c":
					if cell != nil || row < 1 {
						return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
					}
					address := attribute(element, "r")
					nextColumn := column + 1
					if address != "" {
						parsedColumn, parsedRow, valid := cellAddress(address)
						if !valid || parsedRow != row || parsedColumn <= column {
							return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
						}
						nextColumn = parsedColumn
					} else {
						address = columnName(nextColumn) + strconv.Itoa(row)
					}
					if nextColumn > 16384 {
						return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
					}
					column = nextColumn
					cell = &spreadsheetCell{address: address, kind: attribute(element, "t")}
					if attribute(element, "s") != "" {
						result.warn("CELL_FORMATTING_NOT_APPLIED")
					}
				case "v":
					if cell != nil {
						inValue = true
						cell.hasValue = true
					}
				case "t":
					inText = true
				case "rPh":
					phonetic++
				case "f":
					if cell != nil {
						cell.formula = true
					}
				}
			case xml.CharData:
				if cell != nil {
					if inValue {
						cell.value.Write(element)
						if err := textLimit(&cell.value); err != nil {
							return err
						}
					}
					if inText && phonetic == 0 {
						cell.inline.Write(element)
						if err := textLimit(&cell.inline); err != nil {
							return err
						}
					}
				}
			case xml.EndElement:
				if !sameNamespace(element.Name.Space, sheetNS) {
					return nil
				}
				switch element.Name.Local {
				case "v":
					inValue = false
				case "t":
					inText = false
				case "rPh":
					phonetic--
				case "c":
					if cell == nil {
						return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
					}
					value := cell.value.String()
					switch cell.kind {
					case "s":
						index, err := strconv.Atoi(strings.TrimSpace(value))
						if err != nil || index < 0 || index >= len(shared) {
							return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
						}
						value = shared[index]
					case "inlineStr":
						value = cell.inline.String()
					case "b":
						if value == "1" {
							value = "Verdadero"
						} else if value == "0" {
							value = "Falso"
						} else if value != "" {
							return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
						}
					case "", "n", "str", "e", "d":
					default:
						return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
					}
					if cell.formula {
						formulas++
						result.warn("FORMULAS_NOT_EVALUATED")
						if !cell.hasValue || value == "" {
							missing++
							result.warn("FORMULA_VALUE_UNAVAILABLE")
						}
					}
					if value != "" || cell.formula {
						cells++
						label := fmt.Sprintf("Hoja %s · celda %s", sheet.name, cell.address)
						if err := result.add("cell", label, label+"\n"+value, map[string]any{"sheet": sheet.name, "cell": cell.address, "formula": cell.formula, "cached_value": cell.formula && cell.hasValue}); err != nil {
							return err
						}
					}
					cell = nil
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		// An empty sheet still has a searchable name, without claiming a cell.
		if len(result.Units) == previousUnits {
			if err := result.add("sheet", "Hoja "+sheet.name, sheet.name, map[string]any{"sheet": sheet.name}); err != nil {
				return err
			}
		}
	}
	result.Summary = map[string]any{"sheets": len(sheets), "cells": cells, "formula_cells": formulas, "formulas_without_cached_value": missing}
	return nil
}

func sharedStrings(ctx context.Context, archive *documentformat.OfficePackage, name string) ([]string, error) {
	values := []string{}
	var text strings.Builder
	active, inText, phonetic := false, false, 0
	total := 0
	err := archive.XML(ctx, name, "sst", sheetNS, func(token xml.Token) error {
		switch element := token.(type) {
		case xml.StartElement:
			if !sameNamespace(element.Name.Space, sheetNS) {
				return nil
			}
			switch element.Name.Local {
			case "si":
				if active {
					return documentformat.Failure("INVALID_OFFICE_DOCUMENT")
				}
				active = true
				text.Reset()
			case "t":
				inText = true
			case "rPh":
				phonetic++
			}
		case xml.CharData:
			if active && inText && phonetic == 0 {
				text.Write(element)
				return textLimit(&text)
			}
		case xml.EndElement:
			if !sameNamespace(element.Name.Space, sheetNS) {
				return nil
			}
			switch element.Name.Local {
			case "t":
				inText = false
			case "rPh":
				phonetic--
			case "si":
				total += text.Len()
				if len(values) >= 200000 || total > 32<<20 {
					return documentformat.Failure("DOCUMENT_COMPLEXITY_LIMIT")
				}
				values = append(values, text.String())
				active = false
			}
		}
		return nil
	})
	return values, err
}

func cellAddress(address string) (int, int, bool) {
	column, index := 0, 0
	for index < len(address) && address[index] >= 'A' && address[index] <= 'Z' {
		column = column*26 + int(address[index]-'A'+1)
		index++
		if column > 16384 {
			return 0, 0, false
		}
	}
	row, err := strconv.Atoi(address[index:])
	return column, row, err == nil && column > 0 && row > 0 && row <= 1048576
}
func columnName(column int) string {
	name := ""
	for column > 0 {
		column--
		name = string(rune('A'+column%26)) + name
		column /= 26
	}
	return name
}
