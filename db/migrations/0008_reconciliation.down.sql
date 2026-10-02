DROP INDEX jobs_one_root_verification;
DROP TABLE root_watch_recovery;
ALTER TABLE root_scans DROP COLUMN loss_generation;
ALTER TABLE scan_progress DROP COLUMN changed_files;
DELETE FROM schema_migrations WHERE name='0008_reconciliation.up.sql';
