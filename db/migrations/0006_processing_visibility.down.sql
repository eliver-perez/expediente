-- Used only by RollbackEmpty, within its guarded all-or-nothing transaction.
DROP INDEX audit_library_type_time;
DROP INDEX audit_library_actor_time;
DROP INDEX jobs_file_version;
DROP INDEX jobs_processing_target;
DROP TABLE root_scan_controls;
DROP TABLE job_controls;
DROP TABLE job_progress;
DROP TABLE scan_errors;
DROP TABLE scan_progress;
DROP TABLE file_scan_cache;
DELETE FROM schema_migrations WHERE name='0006_processing_visibility.up.sql';
