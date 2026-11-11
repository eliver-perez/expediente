-- Keep recent attempt durations on the index pages used by Dashboard.
CREATE INDEX job_attempts_dashboard_cover ON job_attempts(finished_at,job_id,fencing_token,error_code,started_at)
 WHERE finished_at IS NOT NULL;
