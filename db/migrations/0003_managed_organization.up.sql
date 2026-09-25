-- H4: private uploads and organization, without materialization or review decisions.
CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    name TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    archived_at TEXT,
    UNIQUE (id, library_id)
) STRICT;
CREATE TABLE document_types (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    name TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    allows_multiple INTEGER NOT NULL CHECK (allows_multiple IN (0, 1)),
    requires_descriptive_title INTEGER NOT NULL CHECK (requires_descriptive_title IN (0, 1)),
    archived_at TEXT,
    UNIQUE (id, category_id, library_id),
    FOREIGN KEY (category_id, library_id) REFERENCES categories(id, library_id)
) STRICT;

CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    name TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (id, library_id)
) STRICT;
CREATE TABLE template_versions (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id),
    UNIQUE (template_id, revision),
    FOREIGN KEY (template_id, library_id) REFERENCES templates(id, library_id)
) STRICT;
CREATE TABLE template_requirements (
    id TEXT PRIMARY KEY,
    template_version_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    document_type_id TEXT NOT NULL,
    required_count INTEGER NOT NULL CHECK (required_count > 0),
    mandatory INTEGER NOT NULL CHECK (mandatory IN (0, 1)),
    allows_multiple INTEGER NOT NULL CHECK (allows_multiple IN (0, 1)),
    UNIQUE (template_version_id, document_type_id),
    CHECK (allows_multiple = 1 OR required_count = 1),
    FOREIGN KEY (template_version_id, library_id) REFERENCES template_versions(id, library_id),
    FOREIGN KEY (document_type_id, category_id, library_id) REFERENCES document_types(id, category_id, library_id)
) STRICT;
CREATE TABLE cases (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    identifier TEXT NOT NULL,
    identifier_key TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    exercise TEXT,
    template_version_id TEXT,
    requirements_initialized_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id),
    UNIQUE (library_id, identifier_key),
    FOREIGN KEY (template_version_id, library_id) REFERENCES template_versions(id, library_id)
) STRICT;
CREATE TABLE case_requirements (
    id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    document_type_id TEXT NOT NULL,
    source_template_requirement_id TEXT REFERENCES template_requirements(id),
    required_count INTEGER NOT NULL CHECK (required_count > 0),
    mandatory INTEGER NOT NULL CHECK (mandatory IN (0, 1)),
    allows_multiple INTEGER NOT NULL CHECK (allows_multiple IN (0, 1)),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    UNIQUE (case_id, document_type_id),
    CHECK (allows_multiple = 1 OR required_count = 1),
    FOREIGN KEY (case_id, library_id) REFERENCES cases(id, library_id),
    FOREIGN KEY (document_type_id, category_id, library_id) REFERENCES document_types(id, category_id, library_id)
) STRICT;


ALTER TABLE libraries ADD COLUMN settings_json TEXT NOT NULL DEFAULT '{"identifier_label":"Expediente","exercise_enabled":false,"cases_enabled":true,"structure_pattern":"{identificador}/{categoria}","filename_pattern":"original","filename_prefix":"","default_managed_root_id":""}' CHECK(json_valid(settings_json));
CREATE TABLE library_configuration_versions (
 library_id TEXT NOT NULL REFERENCES libraries(id), revision INTEGER NOT NULL,
 mode TEXT NOT NULL, settings_json TEXT NOT NULL CHECK(json_valid(settings_json)),
 created_by TEXT REFERENCES users(id), created_at TEXT NOT NULL,
 PRIMARY KEY(library_id,revision)
) STRICT;
INSERT INTO library_configuration_versions SELECT id,current_configuration_revision,mode,settings_json,NULL,updated_at FROM libraries;
ALTER TABLE root_plans ADD COLUMN storage_source TEXT NOT NULL DEFAULT 'linked' CHECK(storage_source IN ('linked','managed'));
ALTER TABLE documents ADD COLUMN case_id TEXT REFERENCES cases(id);
ALTER TABLE documents ADD COLUMN category_id TEXT REFERENCES categories(id);
ALTER TABLE documents ADD COLUMN document_type_id TEXT REFERENCES document_types(id);
ALTER TABLE documents ADD COLUMN created_by TEXT REFERENCES users(id);
ALTER TABLE documents ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(metadata_json));
CREATE INDEX documents_case_status ON documents(case_id,document_type_id,approval_status) WHERE deleted_at IS NULL;
CREATE INDEX documents_library_filters ON documents(library_id,category_id,document_type_id) WHERE deleted_at IS NULL;
CREATE TABLE upload_batches (
 id TEXT PRIMARY KEY, library_id TEXT NOT NULL REFERENCES libraries(id),
 created_by TEXT NOT NULL REFERENCES users(id), client_batch_id TEXT NOT NULL, created_at TEXT NOT NULL,
 UNIQUE(id,library_id), UNIQUE(created_by,library_id,client_batch_id)
) STRICT;
CREATE TABLE upload_items (
 id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, library_id TEXT NOT NULL,
 client_file_id TEXT NOT NULL, original_filename TEXT NOT NULL,
 document_id TEXT UNIQUE, private_temporary_locator TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('receiving','staged','failed','retained','cleaned')),
 error_code TEXT NOT NULL DEFAULT '', size_bytes INTEGER NOT NULL DEFAULT 0, sha256 TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL, UNIQUE(batch_id,client_file_id),
 FOREIGN KEY(batch_id,library_id) REFERENCES upload_batches(id,library_id),
 FOREIGN KEY(document_id,library_id) REFERENCES documents(id,library_id)
) STRICT;
CREATE INDEX uploads_document ON upload_items(document_id,status);
INSERT INTO roles VALUES ('library_contributor','Colaborador de biblioteca','library'),('library_reviewer','Revisor de biblioteca','library');
INSERT INTO role_permissions SELECT 'library_contributor',permission_key FROM role_permissions WHERE role_id='library_reader';
INSERT INTO role_permissions SELECT 'library_reviewer',permission_key FROM role_permissions WHERE role_id='library_reader';
INSERT INTO role_permissions VALUES ('library_reviewer','documents.review');
INSERT INTO role_permissions VALUES ('library_manager','documents.upload');
INSERT INTO role_permissions VALUES ('library_manager','documents.classify');
INSERT INTO role_permissions VALUES ('library_manager','documents.associate');
INSERT INTO role_permissions VALUES ('library_manager','documents.cancel_own');
INSERT INTO role_permissions VALUES ('library_contributor','documents.upload');
INSERT INTO role_permissions VALUES ('library_contributor','documents.classify');
INSERT INTO role_permissions VALUES ('library_contributor','documents.associate');
INSERT INTO role_permissions VALUES ('library_contributor','documents.cancel_own');
INSERT INTO role_permissions VALUES ('library_manager','documents.reassign');
INSERT INTO role_permissions VALUES ('library_manager','cases.manage');
INSERT INTO role_permissions VALUES ('library_manager','requirements.edit');
INSERT INTO role_permissions VALUES ('library_manager','catalogs.manage');
INSERT INTO role_permissions VALUES ('library_manager','templates.manage');
CREATE TRIGGER documents_classification_insert BEFORE INSERT ON documents
WHEN (NEW.case_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM cases WHERE id=NEW.case_id AND library_id=NEW.library_id))
 OR (NEW.category_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM categories WHERE id=NEW.category_id AND library_id=NEW.library_id))
 OR (NEW.document_type_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM document_types WHERE id=NEW.document_type_id AND library_id=NEW.library_id AND category_id=NEW.category_id))
BEGIN SELECT RAISE(ABORT,'classification scope mismatch'); END;
CREATE TRIGGER documents_classification_update BEFORE UPDATE ON documents
WHEN (NEW.case_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM cases WHERE id=NEW.case_id AND library_id=NEW.library_id))
 OR (NEW.category_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM categories WHERE id=NEW.category_id AND library_id=NEW.library_id))
 OR (NEW.document_type_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM document_types WHERE id=NEW.document_type_id AND library_id=NEW.library_id AND category_id=NEW.category_id))
BEGIN SELECT RAISE(ABORT,'classification scope mismatch'); END;
CREATE TRIGGER template_versions_immutable BEFORE UPDATE ON template_versions BEGIN SELECT RAISE(ABORT,'historical configuration is immutable'); END;
CREATE TRIGGER template_requirements_immutable BEFORE UPDATE ON template_requirements BEGIN SELECT RAISE(ABORT,'historical configuration is immutable'); END;
CREATE TRIGGER library_configuration_versions_immutable BEFORE UPDATE ON library_configuration_versions BEGIN SELECT RAISE(ABORT,'historical configuration is immutable'); END;
