// Package documentformat identifies document bytes and enforces admission rules.
// It never executes, extracts to disk, or follows references inside a document.
package documentformat

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"gestor-documental/internal/domain"
)

type Format struct {
	ID                 string `json:"id"`
	MIME               string `json:"mime"`
	ExtractorAvailable bool   `json:"extractor_available"`
}

// The catalog declares extraction capability, independently of admission policy.
// The extractor registry tests ensure every declared capability has an engine.
func Formats() []Format {
	return []Format{
		{"pdf", "application/pdf", true},
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", true},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", true},
		{"txt", "text/plain", true},
		{"csv", "text/csv", true},
	}
}

func Lookup(id string) (Format, bool) {
	for _, format := range Formats() {
		if format.ID == id {
			return format, true
		}
	}
	return Format{}, false
}

type Detection struct {
	Format    string `json:"format"`
	MIME      string `json:"detected_mime"`
	Extension string `json:"extension"`
	Mismatch  bool   `json:"extension_mismatch"`
}

func Failure(code string) error {
	messages := map[string]string{
		"FILE_TYPE_BLOCKED":         "El archivo pertenece a una categoría no permitida (ejecutable, script, paquete o archivo comprimido).",
		"FILE_TYPE_UNKNOWN":         "El formato no está reconocido o su codificación no está admitida.",
		"FILE_TYPE_MISMATCH":        "La extensión no coincide con el contenido detectado. Corrige el nombre del archivo antes de incorporarlo.",
		"INVALID_PDF":               "La extensión PDF no coincide con el contenido del archivo.",
		"INVALID_OFFICE_DOCUMENT":   "El documento Office no tiene una estructura DOCX/XLSX válida y segura.",
		"DOCUMENT_COMPLEXITY_LIMIT": "El documento supera los límites de tamaño o complejidad admitidos.",
		"FILE_FORMAT_DISABLED":      "La biblioteca no permite almacenar este formato.",
		"FILE_SIZE_LIMIT":           "El archivo supera el tamaño máximo permitido por la biblioteca.",
		"FILE_NOT_INDEXABLE":        "La biblioteca rechaza documentos que no pueden indexarse con su configuración actual.",
	}
	return domain.Failure(code, messages[code], 422)
}

func BlockedExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimRight(name, " .")))
	// Include common equivalent executable, active markup and macro formats.
	return strings.Contains("|exe|msi|msp|bat|cmd|ps1|psm1|psd1|sh|bash|zsh|fish|dll|so|dylib|jar|com|scr|app|pkg|deb|rpm|zip|rar|7z|tar|gz|bz2|xz|tgz|cab|iso|dmg|sys|bin|lnk|url|reg|hta|vbs|vbe|js|jse|wsf|wsh|py|pyc|pl|rb|php|html|htm|svg|wasm|docm|xlsm|xlam|dotm|xltm|doc|xls|", "|"+strings.TrimPrefix(ext, ".")+"|") && ext != ""
}

// Detect reads from the same opened handle used for hashing. ReadAt/sections do
// not move its offset. ZIP and text validation are bounded and cancellable.
func Detect(ctx context.Context, file *os.File, filename string) (Detection, error) {
	result := Detection{Extension: strings.ToLower(filepath.Ext(filename)), MIME: "application/octet-stream"}
	if BlockedExtension(filename) {
		return result, Failure("FILE_TYPE_BLOCKED")
	}
	info, err := file.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, Failure("FILE_TYPE_BLOCKED")
	}
	if info.Size() > 4096<<20 {
		return result, Failure("FILE_SIZE_LIMIT")
	}
	header := make([]byte, min(info.Size(), 65536))
	if _, err = file.ReadAt(header, 0); err != nil && err != io.EOF {
		return result, err
	}
	result.MIME = http.DetectContentType(header)
	if blockedHeader(header) {
		return result, Failure("FILE_TYPE_BLOCKED")
	}
	format := ""
	switch {
	case bytes.HasPrefix(header, []byte("%PDF-")):
		format = "pdf" // Poppler remains the full PDF validator before extraction/upload.
	case bytes.HasPrefix(header, []byte("PK\x03\x04")), bytes.HasPrefix(header, []byte("PK\x05\x06")):
		format, err = detectOffice(ctx, file, info.Size())
		if err != nil {
			return result, err
		}
	default:
		reader, _, openError := TextReader(ctx, file, info.Size())
		if openError != nil {
			return result, openError
		}
		prefix := bufio.NewReaderSize(reader, 65536)
		decodedHeader, _ := prefix.Peek(65536)
		if blockedHeader(decodedHeader) {
			return result, Failure("FILE_TYPE_BLOCKED")
		}
		if err = validateText(ctx, prefix); err != nil {
			return result, err
		}
		format = "txt"
		// CSV has no magic bytes: require consistent delimited records, and use
		// its extension only to disambiguate valid CSV from ordinary prose.
		if result.Extension == ".csv" && validCSV(ctx, file, info.Size()) {
			format = "csv"
		}
	}
	descriptor, _ := Lookup(format)
	result.Format, result.MIME = format, descriptor.MIME
	result.Mismatch = result.Extension != "."+format
	if result.Mismatch {
		if result.Extension == ".pdf" {
			return result, Failure("INVALID_PDF")
		}
		if _, known := Lookup(strings.TrimPrefix(result.Extension, ".")); known {
			return result, Failure("FILE_TYPE_MISMATCH")
		}
		// Unknown extensions are decided by policy; binary containers never pass.
		if format != "txt" {
			return result, Failure("FILE_TYPE_MISMATCH")
		}
	}
	return result, nil
}

func blockedHeader(header []byte) bool {
	for _, magic := range [][]byte{[]byte("MZ"), []byte("\x7fELF"), []byte("\xfe\xed\xfa\xce"), []byte("\xce\xfa\xed\xfe"), []byte("\xfe\xed\xfa\xcf"), []byte("\xcf\xfa\xed\xfe"), []byte("\xca\xfe\xba\xbe"), []byte("\xbe\xba\xfe\xca"), []byte("\xca\xfe\xba\xbf"), []byte("\xbf\xba\xfe\xca"), []byte("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"), []byte("Rar!"), []byte("7z\xbc\xaf\x27\x1c"), []byte("\x1f\x8b"), []byte("BZh"), []byte("\xfd7zXZ\x00"), []byte("!<arch>"), []byte("\xed\xab\xee\xdb"), []byte("MSCF"), []byte("xar!"), []byte("\x00asm")} {
		if bytes.HasPrefix(header, magic) {
			return true
		}
	}
	if len(header) > 262 && string(header[257:262]) == "ustar" {
		return true
	}
	text := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(string(header), "\ufeff")))
	for _, prefix := range []string{"#!", "@echo", "<?php", "<!doctype html", "<html", "<script", "<svg", "<?xml", "powershell ", "pwsh ", "param(", "param (", "[cmdletbinding", "import ", "from ", "#!/", "function ", "#!/usr"} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func validateText(ctx context.Context, input io.Reader) error {
	reader := bufio.NewReaderSize(input, 65536)
	for count := 0; ; count++ {
		if count%4096 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		character, size, err := reader.ReadRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if character == utf8.RuneError && size == 1 || unicode.IsControl(character) && character != '\t' && character != '\r' && character != '\n' && character != '\f' {
			return Failure("FILE_TYPE_UNKNOWN")
		}
	}
}

func validCSV(ctx context.Context, file *os.File, size int64) bool {
	_, err := OpenCSV(ctx, file, size)
	return err == nil
}

// encoding/csv may buffer a whole quoted record. Bound each read independently
// of the permitted file size to prevent a giant cell exhausting worker memory.
type recordReader struct {
	reader    io.Reader
	remaining int
}

func (reader *recordReader) Read(buffer []byte) (int, error) {
	if reader.remaining <= 0 {
		return 0, Failure("DOCUMENT_COMPLEXITY_LIMIT")
	}
	if len(buffer) > reader.remaining {
		buffer = buffer[:reader.remaining]
	}
	count, err := reader.reader.Read(buffer)
	reader.remaining -= count
	return count, err
}
