// Package extraction runs external PDF/OCR programs without a shell, against a
// private snapshot, with bounded output, resolution, pages and execution time.
package extraction

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gestor-documental/internal/domain"
)

type Options struct {
	PDFInfo            string `json:"pdfinfo"`
	PDFText            string `json:"pdftotext"`
	PDFRender          string `json:"pdftoppm"`
	Tesseract          string `json:"tesseract"`
	Workers            int    `json:"workers"`
	MaximumFileMB      int    `json:"maximum_file_mb"`
	MaximumPages       int    `json:"maximum_pages"`
	PageTimeoutSeconds int    `json:"page_timeout_seconds"`
}

func Defaults() Options {
	return Options{PDFInfo: "pdfinfo", PDFText: "pdftotext", PDFRender: "pdftoppm", Tesseract: "tesseract", Workers: 1, MaximumFileMB: 256, MaximumPages: 1000, PageTimeoutSeconds: 120}
}
func (options Options) Validate() error {
	if options.Workers < 1 || options.Workers > 4 || options.MaximumFileMB < 1 || options.MaximumFileMB > 2048 || options.MaximumPages < 1 || options.MaximumPages > 10000 || options.PageTimeoutSeconds < 5 || options.PageTimeoutSeconds > 300 {
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
	Number int    `json:"page_number"`
	Method string `json:"extraction_method"`
	Text   string `json:"text"`
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
func run(ctx context.Context, timeout int, maximum int, program, directory string, arguments ...string) ([]byte, error) {
	processContext, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	command := exec.CommandContext(processContext, program, arguments...)
	command.Dir = directory
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "LANG=C", "LC_ALL=C", "OMP_THREAD_LIMIT=1", "TMPDIR=" + directory, "XDG_CACHE_HOME=" + directory}
	if data := os.Getenv("TESSDATA_PREFIX"); data != "" {
		command.Env = append(command.Env, "TESSDATA_PREFIX="+data)
	}
	command.WaitDelay = 2 * time.Second
	configureProcess(command)
	stdout := &limitedOutput{maximum: maximum}
	stderr := &limitedOutput{maximum: 16384}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if processContext.Err() != nil {
			return nil, domain.Failure("PROCESS_TIMEOUT", "Se agotó el tiempo de extracción.", 409)
		}
		if _, err := exec.LookPath(program); err != nil {
			return nil, domain.Failure("EXTRACTOR_UNAVAILABLE", "Falta una herramienta PDF/OCR configurada.", 409)
		}
		return nil, domain.Failure("EXTRACTION_FAILED", "El PDF no pudo procesarse: comprueba formato, cifrado e idiomas instalados.", 409)
	}
	return stdout.Bytes(), nil
}
func (options Options) Extract(ctx context.Context, documentPath, languages string) ([]Page, error) {
	if languages != "spa" && languages != "eng" && languages != "spa+eng" {
		return nil, fmt.Errorf("unsupported OCR languages")
	}
	directory := filepath.Dir(documentPath)
	info, err := run(ctx, options.PageTimeoutSeconds, 1<<20, options.PDFInfo, directory, documentPath)
	if err != nil {
		return nil, err
	}
	pageCount := 0
	for _, line := range strings.Split(string(info), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			pageCount, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
		}
	}
	if pageCount < 1 || pageCount > options.MaximumPages {
		return nil, domain.Failure("PAGE_LIMIT", "El número de páginas supera el límite configurado o es inválido.", 409)
	}
	pages := make([]Page, 0, pageCount)
	totalBytes := 0
	for number := 1; number <= pageCount; number++ {
		pageArgument := strconv.Itoa(number)
		contents, err := run(ctx, options.PageTimeoutSeconds, 4<<20, options.PDFText, directory, "-f", pageArgument, "-l", pageArgument, "-enc", "UTF-8", "-layout", "-nopgbrk", documentPath, "-")
		if err != nil {
			return nil, err
		}
		text := strings.ToValidUTF8(string(contents), "")
		method := "native"
		letters := 0
		for _, character := range text {
			if unicode.IsLetter(character) || unicode.IsNumber(character) {
				letters++
			}
		}
		if letters < 32 {
			rendered, err := run(ctx, options.PageTimeoutSeconds, 32<<20, options.PDFRender, directory, "-f", pageArgument, "-l", pageArgument, "-singlefile", "-scale-to", "2500", "-png", documentPath)
			if err != nil {
				return nil, err
			}
			imagePath := filepath.Join(directory, "ocr-page.png")
			if err = os.WriteFile(imagePath, rendered, 0600); err != nil {
				return nil, err
			}
			recognized, err := run(ctx, options.PageTimeoutSeconds, 4<<20, options.Tesseract, directory, imagePath, "stdout", "-l", languages, "--psm", "3")
			removalError := os.Remove(imagePath)
			if err != nil {
				return nil, err
			}
			if removalError != nil {
				return nil, removalError
			}
			text = strings.ToValidUTF8(string(recognized), "")
			method = "ocr"
			if strings.TrimSpace(text) == "" {
				method = "empty"
			}
		}
		if !utf8.ValidString(text) {
			return nil, fmt.Errorf("invalid extraction encoding")
		}
		totalBytes += len(text)
		if totalBytes > 32<<20 {
			return nil, domain.Failure("TEXT_LIMIT", "El texto extraído supera el límite por documento.", 409)
		}
		pages = append(pages, Page{Number: number, Method: method, Text: text})
	}
	return pages, nil
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
