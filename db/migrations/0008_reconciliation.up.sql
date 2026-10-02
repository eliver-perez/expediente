-- Migrations run with exclusive ownership of the application state. Keep one
-- recoverable request per root and close legacy redundant requests explicitly.
UPDATE job_attempts SET finished_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'),error_code='ROOT_REQUEST_COALESCED'
WHERE finished_at IS NULL AND job_id IN (
 SELECT id FROM (SELECT id,row_number() OVER (PARTITION BY target_version ORDER BY CASE status WHEN 'running' THEN 0 ELSE 1 END,created_at,id) AS position
 FROM jobs WHERE job_type IN ('scan','verify_managed') AND status IN ('queued','running','retry_wait','paused')) WHERE position>1
);
UPDATE jobs SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL,last_error_code='ROOT_REQUEST_COALESCED'
WHERE id IN (
 SELECT id FROM (SELECT id,row_number() OVER (PARTITION BY target_version ORDER BY CASE status WHEN 'running' THEN 0 ELSE 1 END,created_at,id) AS position
 FROM jobs WHERE job_type IN ('scan','verify_managed') AND status IN ('queued','running','retry_wait','paused')) WHERE position>1
);
CREATE UNIQUE INDEX jobs_one_root_verification ON jobs(target_version)
 WHERE job_type IN ('scan','verify_managed') AND status IN ('queued','running','retry_wait','paused');
CREATE TABLE root_watch_recovery (
 root_id TEXT PRIMARY KEY REFERENCES storage_roots(id),
 loss_generation INTEGER NOT NULL DEFAULT 0,
 recovered_generation INTEGER NOT NULL DEFAULT 0
) STRICT;
INSERT INTO root_watch_recovery(root_id,loss_generation)
 SELECT id,1 FROM storage_roots WHERE last_error_code='WATCH_EVENTS_LOST';
ALTER TABLE root_scans ADD COLUMN loss_generation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scan_progress ADD COLUMN changed_files INTEGER NOT NULL DEFAULT 0;
