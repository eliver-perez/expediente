package extraction

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
)

type Descriptor struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Format  string `json:"format"`
}
type Result struct {
	Extractor Descriptor     `json:"extractor"`
	Units     []Page         `json:"-"`
	Summary   map[string]any `json:"summary"`
	Warnings  []string       `json:"warnings"`
	textBytes int
}
type documentExtractor struct {
	Descriptor
	prepare  func(context.Context, *os.File, Options, *Result) error
	complete func(context.Context, string, string, Options, *Result) error
}

// The registry is the only dispatch point. PDF keeps its two-stage native/OCR
// flow, so Office or text documents never occupy an OCR worker or invoke Office.
func registry() map[string]documentExtractor {
	return map[string]documentExtractor{
		"pdf":  {Descriptor{"poppler-tesseract", "2", "pdf"}, preparePDF, completePDF},
		"docx": {Descriptor{"docx-openxml", "2", "docx"}, extractDOCX, nil},
		"xlsx": {Descriptor{"xlsx-openxml", "1", "xlsx"}, extractXLSX, nil},
		"txt":  {Descriptor{"text-unicode", "1", "txt"}, extractTXT, nil},
		"csv":  {Descriptor{"csv-structured", "1", "csv"}, extractCSV, nil},
	}
}
func ExtractorFor(format string) (Descriptor, bool) {
	engine, ok := registry()[format]
	return engine.Descriptor, ok
}

func PrepareDocument(ctx context.Context, documentPath, format string, options Options) (Result, error) {
	engine, ok := registry()[format]
	result := Result{Extractor: engine.Descriptor, Summary: map[string]any{}, Warnings: []string{}, Units: []Page{}}
	if !ok {
		return result, domain.Failure("EXTRACTOR_UNAVAILABLE", "No hay extractor para este formato.", 409)
	}
	file, err := os.Open(documentPath)
	if err != nil {
		return result, err
	}
	defer file.Close()
	// Validate the private snapshot, including admission's ZIP and encoding guards.
	if options.Progress != nil {
		options.Progress("validating", 0, 0)
	}
	detected, err := documentformat.Detect(ctx, file, "source."+format)
	if err != nil {
		return result, err
	}
	if detected.Format != format {
		return result, documentformat.Failure("FILE_TYPE_MISMATCH")
	}
	if options.Progress != nil {
		options.Progress("native", 0, 0)
	}
	err = engine.prepare(ctx, file, options, &result)
	if err == nil && options.Progress != nil {
		options.Progress("metadata", len(result.Units), len(result.Units))
	}
	return result, err
}
func CompleteDocument(ctx context.Context, documentPath, languages string, options Options, result *Result) error {
	engine, ok := registry()[result.Extractor.Format]
	if !ok {
		return domain.Failure("EXTRACTOR_UNAVAILABLE", "No hay extractor para este formato.", 409)
	}
	if engine.complete != nil {
		return engine.complete(ctx, documentPath, languages, options, result)
	}
	return ctx.Err()
}

func preparePDF(ctx context.Context, file *os.File, options Options, result *Result) error {
	pages, err := options.Prepare(ctx, file.Name())
	result.Units = pages
	return err
}
func completePDF(ctx context.Context, documentPath, languages string, options Options, result *Result) error {
	pages, err := options.Recognize(ctx, documentPath, languages, result.Units)
	if err != nil {
		return err
	}
	ocr := 0
	for index := range pages {
		page := &pages[index]
		page.Kind, page.Label = "page", fmt.Sprintf("Página %d", page.Number)
		page.Context = map[string]any{"page": page.Number}
		if page.Method == "ocr" || page.Method == "empty" {
			ocr++
		}
	}
	result.Units = pages
	result.Summary = map[string]any{"pages": len(pages), "ocr_pages": ocr}
	return nil
}

func (r *Result) add(kind, label, text string, details map[string]any) error {
	// Fail atomically instead of silently indexing only the start of a document.
	if len(r.Units) >= 50000 || len(text) > 4<<20 || r.textBytes+len(text) > 32<<20 {
		return domain.Failure("TEXT_LIMIT", "El texto o las unidades extraídas superan el límite por documento.", 409)
	}
	r.textBytes += len(text)
	r.Units = append(r.Units, Page{Number: len(r.Units) + 1, Method: "native", Text: text, Kind: kind, Label: label, Context: details})
	return nil
}
func (r *Result) warn(code string) {
	for _, existing := range r.Warnings {
		if existing == code {
			return
		}
	}
	r.Warnings = append(r.Warnings, code)
}
func textLimit(text *strings.Builder) error {
	if text.Len() > 4<<20 {
		return domain.Failure("TEXT_LIMIT", "Una unidad de texto supera el límite permitido.", 409)
	}
	return nil
}
