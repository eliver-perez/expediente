#!/usr/bin/env python3
"""Create tiny, deterministic OOXML/text documents containing synthetic data."""
from pathlib import Path
from zipfile import ZIP_DEFLATED, ZipFile, ZipInfo

destination = Path(__file__).resolve().parents[1] / "testdata" / "documents"
package_ns = "http://schemas.openxmlformats.org/package/2006"
office_ns = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
for extension, main, content_type, parts in [
    ("docx", "word/document.xml", "wordprocessingml.document", {
        "word/document.xml": '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Documento sintético AIBID: conservación del original.</w:t></w:r></w:p></w:body></w:document>',
    }),
    ("xlsx", "xl/workbook.xml", "spreadsheetml.sheet", {
        "xl/workbook.xml": f'<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="{office_ns}"><sheets><sheet name="Prueba" sheetId="1" r:id="rId1"/></sheets></workbook>',
        "xl/_rels/workbook.xml.rels": f'<Relationships xmlns="{package_ns}/relationships"><Relationship Id="rId1" Type="{office_ns}/worksheet" Target="worksheets/sheet1.xml"/></Relationships>',
        "xl/worksheets/sheet1.xml": '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Material sintético</t></is></c><c r="B1"><v>12</v></c></row></sheetData></worksheet>',
    }),
]:
    worksheet_type = '<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>' if extension == "xlsx" else ""
    parts["[Content_Types].xml"] = f'<Types xmlns="{package_ns}/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/{main}" ContentType="application/vnd.openxmlformats-officedocument.{content_type}.main+xml"/>{worksheet_type}</Types>'
    parts["_rels/.rels"] = f'<Relationships xmlns="{package_ns}/relationships"><Relationship Id="rId1" Type="{office_ns}/officeDocument" Target="{main}"/></Relationships>'
    with ZipFile(destination / f"sample.{extension}", "w") as archive:
        for name, text in sorted(parts.items()):
            entry = ZipInfo(name, (2026, 1, 1, 0, 0, 0))
            entry.compress_type = ZIP_DEFLATED
            archive.writestr(entry, text.encode("utf-8"))

(destination / "sample.txt").write_text("Documento sintético AIBID.\nEl original debe conservarse sin cambios.\n", encoding="utf-8")
(destination / "sample.csv").write_text("Material,Cantidad\nArena,12\nCemento,8\n", encoding="utf-8")
