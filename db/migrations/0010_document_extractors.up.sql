-- Preserve every existing extraction and FTS row. "page_number" remains the
-- stable unit ordinal internally; unit_kind distinguishes physical PDF pages.
ALTER TABLE extraction_runs ADD COLUMN extractor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE extraction_runs ADD COLUMN extractor_version TEXT NOT NULL DEFAULT '';
ALTER TABLE extraction_runs ADD COLUMN document_format TEXT NOT NULL DEFAULT 'pdf';
ALTER TABLE extraction_runs ADD COLUMN started_at TEXT;
ALTER TABLE extraction_runs ADD COLUMN result_code TEXT NOT NULL DEFAULT '';
ALTER TABLE extraction_runs ADD COLUMN unit_count INTEGER NOT NULL DEFAULT 0 CHECK(unit_count>=0);
ALTER TABLE extraction_runs ADD COLUMN summary_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(summary_json));
ALTER TABLE extraction_runs ADD COLUMN warnings_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(warnings_json));
UPDATE extraction_runs SET extractor_id=CASE WHEN extractor_revision LIKE 'poppler-tesseract-v%' THEN 'poppler-tesseract' ELSE '' END,
 extractor_version=CASE WHEN extractor_revision LIKE 'poppler-tesseract-v%' THEN substr(extractor_revision,20) ELSE '' END,
 result_code=status,unit_count=coalesce(page_count,0),
 summary_json=CASE WHEN page_count IS NOT NULL THEN json_object('pages',page_count) ELSE '{}' END;
ALTER TABLE extraction_pages ADD COLUMN unit_kind TEXT NOT NULL DEFAULT 'page';
ALTER TABLE extraction_pages ADD COLUMN context_label TEXT NOT NULL DEFAULT '';
ALTER TABLE extraction_pages ADD COLUMN context_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(context_json));
