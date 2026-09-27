DROP INDEX job_attempts_finished;
DROP TABLE processing_settings;
DELETE FROM schema_migrations WHERE name='0007_processing_settings.up.sql';
