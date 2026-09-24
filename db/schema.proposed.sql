-- H1: propuesta verificable, NO migración del producto.
-- UTC se serializará con precisión fija desde Go. UUID opacos como TEXT.
-- No ejecutar sobre una instalación existente. El validador usa una DB temporal.
PRAGMA foreign_keys = ON;

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    username_key TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    credential_version INTEGER NOT NULL DEFAULT 1 CHECK (credential_version > 0),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    disabled_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    token_digest TEXT NOT NULL UNIQUE,
    csrf_token_digest TEXT NOT NULL,
    credential_version INTEGER NOT NULL,
    started_at TEXT NOT NULL,
    last_activity_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    closed_at TEXT,
    close_reason TEXT CHECK (close_reason IN
        ('logout', 'expired', 'new_login', 'password_changed', 'admin_revoked')),
    observed_ip TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    CHECK ((closed_at IS NULL AND close_reason IS NULL) OR
           (closed_at IS NOT NULL AND close_reason IS NOT NULL))
) STRICT;
CREATE UNIQUE INDEX sessions_one_open_per_user ON sessions(user_id) WHERE closed_at IS NULL;
CREATE INDEX sessions_history ON sessions(user_id, started_at DESC);

CREATE TABLE authentication_attempts (
    id TEXT PRIMARY KEY,
    attempted_identifier TEXT NOT NULL,
    user_id TEXT REFERENCES users(id),
    occurred_at TEXT NOT NULL,
    observed_ip TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('success', 'failed', 'rate_limited')),
    request_id TEXT NOT NULL
) STRICT;
CREATE INDEX authentication_attempts_time ON authentication_attempts(occurred_at DESC);
CREATE INDEX authentication_attempts_rate ON authentication_attempts(observed_ip, occurred_at);

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    scope_kind TEXT NOT NULL CHECK (scope_kind IN ('global', 'library')),
    UNIQUE (id, scope_kind)
) STRICT;
CREATE TABLE role_permissions (
    role_id TEXT NOT NULL REFERENCES roles(id),
    permission_key TEXT NOT NULL,
    PRIMARY KEY (role_id, permission_key)
) STRICT;
CREATE TABLE global_role_assignments (
    user_id TEXT NOT NULL REFERENCES users(id),
    role_id TEXT NOT NULL,
    scope_kind TEXT NOT NULL DEFAULT 'global' CHECK (scope_kind = 'global'),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (role_id, scope_kind) REFERENCES roles(id, scope_kind)
) STRICT;

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

CREATE TABLE library_configuration_versions (
    library_id TEXT NOT NULL REFERENCES libraries(id),
    revision INTEGER NOT NULL CHECK (revision > 0),
    identifier_label TEXT NOT NULL,
    exercise_enabled INTEGER NOT NULL CHECK (exercise_enabled IN (0, 1)),
    cases_enabled INTEGER NOT NULL CHECK (cases_enabled IN (0, 1)),
    review_required INTEGER NOT NULL CHECK (review_required IN (0, 1)),
    linked_association_review_required INTEGER NOT NULL CHECK (linked_association_review_required IN (0, 1)),
    structure_pattern TEXT NOT NULL,
    filename_pattern TEXT NOT NULL,
    default_managed_root_id TEXT,
    retention_policy_json TEXT NOT NULL CHECK (json_valid(retention_policy_json)),
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (library_id, revision),
    FOREIGN KEY (default_managed_root_id, library_id) REFERENCES storage_roots(id, library_id)
) STRICT;

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

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    physical_file_id TEXT NOT NULL UNIQUE,
    case_id TEXT,
    category_id TEXT,
    document_type_id TEXT,
    title TEXT NOT NULL DEFAULT '',
    original_filename TEXT NOT NULL,
    filename_search_key TEXT NOT NULL,
    approval_status TEXT NOT NULL CHECK (approval_status IN
        ('draft', 'pending_review', 'materializing', 'approved', 'rejected', 'cancelled', 'needs_review')),
    approved_content_version_id TEXT,
    metadata_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(metadata_json)),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_by TEXT REFERENCES users(id),
    created_at TEXT NOT NULL,
    deleted_at TEXT,
    UNIQUE (id, library_id),
    CHECK (document_type_id IS NULL OR category_id IS NOT NULL),
    FOREIGN KEY (physical_file_id, library_id) REFERENCES physical_files(id, library_id),
    FOREIGN KEY (case_id, library_id) REFERENCES cases(id, library_id),
    FOREIGN KEY (category_id, library_id) REFERENCES categories(id, library_id),
    FOREIGN KEY (document_type_id, category_id, library_id) REFERENCES document_types(id, category_id, library_id),
    FOREIGN KEY (approved_content_version_id, physical_file_id) REFERENCES content_versions(id, physical_file_id)
) STRICT;
CREATE INDEX documents_case_status ON documents(case_id, document_type_id, approval_status) WHERE deleted_at IS NULL;
CREATE INDEX documents_library_filters ON documents(library_id, category_id, document_type_id) WHERE deleted_at IS NULL;
CREATE INDEX documents_filename ON documents(library_id, filename_search_key);

-- Proyección de páginas publicadas. Se reemplaza solo al completar extracción.
CREATE TABLE indexed_pages (
    id INTEGER PRIMARY KEY,
    physical_file_id TEXT NOT NULL REFERENCES physical_files(id),
    extraction_id TEXT NOT NULL,
    page_number INTEGER NOT NULL CHECK (page_number > 0),
    page_text TEXT NOT NULL,
    UNIQUE (physical_file_id, page_number),
    FOREIGN KEY (extraction_id, physical_file_id) REFERENCES extraction_runs(id, physical_file_id),
    FOREIGN KEY (extraction_id, page_number) REFERENCES extraction_pages(extraction_id, page_number)
) STRICT;
CREATE VIRTUAL TABLE page_text_fts USING fts5(
    page_text, content='indexed_pages', content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);
CREATE TRIGGER indexed_pages_insert AFTER INSERT ON indexed_pages BEGIN
    INSERT INTO page_text_fts(rowid, page_text) VALUES (NEW.id, NEW.page_text);
END;
CREATE TRIGGER indexed_pages_delete AFTER DELETE ON indexed_pages BEGIN
    INSERT INTO page_text_fts(page_text_fts, rowid, page_text) VALUES ('delete', OLD.id, OLD.page_text);
END;
CREATE TRIGGER indexed_pages_update AFTER UPDATE ON indexed_pages BEGIN
    INSERT INTO page_text_fts(page_text_fts, rowid, page_text) VALUES ('delete', OLD.id, OLD.page_text);
    INSERT INTO page_text_fts(rowid, page_text) VALUES (NEW.id, NEW.page_text);
END;

CREATE TABLE upload_batches (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id),
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id)
) STRICT;
CREATE TABLE upload_items (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    document_id TEXT NOT NULL UNIQUE,
    private_temporary_locator TEXT,
    status TEXT NOT NULL CHECK (status IN ('receiving', 'staged', 'failed', 'retained', 'cleaned')),
    expires_at TEXT,
    error_code TEXT,
    FOREIGN KEY (batch_id, library_id) REFERENCES upload_batches(id, library_id),
    FOREIGN KEY (document_id, library_id) REFERENCES documents(id, library_id)
) STRICT;
CREATE TABLE review_requests (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id),
    document_revision INTEGER NOT NULL,
    content_version_id TEXT REFERENCES content_versions(id),
    requested_by TEXT NOT NULL REFERENCES users(id),
    assigned_reviewer_id TEXT REFERENCES users(id),
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled', 'superseded')),
    requested_at TEXT NOT NULL,
    decided_by TEXT REFERENCES users(id),
    decided_at TEXT,
    reason TEXT,
    CHECK (status <> 'rejected' OR (reason IS NOT NULL AND length(trim(reason)) > 0))
) STRICT;
CREATE UNIQUE INDEX review_one_pending ON review_requests(document_id) WHERE status = 'pending';
CREATE INDEX review_inbox ON review_requests(assigned_reviewer_id, status, requested_at);

CREATE TABLE materializations (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES documents(id),
    library_id TEXT NOT NULL,
    target_root_id TEXT NOT NULL,
    library_configuration_revision INTEGER NOT NULL,
    target_relative_path TEXT NOT NULL,
    target_comparison_key TEXT NOT NULL UNIQUE,
    expected_sha256 TEXT NOT NULL,
    expected_size_bytes INTEGER NOT NULL CHECK (expected_size_bytes >= 0),
    state TEXT NOT NULL CHECK (state IN ('planned', 'copying', 'verified', 'published', 'committed', 'cleaned', 'failed')),
    approved_by TEXT REFERENCES users(id),
    decision_kind TEXT NOT NULL CHECK (decision_kind IN ('review', 'direct')),
    idempotency_key TEXT NOT NULL UNIQUE,
    error_code TEXT,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (document_id, library_id) REFERENCES documents(id, library_id),
    FOREIGN KEY (target_root_id, library_id) REFERENCES storage_roots(id, library_id),
    FOREIGN KEY (library_id, library_configuration_revision)
        REFERENCES library_configuration_versions(library_id, revision)
) STRICT;
CREATE UNIQUE INDEX materialization_one_active ON materializations(document_id)
    WHERE state NOT IN ('cleaned', 'failed');

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

CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    actor_kind TEXT NOT NULL CHECK (actor_kind IN ('user', 'system', 'anonymous')),
    actor_user_id TEXT,
    session_id TEXT,
    observed_ip TEXT,
    request_id TEXT,
    event_type TEXT NOT NULL,
    library_id TEXT,
    case_id TEXT,
    document_id TEXT,
    details_json TEXT NOT NULL CHECK (json_valid(details_json)),
    CHECK ((actor_kind = 'user' AND actor_user_id IS NOT NULL) OR
           (actor_kind IN ('system', 'anonymous') AND actor_user_id IS NULL))
) STRICT;
-- Identificadores históricos sin FK de borrado: sobrevivir a retenciones/purgas autorizadas.
CREATE INDEX audit_time ON audit_events(occurred_at DESC, id);
CREATE INDEX audit_library ON audit_events(library_id, occurred_at DESC);
CREATE INDEX audit_document ON audit_events(document_id, occurred_at DESC);
CREATE INDEX audit_actor ON audit_events(actor_user_id, occurred_at DESC);
CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events
BEGIN SELECT RAISE(ABORT, 'audit is append-only'); END;
CREATE TRIGGER audit_events_no_delete BEFORE DELETE ON audit_events
BEGIN SELECT RAISE(ABORT, 'audit is append-only'); END;

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

CREATE TABLE installation_identity (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    installation_id TEXT NOT NULL UNIQUE,
    public_key_b64u TEXT NOT NULL,
    private_key_reference TEXT NOT NULL,
    fingerprint_version TEXT NOT NULL,
    fingerprint_hash TEXT NOT NULL,
    last_trusted_validation_at TEXT,
    created_at TEXT NOT NULL
) STRICT;
CREATE TABLE license_activations (
    activation_id TEXT PRIMARY KEY,
    license_id TEXT NOT NULL,
    highest_accepted_revision INTEGER NOT NULL CHECK (highest_accepted_revision > 0),
    accepted_jws TEXT NOT NULL,
    accepted_jws_digest TEXT NOT NULL,
    accepted_at TEXT NOT NULL,
    is_current INTEGER NOT NULL CHECK (is_current IN (0, 1))
) STRICT;
CREATE UNIQUE INDEX license_one_current ON license_activations(is_current) WHERE is_current = 1;
CREATE TABLE license_exchanges (
    request_id TEXT PRIMARY KEY,
    action TEXT NOT NULL CHECK (action IN ('activate', 'refresh', 'renew', 'deactivate')),
    channel TEXT NOT NULL CHECK (channel IN ('online', 'offline')),
    activation_id TEXT,
    artifact_reference TEXT,
    response_digest TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'confirmed', 'failed')),
    created_at TEXT NOT NULL,
    completed_at TEXT
) STRICT;

CREATE TABLE backup_runs (
    id TEXT PRIMARY KEY,
    requested_by TEXT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL CHECK (status IN ('planned', 'running', 'complete', 'failed')),
    private_manifest_reference TEXT,
    consistency_cut_at TEXT,
    started_at TEXT NOT NULL,
    finished_at TEXT,
    error_code TEXT
) STRICT;
