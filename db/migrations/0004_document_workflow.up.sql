-- H5: review decisions and recoverable publication; no repairs of H4 test data.
CREATE TABLE review_requests (
 id TEXT PRIMARY KEY, document_id TEXT NOT NULL, library_id TEXT NOT NULL,
 document_revision INTEGER NOT NULL, content_version_id TEXT NOT NULL REFERENCES content_versions(id),
 requested_by TEXT NOT NULL REFERENCES users(id), assigned_reviewer_id TEXT REFERENCES users(id),
 status TEXT NOT NULL CHECK(status IN ('pending','approved','rejected','cancelled','superseded')),
 requested_at TEXT NOT NULL, decided_by TEXT REFERENCES users(id), decided_at TEXT, reason TEXT NOT NULL DEFAULT '',
 FOREIGN KEY(document_id,library_id) REFERENCES documents(id,library_id),
 CHECK(status<>'rejected' OR length(trim(reason))>0)
) STRICT;
CREATE UNIQUE INDEX review_one_pending ON review_requests(document_id) WHERE status='pending';
CREATE INDEX review_library_inbox ON review_requests(library_id,status,requested_at,id);
CREATE TABLE materializations (
 id TEXT PRIMARY KEY, document_id TEXT NOT NULL UNIQUE, library_id TEXT NOT NULL,
 physical_file_id TEXT NOT NULL, content_version_id TEXT NOT NULL REFERENCES content_versions(id),
 document_revision INTEGER NOT NULL, target_root_id TEXT NOT NULL, root_revision INTEGER NOT NULL,
 library_configuration_revision INTEGER NOT NULL, target_relative_path TEXT NOT NULL,
 target_comparison_key TEXT NOT NULL UNIQUE, partial_relative_path TEXT NOT NULL,
 expected_sha256 TEXT NOT NULL, expected_size_bytes INTEGER NOT NULL CHECK(expected_size_bytes>=0),
 state TEXT NOT NULL CHECK(state IN ('planned','copying','verified','published','committed','cleaned','failed')),
 published_identity TEXT NOT NULL DEFAULT '', approved_by TEXT NOT NULL REFERENCES users(id),
 decision_kind TEXT NOT NULL CHECK(decision_kind IN ('review','direct')),
 review_id TEXT REFERENCES review_requests(id), error_code TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 FOREIGN KEY(document_id,library_id) REFERENCES documents(id,library_id),
 FOREIGN KEY(physical_file_id,library_id) REFERENCES physical_files(id,library_id),
 FOREIGN KEY(target_root_id,library_id) REFERENCES storage_roots(id,library_id),
 FOREIGN KEY(library_id,library_configuration_revision) REFERENCES library_configuration_versions(library_id,revision)
) STRICT;
ALTER TABLE documents ADD COLUMN approved_content_version_id TEXT REFERENCES content_versions(id);
ALTER TABLE documents ADD COLUMN approved_by TEXT REFERENCES users(id);
ALTER TABLE documents ADD COLUMN approved_at TEXT;
ALTER TABLE upload_items ADD COLUMN retain_until TEXT;
CREATE TABLE library_document_sequences (
 library_id TEXT PRIMARY KEY REFERENCES libraries(id), next_number INTEGER NOT NULL CHECK(next_number>0)
) STRICT;
INSERT INTO role_permissions VALUES ('library_contributor','documents.submit'),('library_manager','documents.submit'),
 ('library_manager','documents.finalize'),('library_reviewer','documents.approve'),('library_reviewer','documents.reject');
