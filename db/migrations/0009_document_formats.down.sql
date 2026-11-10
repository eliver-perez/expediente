-- Refuse downgrade while multiformat versions exist; the old binary cannot
-- interpret them. Restore an upgrade backup instead if rollback is necessary.
CREATE TEMP TABLE format_downgrade_guard(value INTEGER CHECK(value=0));
INSERT INTO format_downgrade_guard SELECT count(*) FROM content_versions WHERE document_format<>'pdf';
DROP TABLE format_downgrade_guard;
ALTER TABLE upload_items DROP COLUMN detection_json;
ALTER TABLE file_scan_cache DROP COLUMN detection_json;
DROP TABLE document_file_skips;
DROP TABLE document_file_settings;
ALTER TABLE content_versions DROP COLUMN index_block_reason;
ALTER TABLE content_versions DROP COLUMN extension_mismatch;
ALTER TABLE content_versions DROP COLUMN original_extension;
ALTER TABLE content_versions DROP COLUMN detected_mime;
ALTER TABLE content_versions DROP COLUMN document_format;
DELETE FROM schema_migrations WHERE name='0009_document_formats.up.sql';
