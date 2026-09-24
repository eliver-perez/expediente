-- Run only in a transaction. Refuse to destroy accounts, attempts or audit evidence.
CREATE TEMP TABLE migration_rollback_guard (record_count INTEGER CHECK (record_count = 0));
INSERT INTO migration_rollback_guard SELECT
    (SELECT count(*) FROM users) + (SELECT count(*) FROM sessions) +
    (SELECT count(*) FROM authentication_attempts) + (SELECT count(*) FROM audit_events) +
    (SELECT count(*) FROM bootstrap_state) + (SELECT count(*) FROM authentication_throttles);
DROP TABLE migration_rollback_guard;
DROP TABLE bootstrap_state;
DROP TABLE authentication_throttles;
DROP TABLE authentication_attempts;
DROP TABLE sessions;
DROP TABLE global_role_assignments;
DROP TABLE role_permissions;
DROP TABLE roles;
DROP TABLE users;
DROP TABLE audit_events;
DELETE FROM schema_migrations WHERE name = '0001_identity.up.sql';
