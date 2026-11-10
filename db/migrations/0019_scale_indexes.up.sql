CREATE INDEX documents_dashboard_dates ON documents(created_at,library_id) WHERE deleted_at IS NULL;
CREATE INDEX documents_library_created ON documents(library_id,created_at DESC,id DESC) WHERE deleted_at IS NULL;
CREATE INDEX physical_files_primary_location ON physical_files(primary_location_id) WHERE primary_location_id IS NOT NULL;
CREATE INDEX preview_document ON preview_cache(document_id);
-- These are required for SQLite's parent-key checks when bulk deletion removes
-- content versions. Without them it scans documents/files for every version.
CREATE INDEX physical_files_current_version ON physical_files(current_content_version_id,id) WHERE current_content_version_id IS NOT NULL;
CREATE INDEX documents_approved_version ON documents(approved_content_version_id) WHERE approved_content_version_id IS NOT NULL;
CREATE INDEX extraction_runs_version ON extraction_runs(content_version_id,physical_file_id);
CREATE INDEX notifications_version ON document_notifications(content_version_id) WHERE content_version_id IS NOT NULL;
CREATE INDEX review_requests_version ON review_requests(content_version_id);
CREATE INDEX materializations_version ON materializations(content_version_id);
CREATE INDEX preview_cache_version ON preview_cache(content_version_id);
