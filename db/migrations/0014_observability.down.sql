DROP TABLE diagnostic_events;
DROP TABLE diagnostic_policy;
DELETE FROM schema_migrations WHERE name='0014_observability.up.sql';
