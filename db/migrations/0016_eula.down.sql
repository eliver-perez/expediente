CREATE TEMP TABLE eula_rollback_guard (value INTEGER CHECK(value=0));
INSERT INTO eula_rollback_guard SELECT count(*) FROM eula_acceptances;
DROP TABLE eula_rollback_guard;
DROP INDEX eula_installation_version;
DROP TABLE eula_acceptances;
DELETE FROM schema_migrations WHERE name='0016_eula.up.sql';
