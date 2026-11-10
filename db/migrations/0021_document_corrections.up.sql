-- Correcciones trazables sin alterar archivos físicos ni aprobaciones anteriores.
ALTER TABLE upload_items ADD COLUMN ready_for_incorporation INTEGER NOT NULL DEFAULT 0 CHECK(ready_for_incorporation IN (0,1));
INSERT INTO role_permissions VALUES
 ('library_manager','documents.correct_classification'),
 ('library_manager','documents.request_correction'),
 ('library_manager','documents.replace'),
 ('library_contributor','documents.request_correction');
CREATE TABLE document_corrections (
 id TEXT PRIMARY KEY,
 document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
 operation TEXT NOT NULL CHECK(operation IN ('request','classification','associate','reassign')),
 requested_by TEXT NOT NULL REFERENCES users(id),
 document_revision INTEGER NOT NULL,
 proposed_json TEXT NOT NULL CHECK(json_valid(proposed_json)),
 previous_json TEXT NOT NULL CHECK(json_valid(previous_json)),
 reason TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected','superseded')),
 requested_at TEXT NOT NULL,
 decided_by TEXT REFERENCES users(id),
 decided_at TEXT,
 decision_reason TEXT NOT NULL DEFAULT ''
) STRICT;
CREATE UNIQUE INDEX document_correction_pending ON document_corrections(document_id,operation) WHERE status='pending';
CREATE INDEX document_correction_queue ON document_corrections(library_id,status,requested_at,id);
CREATE TABLE document_replacements (
 id TEXT PRIMARY KEY,
 previous_document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 previous_revision INTEGER NOT NULL,
 new_document_id TEXT NOT NULL UNIQUE REFERENCES documents(id) ON DELETE CASCADE,
 requested_by TEXT NOT NULL REFERENCES users(id),
 reason TEXT NOT NULL,
 previous_json TEXT NOT NULL CHECK(json_valid(previous_json)),
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','completed','cancelled')),
 requested_at TEXT NOT NULL,
 completed_at TEXT,
 authorized_by TEXT REFERENCES users(id),
 CHECK(previous_document_id<>new_document_id)
) STRICT;
CREATE UNIQUE INDEX document_replacement_active ON document_replacements(previous_document_id) WHERE status<>'cancelled';
