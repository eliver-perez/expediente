-- Additive operational state. Existing documents/audit/migrations remain intact.
CREATE TABLE file_scan_cache (
 root_id TEXT NOT NULL REFERENCES storage_roots(id), relative_path TEXT NOT NULL,
 identity_key TEXT NOT NULL, size_bytes INTEGER NOT NULL, modified_ns INTEGER NOT NULL,
 sha256 TEXT NOT NULL, hashed_at TEXT NOT NULL, PRIMARY KEY(root_id,relative_path)
) STRICT;
CREATE TABLE scan_progress (
 scan_id TEXT PRIMARY KEY REFERENCES root_scans(id), current_path TEXT NOT NULL DEFAULT '',
 unchanged_files INTEGER NOT NULL DEFAULT 0, hashed_files INTEGER NOT NULL DEFAULT 0,
 excluded_directories INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL,
 partial INTEGER NOT NULL DEFAULT 0 CHECK(partial IN (0,1))
) STRICT;
CREATE TABLE scan_errors (
 root_id TEXT NOT NULL REFERENCES storage_roots(id), relative_path TEXT NOT NULL,
 error_code TEXT NOT NULL, occurred_at TEXT NOT NULL, PRIMARY KEY(root_id,relative_path)
) STRICT;
CREATE TABLE job_progress (
 job_id TEXT PRIMARY KEY REFERENCES jobs(id), operation TEXT NOT NULL,
 relative_path TEXT NOT NULL DEFAULT '', completed_units INTEGER NOT NULL DEFAULT 0,
 total_units INTEGER, updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE job_controls (
 job_id TEXT PRIMARY KEY REFERENCES jobs(id), cancel_requested INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE TABLE root_scan_controls (
 root_id TEXT PRIMARY KEY REFERENCES storage_roots(id), paused INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE INDEX jobs_processing_target ON jobs(job_type,target_version,status);
CREATE INDEX jobs_file_version ON jobs(physical_file_id,target_version,job_type,created_at DESC);
CREATE INDEX audit_library_actor_time ON audit_events(library_id,actor_user_id,occurred_at DESC,id DESC);
CREATE INDEX audit_library_type_time ON audit_events(library_id,event_type,occurred_at DESC,id DESC);
