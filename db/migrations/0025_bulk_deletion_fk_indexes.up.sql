-- SQLite checks child rows for every parent deleted. These indexes keep
-- large library/root deletions from rescanning the entire child table.
CREATE INDEX jobs_retry_parent ON jobs(retry_of_job_id) WHERE retry_of_job_id IS NOT NULL;
CREATE INDEX indexed_pages_extraction ON indexed_pages(extraction_id);
CREATE INDEX physical_files_indexed_extraction ON physical_files(indexed_extraction_id,id) WHERE indexed_extraction_id IS NOT NULL;
CREATE INDEX scan_file_observations_location ON scan_file_observations(location_id);
