DROP TABLE deletion_cleanup;
DROP TABLE deletion_documents;
DROP TABLE entity_deletions;
DROP TABLE root_retired_documents;
DELETE FROM role_permissions WHERE role_id='installation_admin' AND permission_key='libraries.delete';
