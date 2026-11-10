-- Preview files are disposable; normal rollback still requires an empty installation.
DROP TABLE preview_cache;
DROP TABLE preview_settings;
DELETE FROM schema_migrations WHERE name='0012_preview_cache.up.sql';
