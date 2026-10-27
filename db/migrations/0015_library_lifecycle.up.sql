-- Explicit lifecycle journal; no cascade deletes and no paths used as ownership.
INSERT INTO role_permissions VALUES ('installation_admin','libraries.delete');
CREATE TABLE root_retired_documents (
 root_id TEXT NOT NULL REFERENCES storage_roots(id),
 document_id TEXT NOT NULL REFERENCES documents(id),
 removed_revision INTEGER NOT NULL,
 PRIMARY KEY(root_id,document_id)
) STRICT;
CREATE TABLE entity_deletions (
 id TEXT PRIMARY KEY, kind TEXT NOT NULL CHECK(kind IN ('library','root')),
 entity_id TEXT NOT NULL, library_id TEXT NOT NULL, actor_user_id TEXT NOT NULL,
 name TEXT NOT NULL, library_name TEXT NOT NULL, fingerprint TEXT NOT NULL,
 summary_json TEXT NOT NULL CHECK(json_valid(summary_json)),
 state TEXT NOT NULL CHECK(state IN ('planned','cleanup_pending','complete')),
 created_at TEXT NOT NULL, completed_at TEXT
) STRICT;
CREATE TABLE deletion_documents (
 operation_id TEXT NOT NULL REFERENCES entity_deletions(id), document_id TEXT NOT NULL,
 filename TEXT NOT NULL, sha256 TEXT NOT NULL, locations_json TEXT NOT NULL CHECK(json_valid(locations_json)),
 PRIMARY KEY(operation_id,document_id)
) STRICT;
CREATE TABLE deletion_cleanup (
 id TEXT PRIMARY KEY, operation_id TEXT NOT NULL REFERENCES entity_deletions(id),
 kind TEXT NOT NULL CHECK(kind IN ('upload','preview')),
 locator TEXT NOT NULL, container_path TEXT NOT NULL, container_identity TEXT NOT NULL,
 file_identity TEXT NOT NULL, sha256 TEXT NOT NULL, size_bytes INTEGER NOT NULL,
 quarantine TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('pending','complete','attention')),
 error_code TEXT NOT NULL DEFAULT '', attempts INTEGER NOT NULL DEFAULT 0,
 UNIQUE(operation_id,kind,locator)
) STRICT;
CREATE INDEX deletion_cleanup_pending ON deletion_cleanup(status,operation_id);
CREATE INDEX deletion_history ON entity_deletions(library_id,created_at DESC,id);
