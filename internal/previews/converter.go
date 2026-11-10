// Package previews runs disposable, bounded conversions without opening originals
// for writing. It has no database access and never joins the indexing worker pool.
package previews

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
)

const Revision = "libreoffice-pdf-v1"
const MaximumOutput = 64 << 20

func Failure(code string) error {
	return domain.Failure(code, "No fue posible generar una vista previa. El archivo original continúa disponible.", 409)
}
func Program(configured string) string {
	if configured != "" {
		return configured
	}
	candidates := []string{}
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{"/Applications/LibreOffice.app/Contents/MacOS/soffice"}
	case "windows":
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramW6432")} {
			if base != "" {
				candidates = append(candidates, filepath.Join(base, "LibreOffice", "program", "soffice.com"))
			}
		}
	}
	if path, err := exec.LookPath("libreoffice"); err == nil {
		candidates = append(candidates, path)
	}
	if path, err := exec.LookPath("soffice"); err == nil {
		candidates = append(candidates, path)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate
		}
	}
	return ""
}

// Include the installation fingerprint: replacing/upgrading the converter makes
// its previous representations obsolete without changing document identities.
func Generator(configured string) string {
	program := Program(configured)
	info, err := os.Stat(program)
	identity := program
	if err == nil {
		identity += fmt.Sprintf("|%d|%d", info.Size(), info.ModTime().UnixNano())
	}
	return fmt.Sprintf("%s:%x", Revision, sha256.Sum256([]byte(identity)))
}
func Available(configured string) bool {
	info, err := os.Stat(Program(configured))
	return err == nil && info.Mode().IsRegular()
}

// Profiles are per conversion, never the interactive user's LibreOffice profile.
// Disable active content and all scripting independently of trust signatures.
const profile = `<?xml version="1.0" encoding="UTF-8"?>
<oor:items xmlns:oor="http://openoffice.org/2001/registry">
<item oor:path="/org.openoffice.Office.Common/Security/Scripting">
<prop oor:name="MacroSecurityLevel" oor:op="fuse"><value>3</value></prop>
<prop oor:name="DisableMacrosExecution" oor:op="fuse"><value>true</value></prop>
<prop oor:name="DisableActiveContent" oor:op="fuse"><value>true</value></prop>
<prop oor:name="DisableOLEAutomation" oor:op="fuse"><value>true</value></prop>
<prop oor:name="BlockUntrustedRefererLinks" oor:op="fuse"><value>true</value></prop>
</item>
<item oor:path="/org.openoffice.Office.Writer/Content/Update"><prop oor:name="Link" oor:op="fuse"><value>2</value></prop><prop oor:name="Field" oor:op="fuse"><value>false</value></prop><prop oor:name="Chart" oor:op="fuse"><value>false</value></prop></item>
<item oor:path="/org.openoffice.Office.Common/Misc"><prop oor:name="FirstRun" oor:op="fuse"><value>false</value></prop></item>
</oor:items>`

func Convert(ctx context.Context, source, format, output, configured string) error {
	if format != "docx" && format != "xlsx" {
		return Failure("PREVIEW_UNSUPPORTED")
	}
	program := Program(configured)
	if !Available(configured) {
		return Failure("PREVIEW_CONVERTER_MISSING")
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	detected, err := documentformat.Detect(ctx, file, "source."+format)
	file.Close()
	if err != nil {
		return err
	}
	if detected.Format != format {
		return Failure("FILE_TYPE_MISMATCH")
	}
	// Converters can follow links which the static extractor only ignores. Remove
	// external links/field instructions from the disposable conversion copy first.
	safeSource := filepath.Join(filepath.Dir(source), "safe."+format)
	if err = SanitizeOffice(ctx, source, safeSource, format); err != nil {
		return err
	}
	directory := filepath.Dir(source)
	profileDirectory := filepath.Join(directory, "profile")
	if err = os.MkdirAll(filepath.Join(profileDirectory, "user"), 0700); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(profileDirectory, "user", "registrymodifications.xcu"), []byte(profile), 0600); err != nil {
		return err
	}
	profileURL := url.URL{Scheme: "file", Path: filepath.ToSlash(profileDirectory)}
	if !strings.HasPrefix(profileURL.Path, "/") {
		profileURL.Path = "/" + profileURL.Path
	}
	filter := "pdf:writer_pdf_Export"
	if format == "xlsx" {
		filter = "pdf:calc_pdf_Export"
	}
	conversion, cancel := context.WithCancel(ctx)
	defer cancel()
	command := exec.CommandContext(conversion, program, "-env:UserInstallation="+profileURL.String(), "--headless", "--nologo", "--nodefault", "--norestore", "--convert-to", filter, "--outdir", directory, safeSource)
	command.Dir = directory
	command.Env = append(os.Environ(), "SAL_USE_VCLPLUGIN=svp", "TMPDIR="+directory, "TEMP="+directory, "TMP="+directory)
	// Diagnostic output may contain document content: discard it, never log it.
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	command.WaitDelay = 2 * time.Second
	configureProcess(command)
	if err = command.Start(); err != nil {
		return Failure("PREVIEW_CONVERSION_FAILED")
	}
	completed := make(chan error, 1)
	go func() { completed <- command.Wait() }()
	generated := filepath.Join(directory, "safe.pdf")
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	tooLarge := false
waiting:
	for {
		select {
		case err = <-completed:
			break waiting
		case <-ticker.C:
			if info, statError := os.Stat(generated); statError == nil && info.Size() > MaximumOutput {
				tooLarge = true
				cancel()
			}
		}
	}
	if tooLarge {
		return Failure("PREVIEW_SIZE_LIMIT")
	}
	if ctx.Err() != nil {
		return Failure("PREVIEW_TIMEOUT")
	}
	if err != nil {
		return Failure("PREVIEW_CONVERSION_FAILED")
	}
	info, err := os.Lstat(generated)
	if err != nil || !info.Mode().IsRegular() {
		return Failure("PREVIEW_CONVERSION_FAILED")
	}
	if info.Size() > MaximumOutput {
		return Failure("PREVIEW_SIZE_LIMIT")
	}
	pdf, err := os.Open(generated)
	if err != nil {
		return err
	}
	header := make([]byte, 5)
	_, err = io.ReadFull(pdf, header)
	pdf.Close()
	if err != nil || string(header) != "%PDF-" {
		return Failure("PREVIEW_CONVERSION_FAILED")
	}
	return os.Rename(generated, output)
}
