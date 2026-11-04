-- Keep existing access-auditor assignments. Event access is assigned explicitly.
DELETE FROM role_permissions WHERE role_id='access_auditor' AND permission_key='audit.read_global';
INSERT INTO roles(id,name,scope_kind) VALUES ('event_auditor','Auditor de eventos','global');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('event_auditor','audit.read_global');
