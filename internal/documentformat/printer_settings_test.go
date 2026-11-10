package documentformat

import (
	"strings"
	"testing"
)

func TestOfficePrinterSettingsAreDataNotExecutables(t *testing.T) {
	for _, format := range []string{"docx", "xlsx"} {
		prefix, kind, main := "word", "wordprocessingml", "document"
		if format == "xlsx" {
			prefix, kind, main = "xl", "spreadsheetml", "sheet"
		}
		mainPart := prefix + "/document.xml"
		if format == "xlsx" {
			mainPart = "xl/workbook.xml"
		}
		types := `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="bin" ContentType="application/vnd.openxmlformats-officedocument.` + kind + `.printerSettings"/><Override PartName="/` + mainPart + `" ContentType="application/vnd.openxmlformats-officedocument.` + kind + `.` + main + `.main+xml"/></Types>`
		part := prefix + "/printerSettings/printerSettings1.bin"
		extra := map[string]string{"[Content_Types].xml": types, part: "Printer\x00\x01\x02"}
		if _, err := detectBytes(t, "file."+format, officeFixture(t, format, extra)); err != nil {
			t.Fatal(format, err)
		}
		for _, change := range []map[string]string{
			{part: "MZ disguised executable"},
			{"[Content_Types].xml": strings.ReplaceAll(types, kind+".printerSettings", "application/octet-stream")},
			{prefix + "/arbitrary.bin": "Printer\x00"},
			{prefix + "/vbaProject.bin": "macro"},
		} {
			copied := map[string]string{}
			for key, value := range extra {
				copied[key] = value
			}
			for key, value := range change {
				copied[key] = value
			}
			if _, err := detectBytes(t, "file."+format, officeFixture(t, format, copied)); err == nil {
				t.Fatal("unsafe exception", format, change)
			}
		}
	}
}
