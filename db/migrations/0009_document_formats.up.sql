-- Existing versions were admitted by the PDF-only pipeline. Preserve every
-- identifier, hash, extraction, relationship and FTS row; no source reads.
ALTER TABLE content_versions ADD COLUMN document_format TEXT NOT NULL DEFAULT 'pdf';
ALTER TABLE content_versions ADD COLUMN detected_mime TEXT NOT NULL DEFAULT 'application/pdf';
ALTER TABLE content_versions ADD COLUMN original_extension TEXT NOT NULL DEFAULT '.pdf';
ALTER TABLE content_versions ADD COLUMN extension_mismatch INTEGER NOT NULL DEFAULT 0 CHECK(extension_mismatch IN (0,1));
ALTER TABLE content_versions ADD COLUMN index_block_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE upload_items ADD COLUMN detection_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(detection_json));

CREATE TABLE document_file_settings (
 scope TEXT PRIMARY KEY,
 configuration_json TEXT NOT NULL CHECK(json_valid(configuration_json)),
 revision INTEGER NOT NULL DEFAULT 1,
 updated_at TEXT NOT NULL
) STRICT;

-- Admission skips are informational, not transient scan errors. They never
-- cause repeated extraction attempts or mutations to a linked original.
CREATE TABLE document_file_skips (
 root_id TEXT NOT NULL REFERENCES storage_roots(id),
 relative_path TEXT NOT NULL,
 error_code TEXT NOT NULL,
 detection_json TEXT NOT NULL CHECK(json_valid(detection_json)),
 observed_at TEXT NOT NULL,
 PRIMARY KEY(root_id,relative_path)
) STRICT;
ALTER TABLE file_scan_cache ADD COLUMN detection_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(detection_json));
