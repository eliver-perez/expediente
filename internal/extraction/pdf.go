// Package extraction runs external PDF/OCR programs without a shell, against a
// private snapshot, with bounded output, resolution, pages and execution time.
package extraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gestor-documental/internal/domain"
)

type Options struct {
	DisableOCR              bool                                         `json:"-"`
	MinimumNativeCharacters int                                          `json:"-"`
	Progress                func(operation string, completed, total int) `json:"-"`
	Concurrency             Concurrency                                  `json:"concurrency"`
	FontconfigFile          string                                       `json:"fontconfig_file,omitempty"`
	TessdataDirectory       string                                       `json:"tessdata_directory,omitempty"`
	PDFInfo                 string                                       `json:"pdfinfo"`
	PDFText                 string                                       `json:"pdftotext"`
	PDFRender               string                                       `json:"pdftoppm"`
	Tesseract               string                                       `json:"tesseract"`
	Workers                 int                                          `json:"workers"`
	MaximumFileMB           int                                          `json:"maximum_file_mb"`
	MaximumPages            int                                          `json:"maximum_pages"`
	PageTimeoutSeconds      int                                          `json:"page_timeout_seconds"`
}

func Defaults() Options {
	return Options{PDFInfo: "pdfinfo", PDFText: "pdftotext", PDFRender: "pdftoppm", Tesseract: "tesseract", Workers: 1, MaximumFileMB: 256, MaximumPages: 1000, PageTimeoutSeconds: 120}
}
func (options Options) Validate() error {
	if options.FontconfigFile != "" && !filepath.IsAbs(options.FontconfigFile) {
		return fmt.Errorf("fontconfig_file must be absolute")
	}
	if options.TessdataDirectory != "" && !filepath.IsAbs(options.TessdataDirectory) {
		return fmt.Errorf("tessdata_directory must be absolute")
	}
	if options.Concurrency.Mode != "" {
		if err := options.Concurrency.Validate(); err != nil {
			return err
		}
	}
	if options.Workers < 1 || options.Workers > 8 || options.MaximumFileMB < 1 || options.MaximumFileMB > 2048 || options.MaximumPages < 1 || options.MaximumPages > 10000 || options.PageTimeoutSeconds < 5 || options.PageTimeoutSeconds > 300 {
		return fmt.Errorf("indexing budgets outside allowed range")
	}
	for _, program := range []string{options.PDFInfo, options.PDFText, options.PDFRender, options.Tesseract} {
		if program == "" || strings.ContainsAny(program, "\r\n") {
			return fmt.Errorf("invalid extraction program")
		}
	}
	return nil
}

type Page struct {
	Number  int            `json:"page_number"`
	Method  string         `json:"extraction_method"`
	Text    string         `json:"text"`
	Kind    string         `json:"unit_kind"`
	Label   string         `json:"context_label"`
	Context map[string]any `json:"context"`
}
type limitedOutput struct {
	bytes.Buffer
	maximum int
}

func (output *limitedOutput) Write(contents []byte) (int, error) {
	if len(contents) > output.maximum-output.Len() {
		return 0, fmt.Errorf("process output limit")
	}
	return output.Buffer.Write(contents)
}
func (options Options) run(ctx context.Context, timeout int, maximum int, program, directory string, arguments ...string) ([]byte, error) {
	processContext, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	command := exec.CommandContext(processContext, program, arguments...)
	command.Dir = directory
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "LANG=C", "LC_ALL=C", "OMP_THREAD_LIMIT=1", "TMPDIR=" + directory, "XDG_CACHE_HOME=" + directory}
	data := options.TessdataDirectory
	if data == "" {
		data = os.Getenv("TESSDATA_PREFIX")
	}
	if data != "" {
		command.Env = append(command.Env, "TESSDATA_PREFIX="+data)
	}
	if options.FontconfigFile != "" {
		command.Env = append(command.Env, "FONTCONFIG_FILE="+options.FontconfigFile)
	}
	if runtime.GOOS == "windows" {
		command.Env = append(command.Env, "SystemRoot="+os.Getenv("SystemRoot"), "TEMP="+directory, "TMP="+directory)
	}
	command.WaitDelay = 2 * time.Second
	configureProcess(command)
	stdout := &limitedOutput{maximum: maximum}
	stderr := &limitedOutput{maximum: 16384}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if processContext.Err() != nil {
			return nil, domain.Failure("PROCESS_TIMEOUT", "Se agotó el tiempo de extracción.", 409)
		}
		if _, err := exec.LookPath(program); err != nil {
			return nil, domain.Failure("EXTRACTOR_UNAVAILABLE", "Falta una herramienta PDF/OCR configurada.", 409)
		}
		if program == options.Tesseract {
			return nil, domain.Failure("OCR_FAILED", "No se pudo reconocer el texto con OCR.", 409)
		}
		return nil, domain.Failure("EXTRACTION_FAILED", "El PDF no pudo procesarse: comprueba formato, cifrado e idiomas instalados.", 409)
	}
	return stdout.Bytes(), nil
}
func (options Options) ValidatePDF(ctx context.Context, documentPath string) (int, error) {
	directory := filepath.Dir(documentPath)
	info, err := options.run(ctx, options.PageTimeoutSeconds, 1<<20, options.PDFInfo, directory, documentPath)
	if err != nil {
		var failure *domain.Error
		if errors.As(err, &failure) && failure.Code == "EXTRACTION_FAILED" {
			return 0, domain.Failure("INVALID_PDF", "No se pudo validar el PDF; puede estar dañado o cifrado.", 409)
		}
		return 0, err
	}
	pageCount := 0
	for _, line := range strings.Split(string(info), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			pageCount, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
		}
	}
	if pageCount < 1 || pageCount > options.MaximumPages {
		return 0, domain.Failure("PAGE_LIMIT", "El número de páginas supera el límite configurado o es inválido.", 409)
	}
	return pageCount, nil
}

// Prepare reads native text in batches, amortizing process startup and PDF parsing.
// Form-feed boundaries preserve page numbers, including blank pages.
func (options Options) Prepare(ctx context.Context, documentPath string) ([]Page, error) {
	count, err := options.ValidatePDF(ctx, documentPath)
	if err != nil {
		return nil, err
	}
	pages := make([]Page, 0, count)
	totalBytes := 0
	for start := 1; start <= count; start += 32 {
		end := min(count, start+31)
		if options.Progress != nil {
			options.Progress("native", start-1, count)
		}
		contents, err := options.run(ctx, options.PageTimeoutSeconds, 32<<20, options.PDFText, filepath.Dir(documentPath), "-f", strconv.Itoa(start), "-l", strconv.Itoa(end), "-enc", "UTF-8", "-layout", documentPath, "-")
		if err != nil {
			return nil, err
		}
		texts := strings.Split(string(contents), "\f")
		if len(texts) == end-start+2 && strings.TrimSpace(texts[len(texts)-1]) == "" {
			texts = texts[:len(texts)-1]
		}
		if len(texts) != end-start+1 {
			return nil, domain.Failure("EXTRACTION_FAILED", "No se pudieron separar las páginas del PDF.", 409)
		}
		for index, text := range texts {
			text = strings.ToValidUTF8(text, "")
			totalBytes += len(text)
			if len(text) > 4<<20 || totalBytes > 32<<20 {
				return nil, domain.Failure("TEXT_LIMIT", "El texto extraído supera el límite por documento.", 409)
			}
			letters := 0
			for _, character := range text {
				if unicode.IsLetter(character) || unicode.IsNumber(character) {
					letters++
				}
			}
			method := "native"
			threshold := options.MinimumNativeCharacters
			if threshold <= 0 {
				threshold = 32
			}
			if letters < threshold && !options.DisableOCR {
				method = "pending_ocr"
			}
			pages = append(pages, Page{Number: start + index, Method: method, Text: text})
		}
		if options.Progress != nil {
			options.Progress("native", end, count)
		}
	}
	return pages, nil
}
func NeedsOCR(pages []Page) bool {
	for _, page := range pages {
		if page.Method == "pending_ocr" {
			return true
		}
	}
	return false
}
func (options Options) Recognize(ctx context.Context, documentPath, languages string, pages []Page) ([]Page, error) {
	if languages != "spa" && languages != "eng" && languages != "spa+eng" {
		return nil, fmt.Errorf("unsupported OCR languages")
	}
	directory := filepath.Dir(documentPath)
	totalBytes := 0
	completed := 0
	for _, page := range pages {
		if page.Method != "pending_ocr" {
			completed++
		}
	}
	for index := range pages {
		page := &pages[index]
		if page.Method == "pending_ocr" {
			argument := strconv.Itoa(page.Number)
			if options.Progress != nil {
				options.Progress("render", completed, len(pages))
			}
			rendered, err := options.run(ctx, options.PageTimeoutSeconds, 32<<20, options.PDFRender, directory, "-f", argument, "-l", argument, "-singlefile", "-scale-to", "2500", "-png", documentPath)
			if err != nil {
				return nil, err
			}
			imagePath := filepath.Join(directory, "ocr-page.png")
			if err = os.WriteFile(imagePath, rendered, 0600); err != nil {
				return nil, err
			}
			if options.Progress != nil {
				options.Progress("ocr", completed, len(pages))
			}
			recognized, err := options.run(ctx, options.PageTimeoutSeconds, 4<<20, options.Tesseract, directory, imagePath, "stdout", "-l", languages, "--psm", "3")
			removalError := os.Remove(imagePath)
			if err != nil {
				return nil, err
			}
			if removalError != nil {
				return nil, removalError
			}
			page.Text = strings.ToValidUTF8(string(recognized), "")
			page.Method = "ocr"
			if strings.TrimSpace(page.Text) == "" {
				page.Method = "empty"
			}
			completed++
			if options.Progress != nil {
				options.Progress("ocr", completed, len(pages))
			}
		}
		totalBytes += len(page.Text)
		if totalBytes > 32<<20 {
			return nil, domain.Failure("TEXT_LIMIT", "El texto extraído supera el límite por documento.", 409)
		}
	}
	return pages, nil
}
func (options Options) Extract(ctx context.Context, documentPath, languages string) ([]Page, error) {
	pages, err := options.Prepare(ctx, documentPath)
	if err != nil {
		return nil, err
	}
	return options.Recognize(ctx, documentPath, languages, pages)
}
func (options Options) Diagnostics() map[string]bool {
	result := map[string]bool{}
	for name, program := range map[string]string{"pdfinfo": options.PDFInfo, "pdftotext": options.PDFText, "pdftoppm": options.PDFRender, "tesseract": options.Tesseract} {
		_, err := exec.LookPath(program)
		result[name] = err == nil
	}
	return result
}

var _ io.Writer = (*limitedOutput)(nil)
