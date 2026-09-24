-- H3: linked libraries, durable index and work queue. H4 entities remain deferred.
CREATE TABLE libraries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('linked', 'managed', 'hybrid')),
    current_configuration_revision INTEGER NOT NULL DEFAULT 1,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    disabled_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;
CREATE TABLE library_role_assignments (
    user_id TEXT NOT NULL REFERENCES users(id),
    library_id TEXT NOT NULL REFERENCES libraries(id),
    role_id TEXT NOT NULL,
    scope_kind TEXT NOT NULL DEFAULT 'library' CHECK (scope_kind = 'library'),
    PRIMARY KEY (user_id, library_id, role_id),
    FOREIGN KEY (role_id, scope_kind) REFERENCES roles(id, scope_kind)
) STRICT;
CREATE INDEX library_memberships ON library_role_assignments(library_id, user_id);

CREATE TABLE storage_roots (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    storage_source TEXT NOT NULL CHECK (storage_source IN ('linked', 'managed')),
    canonical_path TEXT NOT NULL,
    comparison_key TEXT NOT NULL,
    volume_identity TEXT,
    case_sensitive INTEGER NOT NULL CHECK (case_sensitive IN (0, 1)),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'inaccessible', 'retired', 'superseded')),
    configuration_revision INTEGER NOT NULL DEFAULT 1 CHECK (configuration_revision > 0),
    watch_mode TEXT NOT NULL CHECK (watch_mode IN ('native', 'polling', 'paused')),
    reconcile_interval_seconds INTEGER NOT NULL CHECK (reconcile_interval_seconds > 0),
    last_verified_at TEXT,
    last_error_code TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id, storage_source),
    UNIQUE (id, library_id)
) STRICT;
CREATE UNIQUE INDEX storage_roots_unique_path ON storage_roots(comparison_key)
    WHERE status NOT IN ('retired', 'superseded');
CREATE INDEX storage_roots_library ON storage_roots(library_id, status);

CREATE TABLE storage_root_configuration_versions (
    root_id TEXT NOT NULL REFERENCES storage_roots(id),
    revision INTEGER NOT NULL CHECK (revision > 0),
    canonical_path TEXT NOT NULL,
    comparison_key TEXT NOT NULL,
    configuration_json TEXT NOT NULL CHECK (json_valid(configuration_json)),
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (root_id, revision)
) STRICT;
CREATE TABLE root_operations (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    requested_by TEXT NOT NULL REFERENCES users(id),
    operation_kind TEXT NOT NULL CHECK (operation_kind IN ('add', 'consolidate', 'retire', 'reorganize')),
    state TEXT NOT NULL CHECK (state IN ('planned', 'confirmed', 'running', 'complete', 'failed', 'cancelled')),
    expected_configuration_json TEXT NOT NULL CHECK (json_valid(expected_configuration_json)),
    private_plan_json TEXT NOT NULL CHECK (json_valid(private_plan_json)),
    checkpoint_json TEXT NOT NULL CHECK (json_valid(checkpoint_json)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE TABLE logical_folder_views (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    root_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relative_prefix TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (root_id, library_id) REFERENCES storage_roots(id, library_id),
    UNIQUE (library_id, root_id, relative_prefix)
) STRICT;

CREATE TABLE physical_files (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    storage_source TEXT NOT NULL CHECK (storage_source IN ('linked', 'managed')),
    os_identity_key TEXT,
    availability TEXT NOT NULL CHECK (availability IN ('available', 'missing', 'unknown', 'staged')),
    integrity_status TEXT NOT NULL CHECK (integrity_status IN ('unknown', 'verified', 'changed')),
    primary_location_id TEXT,
    current_content_version_id TEXT,
    indexed_extraction_id TEXT,
    indexed_at TEXT,
    extraction_freshness TEXT NOT NULL DEFAULT 'none' CHECK (extraction_freshness IN ('none', 'current', 'stale')),
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id, storage_source),
    UNIQUE (id, library_id),
    CHECK (availability <> 'staged' OR storage_source = 'managed'),
    FOREIGN KEY (primary_location_id, id) REFERENCES physical_file_locations(id, physical_file_id)
        DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (current_content_version_id, id) REFERENCES content_versions(id, physical_file_id)
        DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (indexed_extraction_id, id) REFERENCES extraction_runs(id, physical_file_id)
        DEFERRABLE INITIALLY DEFERRED
) STRICT;
CREATE UNIQUE INDEX physical_files_os_identity ON physical_files(os_identity_key) WHERE os_identity_key IS NOT NULL;
CREATE INDEX physical_files_library_availability ON physical_files(library_id, availability);
CREATE TRIGGER physical_files_source_immutable BEFORE UPDATE OF storage_source ON physical_files
WHEN OLD.storage_source <> NEW.storage_source
BEGIN SELECT RAISE(ABORT, 'storage_source is immutable'); END;

CREATE TABLE physical_file_locations (
    id TEXT PRIMARY KEY,
    physical_file_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    storage_source TEXT NOT NULL,
    root_id TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    canonical_path TEXT NOT NULL,
    comparison_key TEXT NOT NULL,
    root_configuration_revision INTEGER NOT NULL,
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    retired_at TEXT,
    UNIQUE (id, physical_file_id),
    FOREIGN KEY (physical_file_id, library_id, storage_source)
        REFERENCES physical_files(id, library_id, storage_source),
    FOREIGN KEY (root_id, library_id, storage_source)
        REFERENCES storage_roots(id, library_id, storage_source)
) STRICT;
CREATE UNIQUE INDEX physical_locations_current_path ON physical_file_locations(comparison_key) WHERE retired_at IS NULL;
CREATE INDEX physical_locations_explorer ON physical_file_locations(root_id, relative_path);
CREATE INDEX physical_locations_file ON physical_file_locations(physical_file_id, retired_at);

CREATE TABLE content_versions (
    id TEXT PRIMARY KEY,
    physical_file_id TEXT NOT NULL REFERENCES physical_files(id),
    generation INTEGER NOT NULL CHECK (generation > 0),
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    os_created_at TEXT,
    os_modified_at TEXT NOT NULL,
    sha256 TEXT CHECK (sha256 IS NULL OR (length(sha256) = 64 AND sha256 NOT GLOB '*[^0-9a-f]*')),
    observed_change_key TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    change_origin TEXT NOT NULL CHECK (change_origin IN ('scan', 'external', 'upload', 'materialization', 'restore')),
    UNIQUE (id, physical_file_id),
    UNIQUE (physical_file_id, generation),
    UNIQUE (physical_file_id, observed_change_key)
) STRICT;
-- No UNIQUE de hash: contenido idéntico puede existir en archivos independientes.
CREATE INDEX content_versions_hash ON content_versions(sha256) WHERE sha256 IS NOT NULL;

CREATE TABLE extraction_runs (
    id TEXT PRIMARY KEY,
    physical_file_id TEXT NOT NULL REFERENCES physical_files(id),
    content_version_id TEXT NOT NULL,
    extractor_revision TEXT NOT NULL,
    language_codes TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'complete', 'failed', 'superseded')),
    page_count INTEGER CHECK (page_count >= 0),
    completed_at TEXT,
    error_code TEXT,
    UNIQUE (id, physical_file_id),
    FOREIGN KEY (content_version_id, physical_file_id) REFERENCES content_versions(id, physical_file_id)
) STRICT;
CREATE TABLE extraction_pages (
    id INTEGER PRIMARY KEY,
    extraction_id TEXT NOT NULL REFERENCES extraction_runs(id),
    page_number INTEGER NOT NULL CHECK (page_number > 0),
    extraction_method TEXT NOT NULL CHECK (extraction_method IN ('native', 'ocr', 'empty')),
    page_text TEXT NOT NULL,
    UNIQUE (extraction_id, page_number)
) STRICT;

CREATE TABLE documents (
 id TEXT PRIMARY KEY, library_id TEXT NOT NULL REFERENCES libraries(id),
 physical_file_id TEXT NOT NULL UNIQUE, title TEXT NOT NULL DEFAULT '',
 original_filename TEXT NOT NULL, filename_search_key TEXT NOT NULL,
 approval_status TEXT NOT NULL DEFAULT 'draft' CHECK(approval_status IN ('draft','pending_review','materializing','approved','rejected','cancelled','needs_review')),
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision > 0), deleted_at TEXT, created_at TEXT NOT NULL,
 UNIQUE(id, library_id), FOREIGN KEY(physical_file_id,library_id) REFERENCES physical_files(id,library_id)
) STRICT;
CREATE TABLE indexed_pages (
 id INTEGER PRIMARY KEY, physical_file_id TEXT NOT NULL REFERENCES physical_files(id),
 extraction_id TEXT NOT NULL REFERENCES extraction_runs(id), page_number INTEGER NOT NULL,
 page_text TEXT NOT NULL, UNIQUE(physical_file_id,page_number)
) STRICT;
CREATE VIRTUAL TABLE pages_fts USING fts5(page_text,content='indexed_pages',content_rowid='id',tokenize='unicode61 remove_diacritics 2');
CREATE TRIGGER indexed_pages_insert AFTER INSERT ON indexed_pages BEGIN INSERT INTO pages_fts(rowid,page_text) VALUES(new.id,new.page_text); END;
CREATE TRIGGER indexed_pages_delete AFTER DELETE ON indexed_pages BEGIN INSERT INTO pages_fts(pages_fts,rowid,page_text) VALUES('delete',old.id,old.page_text); END;
CREATE TRIGGER indexed_pages_update AFTER UPDATE ON indexed_pages BEGIN
 INSERT INTO pages_fts(pages_fts,rowid,page_text) VALUES('delete',old.id,old.page_text);
 INSERT INTO pages_fts(rowid,page_text) VALUES(new.id,new.page_text); END;
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    library_id TEXT REFERENCES libraries(id),
    physical_file_id TEXT REFERENCES physical_files(id),
    job_type TEXT NOT NULL,
    target_version TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'retry_wait', 'succeeded', 'failed', 'paused', 'cancelled')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 5),
    max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 5),
    available_at TEXT NOT NULL,
    lease_owner TEXT,
    lease_expires_at TEXT,
    fencing_token INTEGER NOT NULL DEFAULT 0,
    last_error_code TEXT,
    retry_of_job_id TEXT REFERENCES jobs(id),
    created_at TEXT NOT NULL,
    CHECK (attempt_count <= max_attempts)
) STRICT;
CREATE INDEX jobs_claim ON jobs(status, available_at, created_at);
CREATE UNIQUE INDEX jobs_one_running_file ON jobs(physical_file_id)
    WHERE status = 'running' AND physical_file_id IS NOT NULL;
CREATE TABLE job_attempts (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id),
    attempt_number INTEGER NOT NULL CHECK (attempt_number BETWEEN 1 AND 5),
    fencing_token INTEGER NOT NULL,
    started_at TEXT NOT NULL,
    finished_at TEXT,
    error_code TEXT,
    UNIQUE (job_id, attempt_number)
) STRICT;
CREATE TABLE root_scans (
    id TEXT PRIMARY KEY,
    root_id TEXT NOT NULL REFERENCES storage_roots(id),
    configuration_revision INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('running', 'paused', 'complete', 'failed', 'superseded')),
    checkpoint_json TEXT NOT NULL CHECK (json_valid(checkpoint_json)),
    directories_seen INTEGER NOT NULL DEFAULT 0,
    files_seen INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    can_confirm_absence INTEGER NOT NULL DEFAULT 0 CHECK (can_confirm_absence IN (0, 1))
) STRICT;
CREATE UNIQUE INDEX root_scans_one_running ON root_scans(root_id) WHERE status = 'running';
CREATE TABLE scan_file_observations (
    scan_id TEXT NOT NULL REFERENCES root_scans(id),
    location_id TEXT NOT NULL REFERENCES physical_file_locations(id),
    seen_at TEXT NOT NULL,
    PRIMARY KEY (scan_id, location_id)
) STRICT;

CREATE TABLE search_audit_details (
    event_id TEXT PRIMARY KEY REFERENCES audit_events(id),
    exact_query_text TEXT NOT NULL CHECK (length(exact_query_text) <= 2000),
    search_type TEXT NOT NULL CHECK (search_type IN ('name', 'identifier', 'ocr', 'general')),
    consulted_library_ids_json TEXT NOT NULL CHECK (json_valid(consulted_library_ids_json)),
    applied_filters_json TEXT NOT NULL CHECK (json_valid(applied_filters_json)),
    result_count INTEGER NOT NULL CHECK (result_count >= 0),
    retain_until TEXT
) STRICT;

CREATE TABLE document_notifications (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id),
    content_version_id TEXT REFERENCES content_versions(id),
    event_id TEXT NOT NULL REFERENCES audit_events(id),
    notification_kind TEXT NOT NULL,
    observation_key TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
) STRICT;
CREATE TABLE notification_receipts (
    notification_id TEXT NOT NULL REFERENCES document_notifications(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    acknowledged_at TEXT NOT NULL,
    PRIMARY KEY (notification_id, user_id)
) STRICT;
CREATE TABLE document_history (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id),
    revision INTEGER NOT NULL,
    event_id TEXT NOT NULL REFERENCES audit_events(id),
    snapshot_json TEXT NOT NULL CHECK (json_valid(snapshot_json)),
    UNIQUE (document_id, revision)
) STRICT;

CREATE TABLE idempotency_requests (
    actor_user_id TEXT NOT NULL REFERENCES users(id),
    operation TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    resource_id TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'complete', 'failed')),
    created_at TEXT NOT NULL,
    PRIMARY KEY (actor_user_id, operation, idempotency_key)
) STRICT;


ALTER TABLE root_scans ADD COLUMN last_error_code TEXT;
ALTER TABLE storage_roots ADD COLUMN directory_identity TEXT NOT NULL DEFAULT '';
ALTER TABLE storage_roots ADD COLUMN last_scan_at TEXT;
ALTER TABLE libraries ADD COLUMN ocr_languages TEXT NOT NULL DEFAULT 'spa';
CREATE TABLE root_plans (
 id TEXT PRIMARY KEY, library_id TEXT NOT NULL REFERENCES libraries(id), actor_user_id TEXT NOT NULL REFERENCES users(id),
 server_path TEXT NOT NULL, directory_identity TEXT NOT NULL, relation TEXT NOT NULL,
 related_root_ids_json TEXT NOT NULL CHECK(json_valid(related_root_ids_json)),
 expected_revision INTEGER NOT NULL, expires_at TEXT NOT NULL, committed_root_id TEXT REFERENCES storage_roots(id)
) STRICT;
INSERT INTO roles(id,name,scope_kind) VALUES
 ('library_reader','Lector de biblioteca','library'),('library_manager','Gestor de biblioteca','library'),('library_auditor','Auditor de biblioteca','library');
INSERT INTO role_permissions VALUES ('library_reader','documents.read');
INSERT INTO role_permissions VALUES ('library_reader','documents.download');
INSERT INTO role_permissions VALUES ('library_reader','search.execute');
INSERT INTO role_permissions VALUES ('library_manager','documents.read');
INSERT INTO role_permissions VALUES ('library_manager','documents.download');
INSERT INTO role_permissions VALUES ('library_manager','search.execute');
INSERT INTO role_permissions VALUES ('library_manager','storage.view_paths');
INSERT INTO role_permissions VALUES ('library_manager','libraries.configure');
INSERT INTO role_permissions VALUES ('library_manager','storage.manage_roots');
INSERT INTO role_permissions VALUES ('library_manager','indexing.run');
INSERT INTO role_permissions VALUES ('library_manager','indexing.retry');
INSERT INTO role_permissions VALUES ('library_manager','permissions.manage_library');
INSERT INTO role_permissions VALUES ('library_manager','documents.remove_index');
INSERT INTO role_permissions VALUES ('library_manager','audit.read_library');
INSERT INTO role_permissions VALUES ('library_auditor','audit.read_library');
INSERT INTO role_permissions VALUES ('library_auditor','audit.search_details');

CREATE UNIQUE INDEX jobs_one_running_root ON jobs(target_version) WHERE status='running' AND job_type='scan';
CREATE INDEX root_scans_history ON root_scans(root_id,started_at DESC);
