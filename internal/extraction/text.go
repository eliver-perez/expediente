package extraction

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"gestor-documental/internal/documentformat"
)

func extractTXT(ctx context.Context, file *os.File, options Options, result *Result) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	reader, encoding, err := documentformat.TextReader(ctx, file, info.Size())
	if err != nil {
		return err
	}
	// Chunk by actual lines; the API calls these text units, never PDF pages.
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 65536), 4<<20)
	lines, characters, start := 0, 0, 1
	var text strings.Builder
	flush := func() error {
		if text.Len() == 0 && lines < start {
			return nil
		}
		err := result.add("lines", fmt.Sprintf("Líneas %d–%d", start, lines), text.String(), map[string]any{"line_start": start, "line_end": lines})
		text.Reset()
		start = lines + 1
		return err
	}
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		lines++
		characters += utf8.RuneCountInString(line)
		text.WriteString(line)
		text.WriteByte('\n')
		if text.Len() >= 64<<10 || lines-start >= 199 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := scanner.Err(); err != nil {
		return documentformat.Failure("DOCUMENT_COMPLEXITY_LIMIT")
	}
	if err := flush(); err != nil {
		return err
	}
	result.Summary = map[string]any{"encoding": encoding, "lines": lines, "characters_without_line_breaks": characters}
	return nil
}

func extractCSV(ctx context.Context, file *os.File, options Options, result *Result) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	stream, err := documentformat.OpenCSV(ctx, file, info.Size())
	if err != nil {
		return err
	}
	rows, columns, cells := 0, 0, 0
	first, err := stream.Read()
	if err != nil {
		return err
	}
	firstLine, _ := stream.FieldPos(0)
	second, secondError := stream.Read()
	if secondError != nil && secondError != io.EOF {
		return secondError
	}
	header := probableHeader(first, second)
	if header {
		result.warn("CSV_HEADER_INFERRED")
	}
	buffered := [][]string{first}
	lines := []int{firstLine}
	// FieldPos refers to the most recent record, so retain physical line numbers
	// separately while inspecting the first two rows for a conservative hint.
	if secondError == nil {
		buffered = append(buffered, second)
		line, _ := stream.FieldPos(0)
		lines = append(lines, line)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var record []string
		line := 0
		var err error
		if len(buffered) > 0 {
			record = buffered[0]
			line = lines[0]
			buffered = buffered[1:]
			lines = lines[1:]
		} else {
			record, err = stream.Read()
			if err == nil {
				line, _ = stream.FieldPos(0)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			if cancelled := ctx.Err(); cancelled != nil {
				return cancelled
			}
			return documentformat.Failure("FILE_TYPE_MISMATCH")
		}
		rows++
		columns = len(record)
		if len(record) > 16384 {
			return documentformat.Failure("DOCUMENT_COMPLEXITY_LIMIT")
		}
		var text strings.Builder
		for index, value := range record {
			if value != "" {
				cells++
			}
			if header && rows > 1 {
				fmt.Fprintf(&text, "%s: %s\n", first[index], value)
			} else {
				fmt.Fprintf(&text, "Columna %d: %s\n", index+1, value)
			}
			if err := textLimit(&text); err != nil {
				return err
			}
		}
		if err := result.add("record", fmt.Sprintf("Registro %d · línea %d", rows, line), text.String(), map[string]any{"record": rows, "source_line": line, "columns": columns}); err != nil {
			return err
		}
	}
	// The hint never removes the first row: CSV has no authoritative header flag.
	headerMode := "not_declared"
	if header {
		headerMode = "inferred_first_record"
	}
	result.Summary = map[string]any{"encoding": stream.Encoding, "delimiter": string(stream.Delimiter), "records": rows, "columns": columns, "nonempty_cells": cells, "header": headerMode}
	return nil
}

func probableHeader(first, second []string) bool {
	if len(first) != len(second) || len(first) == 0 || len(first) > 16384 {
		return false
	}
	seen := map[string]bool{}
	numericBelow := false
	for index, value := range first {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 256 || seen[strings.ToLower(value)] {
			return false
		}
		seen[strings.ToLower(value)] = true
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return false
		}
		if _, err := strconv.ParseFloat(strings.TrimSpace(second[index]), 64); err == nil {
			numericBelow = true
		}
	}
	return numericBelow
}
