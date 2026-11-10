-- Do not discard an explicit assignment when reverting this migration.
CREATE TEMP TABLE event_auditor_rollback_guard (record_count INTEGER CHECK (record_count = 0));
INSERT INTO event_auditor_rollback_guard SELECT count(*) FROM global_role_assignments WHERE role_id='event_auditor';
DROP TABLE event_auditor_rollback_guard;
DELETE FROM role_permissions WHERE role_id='event_auditor';
DELETE FROM roles WHERE id='event_auditor';
INSERT INTO role_permissions(role_id,permission_key) VALUES ('access_auditor','audit.read_global');
DELETE FROM schema_migrations WHERE name='0020_event_auditor.up.sql';
