CREATE TEMP TABLE processing_policy_guard(empty INTEGER CHECK(empty=0));
INSERT INTO processing_policy_guard SELECT count(*) FROM jobs;
DROP TABLE processing_policy_guard;
ALTER TABLE jobs DROP COLUMN timeout_seconds;
ALTER TABLE jobs DROP COLUMN retry_delay_seconds;
DROP TABLE processing_policy;
DELETE FROM schema_migrations WHERE name='0011_processing_policy.up.sql';
