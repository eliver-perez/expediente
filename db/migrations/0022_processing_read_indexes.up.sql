-- Live progress and the dashboard must avoid scanning historical pages and jobs.
CREATE INDEX extraction_pages_ocr_lookup ON extraction_pages(extraction_id)
 WHERE extraction_method IN ('ocr','empty');
CREATE INDEX jobs_library_status_created ON jobs(library_id,status,created_at);
CREATE INDEX scan_errors_recent ON scan_errors(occurred_at DESC,root_id);
CREATE INDEX document_file_skips_recent ON document_file_skips(observed_at DESC,root_id);
