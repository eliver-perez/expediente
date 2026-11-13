-- AIDEX 1.0.0: initial PostgreSQL schema. No legacy data conversion.
CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;
CREATE TEXT SEARCH CONFIGURATION aidex_search (COPY=pg_catalog.simple);
ALTER TEXT SEARCH CONFIGURATION aidex_search ALTER MAPPING FOR asciiword, word, hword_part, hword_asciipart, asciihword, hword WITH public.unaccent, pg_catalog.simple;



-- Token boundaries match the existing alphanumeric document search. The simple
-- dictionary keeps every word and position; no stemming or stopword removal.
CREATE FUNCTION aidex_search_text(value text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT regexp_replace(public.unaccent('public.unaccent', value) COLLATE pg_catalog.pg_c_utf8, '[^[:alnum:]]', ' ', 'g')
$$;
CREATE FUNCTION aidex_search_vector(value text) RETURNS tsvector
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT to_tsvector('pg_catalog.simple', aidex_search_text(value))
$$;
CREATE FUNCTION aidex_search_query(phrases text[]) RETURNS tsquery
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
 SELECT CASE WHEN bool_or(numnode(query)=0) THEN ''::tsquery
 ELSE coalesce(string_agg('(' || query::text || ')',' & '),'')::tsquery END
 FROM (SELECT phraseto_tsquery('pg_catalog.simple',aidex_search_text(phrase)) AS query FROM unnest(phrases) phrase) parsed
$$;


CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    username_key TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    credential_version BIGINT NOT NULL DEFAULT 1 CHECK (credential_version > 0),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    disabled_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token_digest TEXT NOT NULL UNIQUE,
    csrf_token_digest TEXT NOT NULL,
    credential_version BIGINT NOT NULL,
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
);

CREATE TABLE authentication_attempts (
    id TEXT PRIMARY KEY,
    attempted_identifier TEXT NOT NULL,
    user_id TEXT,
    occurred_at TEXT NOT NULL,
    observed_ip TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('success', 'failed', 'rate_limited')),
    request_id TEXT NOT NULL
);

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    scope_kind TEXT NOT NULL CHECK (scope_kind IN ('global', 'library')),
    UNIQUE (id, scope_kind)
);

CREATE TABLE role_permissions (
    role_id TEXT NOT NULL,
    permission_key TEXT NOT NULL,
    PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE global_role_assignments (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    scope_kind TEXT NOT NULL DEFAULT 'global' CHECK (scope_kind = 'global'),
    PRIMARY KEY (user_id, role_id)
);

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
    details_json TEXT NOT NULL CHECK ((details_json IS JSON)),
    CHECK ((actor_kind = 'user' AND actor_user_id IS NOT NULL) OR
           (actor_kind IN ('system', 'anonymous') AND actor_user_id IS NULL))
);

CREATE TABLE bootstrap_state (
    singleton BIGINT PRIMARY KEY CHECK (singleton = 1),
    first_admin_id TEXT NOT NULL,
    completed_at TEXT NOT NULL
);

CREATE TABLE authentication_throttles (
    scope TEXT NOT NULL CHECK (scope IN ('ip', 'identifier')),
    key_digest TEXT NOT NULL,
    attempt_count BIGINT NOT NULL CHECK (attempt_count >= 0),
    window_started_at TEXT NOT NULL,
    next_allowed_at TEXT NOT NULL,
    PRIMARY KEY (scope, key_digest)
);

CREATE TABLE libraries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('linked', 'managed', 'hybrid')),
    current_configuration_revision BIGINT NOT NULL DEFAULT 1,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    disabled_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    ocr_languages TEXT NOT NULL DEFAULT 'spa',
    settings_json TEXT NOT NULL DEFAULT '{"identifier_label":"Expediente","exercise_enabled":false,"cases_enabled":true,"structure_pattern":"{identificador}/{categoria}","filename_pattern":"original","filename_prefix":"","default_managed_root_id":""}' CHECK((settings_json IS JSON)),
    description TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 500)
);

CREATE TABLE library_role_assignments (
    user_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    scope_kind TEXT NOT NULL DEFAULT 'library' CHECK (scope_kind = 'library'),
    PRIMARY KEY (user_id, library_id, role_id)
);

CREATE TABLE storage_roots (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    storage_source TEXT NOT NULL CHECK (storage_source IN ('linked', 'managed')),
    canonical_path TEXT NOT NULL,
    comparison_key TEXT NOT NULL,
    volume_identity TEXT,
    case_sensitive BIGINT NOT NULL CHECK (case_sensitive IN (0, 1)),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'inaccessible', 'retired', 'superseded')),
    configuration_revision BIGINT NOT NULL DEFAULT 1 CHECK (configuration_revision > 0),
    watch_mode TEXT NOT NULL CHECK (watch_mode IN ('native', 'polling', 'paused')),
    reconcile_interval_seconds BIGINT NOT NULL CHECK (reconcile_interval_seconds > 0),
    last_verified_at TEXT,
    last_error_code TEXT,
    created_at TEXT NOT NULL,
    directory_identity TEXT NOT NULL DEFAULT '',
    last_scan_at TEXT,
    UNIQUE (id, library_id, storage_source),
    UNIQUE (id, library_id)
);

CREATE TABLE storage_root_configuration_versions (
    root_id TEXT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    canonical_path TEXT NOT NULL,
    comparison_key TEXT NOT NULL,
    configuration_json TEXT NOT NULL CHECK ((configuration_json IS JSON)),
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL,
    PRIMARY KEY (root_id, revision)
);

CREATE TABLE root_operations (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    requested_by TEXT NOT NULL,
    operation_kind TEXT NOT NULL CHECK (operation_kind IN ('add', 'consolidate', 'retire', 'reorganize')),
    state TEXT NOT NULL CHECK (state IN ('planned', 'confirmed', 'running', 'complete', 'failed', 'cancelled')),
    expected_configuration_json TEXT NOT NULL CHECK ((expected_configuration_json IS JSON)),
    private_plan_json TEXT NOT NULL CHECK ((private_plan_json IS JSON)),
    checkpoint_json TEXT NOT NULL CHECK ((checkpoint_json IS JSON)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE logical_folder_views (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    root_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relative_prefix TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (library_id, root_id, relative_prefix)
);

CREATE TABLE physical_files (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
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
    CHECK (availability <> 'staged' OR storage_source = 'managed')
);

CREATE TABLE physical_file_locations (
    id TEXT PRIMARY KEY,
    physical_file_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    storage_source TEXT NOT NULL,
    root_id TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    canonical_path TEXT NOT NULL,
    comparison_key TEXT NOT NULL,
    root_configuration_revision BIGINT NOT NULL,
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    retired_at TEXT,
    UNIQUE (id, physical_file_id)
);

CREATE TABLE content_versions (
    id TEXT PRIMARY KEY,
    physical_file_id TEXT NOT NULL,
    generation BIGINT NOT NULL CHECK (generation > 0),
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    os_created_at TEXT,
    os_modified_at TEXT NOT NULL,
    sha256 TEXT CHECK (sha256 IS NULL OR (length(sha256) = 64 AND sha256 !~ '[^0-9a-f]')),
    observed_change_key TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    change_origin TEXT NOT NULL CHECK (change_origin IN ('scan', 'external', 'upload', 'materialization', 'restore')),
    document_format TEXT NOT NULL DEFAULT 'pdf',
    detected_mime TEXT NOT NULL DEFAULT 'application/pdf',
    original_extension TEXT NOT NULL DEFAULT '.pdf',
    extension_mismatch BIGINT NOT NULL DEFAULT 0 CHECK(extension_mismatch IN (0,1)),
    index_block_reason TEXT NOT NULL DEFAULT '',
    UNIQUE (id, physical_file_id),
    UNIQUE (physical_file_id, generation),
    UNIQUE (physical_file_id, observed_change_key)
);

CREATE TABLE extraction_runs (
    id TEXT PRIMARY KEY,
    physical_file_id TEXT NOT NULL,
    content_version_id TEXT NOT NULL,
    extractor_revision TEXT NOT NULL,
    language_codes TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'complete', 'failed', 'superseded')),
    page_count BIGINT CHECK (page_count >= 0),
    completed_at TEXT,
    error_code TEXT,
    extractor_id TEXT NOT NULL DEFAULT '',
    extractor_version TEXT NOT NULL DEFAULT '',
    document_format TEXT NOT NULL DEFAULT 'pdf',
    started_at TEXT,
    result_code TEXT NOT NULL DEFAULT '',
    unit_count BIGINT NOT NULL DEFAULT 0 CHECK(unit_count>=0),
    summary_json TEXT NOT NULL DEFAULT '{}' CHECK((summary_json IS JSON)),
    warnings_json TEXT NOT NULL DEFAULT '[]' CHECK((warnings_json IS JSON)),
    UNIQUE (id, physical_file_id)
);

CREATE TABLE extraction_pages (
    id BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    extraction_id TEXT NOT NULL,
    page_number BIGINT NOT NULL CHECK (page_number > 0),
    extraction_method TEXT NOT NULL CHECK (extraction_method IN ('native', 'ocr', 'empty')),
    page_text TEXT NOT NULL,
    unit_kind TEXT NOT NULL DEFAULT 'page',
    context_label TEXT NOT NULL DEFAULT '',
    context_json TEXT NOT NULL DEFAULT '{}' CHECK((context_json IS JSON)),
    UNIQUE (extraction_id, page_number)
);

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    physical_file_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    original_filename TEXT NOT NULL,
    filename_search_key TEXT NOT NULL,
    approval_status TEXT NOT NULL DEFAULT 'draft' CHECK(approval_status IN ('draft','pending_review','materializing','approved','rejected','cancelled','needs_review')),
    revision BIGINT NOT NULL DEFAULT 1 CHECK(revision > 0),
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    case_id TEXT,
    category_id TEXT,
    document_type_id TEXT,
    created_by TEXT,
    metadata_json TEXT NOT NULL DEFAULT '{}' CHECK((metadata_json IS JSON)),
    approved_content_version_id TEXT,
    approved_by TEXT,
    approved_at TEXT,
    UNIQUE(id, library_id)
);

CREATE TABLE indexed_pages (
    id BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    physical_file_id TEXT NOT NULL,
    extraction_id TEXT NOT NULL,
    page_number BIGINT NOT NULL,
    page_text TEXT NOT NULL
    , search_vector tsvector GENERATED ALWAYS AS (aidex_search_vector(page_text)) STORED,
    UNIQUE(physical_file_id,page_number)
);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    library_id TEXT,
    physical_file_id TEXT,
    job_type TEXT NOT NULL,
    target_version TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    payload_json TEXT NOT NULL CHECK ((payload_json IS JSON)),
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'retry_wait', 'succeeded', 'failed', 'paused', 'cancelled')),
    attempt_count BIGINT NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 5),
    max_attempts BIGINT NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 5),
    available_at TEXT NOT NULL,
    lease_owner TEXT,
    lease_expires_at TEXT,
    fencing_token BIGINT NOT NULL DEFAULT 0,
    last_error_code TEXT,
    retry_of_job_id TEXT,
    created_at TEXT NOT NULL,
    retry_delay_seconds BIGINT NOT NULL DEFAULT 2 CHECK(retry_delay_seconds BETWEEN 1 AND 3600),
    timeout_seconds BIGINT NOT NULL DEFAULT 3600 CHECK(timeout_seconds BETWEEN 10 AND 3600),
    CHECK (attempt_count <= max_attempts)
);

CREATE TABLE job_attempts (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    attempt_number BIGINT NOT NULL CHECK (attempt_number BETWEEN 1 AND 5),
    fencing_token BIGINT NOT NULL,
    started_at TEXT NOT NULL,
    finished_at TEXT,
    error_code TEXT,
    UNIQUE (job_id, attempt_number)
);

CREATE TABLE root_scans (
    id TEXT PRIMARY KEY,
    root_id TEXT NOT NULL,
    configuration_revision BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('running', 'paused', 'complete', 'failed', 'superseded')),
    checkpoint_json TEXT NOT NULL CHECK ((checkpoint_json IS JSON)),
    directories_seen BIGINT NOT NULL DEFAULT 0,
    files_seen BIGINT NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    can_confirm_absence BIGINT NOT NULL DEFAULT 0 CHECK (can_confirm_absence IN (0, 1)),
    last_error_code TEXT,
    loss_generation BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE scan_file_observations (
    scan_id TEXT NOT NULL,
    location_id TEXT NOT NULL,
    seen_at TEXT NOT NULL,
    PRIMARY KEY (scan_id, location_id)
);

CREATE TABLE search_audit_details (
    event_id TEXT PRIMARY KEY,
    exact_query_text TEXT NOT NULL CHECK (length(exact_query_text) <= 2000),
    search_type TEXT NOT NULL CHECK (search_type IN ('name', 'identifier', 'ocr', 'general')),
    consulted_library_ids_json TEXT NOT NULL CHECK ((consulted_library_ids_json IS JSON)),
    applied_filters_json TEXT NOT NULL CHECK ((applied_filters_json IS JSON)),
    result_count BIGINT NOT NULL CHECK (result_count >= 0),
    retain_until TEXT
);

CREATE TABLE document_notifications (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    content_version_id TEXT,
    event_id TEXT NOT NULL,
    notification_kind TEXT NOT NULL,
    observation_key TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE notification_receipts (
    notification_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    acknowledged_at TEXT NOT NULL,
    PRIMARY KEY (notification_id, user_id)
);

CREATE TABLE document_history (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    revision BIGINT NOT NULL,
    event_id TEXT NOT NULL,
    snapshot_json TEXT NOT NULL CHECK ((snapshot_json IS JSON)),
    UNIQUE (document_id, revision)
);

CREATE TABLE idempotency_requests (
    actor_user_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    resource_id TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'complete', 'failed')),
    created_at TEXT NOT NULL,
    PRIMARY KEY (actor_user_id, operation, idempotency_key)
);

CREATE TABLE root_plans (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    actor_user_id TEXT NOT NULL,
    server_path TEXT NOT NULL,
    directory_identity TEXT NOT NULL,
    relation TEXT NOT NULL,
    related_root_ids_json TEXT NOT NULL CHECK((related_root_ids_json IS JSON)),
    expected_revision BIGINT NOT NULL,
    expires_at TEXT NOT NULL,
    committed_root_id TEXT,
    storage_source TEXT NOT NULL DEFAULT 'linked' CHECK(storage_source IN ('linked','managed'))
);

CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    name TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    archived_at TEXT,
    UNIQUE (id, library_id)
);

CREATE TABLE document_types (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    name TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    allows_multiple BIGINT NOT NULL CHECK (allows_multiple IN (0, 1)),
    requires_descriptive_title BIGINT NOT NULL CHECK (requires_descriptive_title IN (0, 1)),
    archived_at TEXT,
    UNIQUE (id, category_id, library_id)
);

CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    name TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (id, library_id)
);

CREATE TABLE template_versions (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id),
    UNIQUE (template_id, revision)
);

CREATE TABLE template_requirements (
    id TEXT PRIMARY KEY,
    template_version_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    document_type_id TEXT NOT NULL,
    required_count BIGINT NOT NULL CHECK (required_count > 0),
    mandatory BIGINT NOT NULL CHECK (mandatory IN (0, 1)),
    allows_multiple BIGINT NOT NULL CHECK (allows_multiple IN (0, 1)),
    UNIQUE (template_version_id, document_type_id),
    CHECK (allows_multiple = 1 OR required_count = 1)
);

CREATE TABLE cases (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    identifier TEXT NOT NULL,
    identifier_key TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    exercise TEXT,
    template_version_id TEXT,
    requirements_initialized_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (id, library_id),
    UNIQUE (library_id, identifier_key)
);

CREATE TABLE case_requirements (
    id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    document_type_id TEXT NOT NULL,
    source_template_requirement_id TEXT,
    required_count BIGINT NOT NULL CHECK (required_count > 0),
    mandatory BIGINT NOT NULL CHECK (mandatory IN (0, 1)),
    allows_multiple BIGINT NOT NULL CHECK (allows_multiple IN (0, 1)),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    UNIQUE (case_id, document_type_id),
    CHECK (allows_multiple = 1 OR required_count = 1)
);

CREATE TABLE library_configuration_versions (
    library_id TEXT NOT NULL,
    revision BIGINT NOT NULL,
    mode TEXT NOT NULL,
    settings_json TEXT NOT NULL CHECK((settings_json IS JSON)),
    created_by TEXT,
    created_at TEXT NOT NULL,
    PRIMARY KEY(library_id,revision)
);

CREATE TABLE upload_batches (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    created_by TEXT NOT NULL,
    client_batch_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(id,library_id),
    UNIQUE(created_by,library_id,client_batch_id)
);

CREATE TABLE upload_items (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    client_file_id TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    document_id TEXT UNIQUE,
    private_temporary_locator TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('receiving','staged','failed','retained','cleaned')),
    error_code TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    retain_until TEXT,
    detection_json TEXT NOT NULL DEFAULT '{}' CHECK((detection_json IS JSON)),
    ready_for_incorporation BIGINT NOT NULL DEFAULT 0 CHECK(ready_for_incorporation IN (0,1)),
    UNIQUE(batch_id,client_file_id)
);

CREATE TABLE review_requests (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    document_revision BIGINT NOT NULL,
    content_version_id TEXT NOT NULL,
    requested_by TEXT NOT NULL,
    assigned_reviewer_id TEXT,
    status TEXT NOT NULL CHECK(status IN ('pending','approved','rejected','cancelled','superseded')),
    requested_at TEXT NOT NULL,
    decided_by TEXT,
    decided_at TEXT,
    reason TEXT NOT NULL DEFAULT '',
    CHECK(status<>'rejected' OR length(trim(reason))>0)
);

CREATE TABLE materializations (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL UNIQUE,
    library_id TEXT NOT NULL,
    physical_file_id TEXT NOT NULL,
    content_version_id TEXT NOT NULL,
    document_revision BIGINT NOT NULL,
    target_root_id TEXT NOT NULL,
    root_revision BIGINT NOT NULL,
    library_configuration_revision BIGINT NOT NULL,
    target_relative_path TEXT NOT NULL,
    target_comparison_key TEXT NOT NULL UNIQUE,
    partial_relative_path TEXT NOT NULL,
    expected_sha256 TEXT NOT NULL,
    expected_size_bytes BIGINT NOT NULL CHECK(expected_size_bytes>=0),
    state TEXT NOT NULL CHECK(state IN ('planned','copying','verified','published','committed','cleaned','failed')),
    published_identity TEXT NOT NULL DEFAULT '',
    approved_by TEXT NOT NULL,
    decision_kind TEXT NOT NULL CHECK(decision_kind IN ('review','direct')),
    review_id TEXT,
    error_code TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE library_document_sequences (
    library_id TEXT PRIMARY KEY,
    next_number BIGINT NOT NULL CHECK(next_number>0)
);

CREATE TABLE license_installation (
    singleton BIGINT PRIMARY KEY CHECK(singleton=1),
    installation_id TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    fingerprint_version TEXT NOT NULL CHECK(fingerprint_version='1'),
    initial_fingerprint_hash TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE license_state (
    singleton BIGINT PRIMARY KEY CHECK(singleton=1),
    current_jws TEXT NOT NULL DEFAULT '',
    deactivated BIGINT NOT NULL DEFAULT 0 CHECK(deactivated IN (0,1)),
    offline_deactivation_pending BIGINT NOT NULL DEFAULT 0 CHECK(offline_deactivation_pending IN (0,1)),
    last_trusted_at TEXT NOT NULL DEFAULT '',
    last_observed_at TEXT NOT NULL DEFAULT '',
    last_contact_at TEXT NOT NULL DEFAULT '',
    last_error_code TEXT NOT NULL DEFAULT '',
    clock_warning BIGINT NOT NULL DEFAULT 0 CHECK(clock_warning IN (0,1))
);

CREATE TABLE license_revision_floors (
    activation_id TEXT PRIMARY KEY,
    license_id TEXT NOT NULL,
    highest_revision BIGINT NOT NULL CHECK(highest_revision>0),
    jws_sha256 TEXT NOT NULL,
    deactivated BIGINT NOT NULL DEFAULT 0 CHECK(deactivated IN (0,1))
);

CREATE TABLE license_artifacts (
    id TEXT PRIMARY KEY,
    direction TEXT NOT NULL CHECK(direction IN ('request','response')),
    action TEXT NOT NULL,
    request_id TEXT NOT NULL,
    contents TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    actor_user_id TEXT,
    created_at TEXT NOT NULL,
    UNIQUE(direction,sha256)
);

CREATE TABLE license_operations (
    request_id TEXT PRIMARY KEY,
    action TEXT NOT NULL CHECK(action IN ('activate','refresh','deactivate')),
    request_digest TEXT NOT NULL,
    request_json TEXT NOT NULL CHECK((request_json IS JSON)),
    status TEXT NOT NULL CHECK(status IN ('prepared','succeeded','failed')),
    error_code TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE file_scan_cache (
    root_id TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    identity_key TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    modified_ns BIGINT NOT NULL,
    sha256 TEXT NOT NULL,
    hashed_at TEXT NOT NULL,
    detection_json TEXT NOT NULL DEFAULT '{}' CHECK((detection_json IS JSON)),
    PRIMARY KEY(root_id,relative_path)
);

CREATE TABLE scan_progress (
    scan_id TEXT PRIMARY KEY,
    current_path TEXT NOT NULL DEFAULT '',
    unchanged_files BIGINT NOT NULL DEFAULT 0,
    hashed_files BIGINT NOT NULL DEFAULT 0,
    excluded_directories BIGINT NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    partial BIGINT NOT NULL DEFAULT 0 CHECK(partial IN (0,1)),
    changed_files BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE scan_errors (
    root_id TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    error_code TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    PRIMARY KEY(root_id,relative_path)
);

CREATE TABLE job_progress (
    job_id TEXT PRIMARY KEY,
    operation TEXT NOT NULL,
    relative_path TEXT NOT NULL DEFAULT '',
    completed_units BIGINT NOT NULL DEFAULT 0,
    total_units BIGINT,
    updated_at TEXT NOT NULL
);

CREATE TABLE job_controls (
    job_id TEXT PRIMARY KEY,
    cancel_requested BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE root_scan_controls (
    root_id TEXT PRIMARY KEY,
    paused BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE processing_settings (
    singleton BIGINT PRIMARY KEY CHECK(singleton=1),
    mode TEXT NOT NULL CHECK(mode IN ('auto','manual')),
    native_workers BIGINT NOT NULL CHECK(native_workers BETWEEN 1 AND 8),
    ocr_workers BIGINT NOT NULL CHECK(ocr_workers BETWEEN 1 AND 4),
    total_workers BIGINT NOT NULL CHECK(total_workers BETWEEN 1 AND 8),
    revision BIGINT NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL,
    CHECK(native_workers<=total_workers AND ocr_workers<=total_workers)
);

CREATE TABLE root_watch_recovery (
    root_id TEXT PRIMARY KEY,
    loss_generation BIGINT NOT NULL DEFAULT 0,
    recovered_generation BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE document_file_settings (
    scope TEXT PRIMARY KEY,
    configuration_json TEXT NOT NULL CHECK((configuration_json IS JSON)),
    revision BIGINT NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL
);

CREATE TABLE document_file_skips (
    root_id TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    error_code TEXT NOT NULL,
    detection_json TEXT NOT NULL CHECK((detection_json IS JSON)),
    observed_at TEXT NOT NULL,
    PRIMARY KEY(root_id,relative_path)
);

CREATE TABLE processing_policy (
    singleton BIGINT PRIMARY KEY CHECK(singleton=1),
    paused BIGINT NOT NULL DEFAULT 0 CHECK(paused IN (0,1)),
    maximum_attempts BIGINT NOT NULL DEFAULT 5 CHECK(maximum_attempts BETWEEN 1 AND 5),
    retry_delay_seconds BIGINT NOT NULL DEFAULT 2 CHECK(retry_delay_seconds BETWEEN 1 AND 3600),
    timeout_seconds BIGINT NOT NULL DEFAULT 3600 CHECK(timeout_seconds BETWEEN 10 AND 3600),
    revision BIGINT NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL
);

CREATE TABLE preview_settings (
    singleton BIGINT PRIMARY KEY CHECK(singleton=1),
    enabled BIGINT NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
    maximum_cache_mb BIGINT NOT NULL DEFAULT 512 CHECK(maximum_cache_mb BETWEEN 1 AND 20480),
    maximum_idle_days BIGINT NOT NULL DEFAULT 30 CHECK(maximum_idle_days BETWEEN 1 AND 365),
    maximum_source_mb BIGINT NOT NULL DEFAULT 64 CHECK(maximum_source_mb BETWEEN 1 AND 512),
    automatic_cleanup BIGINT NOT NULL DEFAULT 1 CHECK(automatic_cleanup IN (0,1)),
    converter_path TEXT NOT NULL DEFAULT '',
    revision BIGINT NOT NULL DEFAULT 1
);

CREATE TABLE preview_cache (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    content_version_id TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    generator_version TEXT NOT NULL,
    requested_by TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('queued','generating','ready','error')),
    media_type TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0 CHECK(size_bytes>=0),
    error_code TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    generated_at TEXT NOT NULL DEFAULT '',
    last_accessed_at TEXT NOT NULL,
    UNIQUE(document_id,source_hash,generator_version)
);

CREATE TABLE advanced_settings (
    scope TEXT PRIMARY KEY,
    configuration_json TEXT NOT NULL CHECK((configuration_json IS JSON)),
    revision BIGINT NOT NULL CHECK(revision>=1),
    updated_at TEXT NOT NULL
);

CREATE TABLE diagnostic_policy (
    singleton BIGINT PRIMARY KEY CHECK(singleton=1),
    retention_days BIGINT NOT NULL DEFAULT 30 CHECK(retention_days BETWEEN 1 AND 365),
    maximum_events BIGINT NOT NULL DEFAULT 5000 CHECK(maximum_events BETWEEN 100 AND 50000),
    revision BIGINT NOT NULL DEFAULT 1
);

CREATE TABLE diagnostic_events (
    id TEXT PRIMARY KEY,
    signature TEXT NOT NULL UNIQUE,
    occurred_at TEXT NOT NULL,
    last_occurred_at TEXT NOT NULL,
    module TEXT NOT NULL,
    code TEXT NOT NULL,
    severity TEXT NOT NULL CHECK(severity IN ('info','warning','error','critical')),
    status TEXT NOT NULL DEFAULT 'open' CHECK(status IN ('open','reviewed')),
    message TEXT NOT NULL,
    context_json TEXT NOT NULL CHECK((context_json IS JSON) AND length(context_json)<=2048),
    occurrences BIGINT NOT NULL DEFAULT 1 CHECK(occurrences>0),
    revision BIGINT NOT NULL DEFAULT 1
);

CREATE TABLE root_retired_documents (
    root_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    removed_revision BIGINT NOT NULL,
    PRIMARY KEY(root_id,document_id)
);

CREATE TABLE entity_deletions (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK(kind IN ('library','root')),
    entity_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    actor_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    library_name TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    summary_json TEXT NOT NULL CHECK((summary_json IS JSON)),
    state TEXT NOT NULL CHECK(state IN ('planned','cleanup_pending','complete')),
    created_at TEXT NOT NULL,
    completed_at TEXT,
    execution_status TEXT NOT NULL DEFAULT 'unconfirmed'
 CHECK(execution_status IN ('unconfirmed','pending','running','completed','failed')),
    execution_error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE deletion_documents (
    operation_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    locations_json TEXT NOT NULL CHECK((locations_json IS JSON)),
    PRIMARY KEY(operation_id,document_id)
);

CREATE TABLE deletion_cleanup (
    id TEXT PRIMARY KEY,
    operation_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK(kind IN ('upload','preview')),
    locator TEXT NOT NULL,
    container_path TEXT NOT NULL,
    container_identity TEXT NOT NULL,
    file_identity TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    quarantine TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending','complete','attention')),
    error_code TEXT NOT NULL DEFAULT '',
    attempts BIGINT NOT NULL DEFAULT 0,
    UNIQUE(operation_id,kind,locator)
);

-- Installation-local acceptance evidence retains historical actor identifiers,
-- just like audit_events, across logical restores from another installation.
CREATE TABLE eula_acceptances (
    id TEXT PRIMARY KEY,
    installation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    version TEXT NOT NULL,
    text_sha256 TEXT NOT NULL,
    accepted_at TEXT NOT NULL
);

CREATE TABLE document_corrections (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    library_id TEXT NOT NULL,
    operation TEXT NOT NULL CHECK(operation IN ('request','classification','associate','reassign')),
    requested_by TEXT NOT NULL,
    document_revision BIGINT NOT NULL,
    proposed_json TEXT NOT NULL CHECK((proposed_json IS JSON)),
    previous_json TEXT NOT NULL CHECK((previous_json IS JSON)),
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected','superseded')),
    requested_at TEXT NOT NULL,
    decided_by TEXT,
    decided_at TEXT,
    decision_reason TEXT NOT NULL DEFAULT ''
);

CREATE TABLE document_replacements (
    id TEXT PRIMARY KEY,
    previous_document_id TEXT NOT NULL,
    previous_revision BIGINT NOT NULL,
    new_document_id TEXT NOT NULL UNIQUE,
    requested_by TEXT NOT NULL,
    reason TEXT NOT NULL,
    previous_json TEXT NOT NULL CHECK((previous_json IS JSON)),
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','completed','cancelled')),
    requested_at TEXT NOT NULL,
    completed_at TEXT,
    authorized_by TEXT,
    CHECK(previous_document_id<>new_document_id)
);

ALTER TABLE sessions ADD FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE authentication_attempts ADD FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE role_permissions ADD FOREIGN KEY (role_id) REFERENCES roles(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE global_role_assignments ADD FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE global_role_assignments ADD FOREIGN KEY (role_id, scope_kind) REFERENCES roles(id, scope_kind) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE bootstrap_state ADD FOREIGN KEY (first_admin_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE library_role_assignments ADD FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE library_role_assignments ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE library_role_assignments ADD FOREIGN KEY (role_id, scope_kind) REFERENCES roles(id, scope_kind) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE storage_roots ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE storage_root_configuration_versions ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE storage_root_configuration_versions ADD FOREIGN KEY (created_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_operations ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_operations ADD FOREIGN KEY (requested_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE logical_folder_views ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE logical_folder_views ADD FOREIGN KEY (root_id, library_id) REFERENCES storage_roots(id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE physical_files ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE physical_files ADD FOREIGN KEY (primary_location_id, id) REFERENCES physical_file_locations(id, physical_file_id)
        DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE physical_files ADD FOREIGN KEY (current_content_version_id, id) REFERENCES content_versions(id, physical_file_id)
        DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE physical_files ADD FOREIGN KEY (indexed_extraction_id, id) REFERENCES extraction_runs(id, physical_file_id)
        DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE physical_file_locations ADD FOREIGN KEY (physical_file_id, library_id, storage_source)
        REFERENCES physical_files(id, library_id, storage_source) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE physical_file_locations ADD FOREIGN KEY (root_id, library_id, storage_source)
        REFERENCES storage_roots(id, library_id, storage_source) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE content_versions ADD FOREIGN KEY (physical_file_id) REFERENCES physical_files(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE extraction_runs ADD FOREIGN KEY (physical_file_id) REFERENCES physical_files(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE extraction_runs ADD FOREIGN KEY (content_version_id, physical_file_id) REFERENCES content_versions(id, physical_file_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE extraction_pages ADD FOREIGN KEY (extraction_id) REFERENCES extraction_runs(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (case_id) REFERENCES cases(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (category_id) REFERENCES categories(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (document_type_id) REFERENCES document_types(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (created_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (approved_content_version_id) REFERENCES content_versions(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY (approved_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE documents ADD FOREIGN KEY(physical_file_id,library_id) REFERENCES physical_files(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE indexed_pages ADD FOREIGN KEY (physical_file_id) REFERENCES physical_files(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE indexed_pages ADD FOREIGN KEY (extraction_id) REFERENCES extraction_runs(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE jobs ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE jobs ADD FOREIGN KEY (physical_file_id) REFERENCES physical_files(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE jobs ADD FOREIGN KEY (retry_of_job_id) REFERENCES jobs(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE job_attempts ADD FOREIGN KEY (job_id) REFERENCES jobs(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_scans ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE scan_file_observations ADD FOREIGN KEY (scan_id) REFERENCES root_scans(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE scan_file_observations ADD FOREIGN KEY (location_id) REFERENCES physical_file_locations(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE search_audit_details ADD FOREIGN KEY (event_id) REFERENCES audit_events(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_notifications ADD FOREIGN KEY (document_id) REFERENCES documents(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_notifications ADD FOREIGN KEY (content_version_id) REFERENCES content_versions(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_notifications ADD FOREIGN KEY (event_id) REFERENCES audit_events(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE notification_receipts ADD FOREIGN KEY (notification_id) REFERENCES document_notifications(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE notification_receipts ADD FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_history ADD FOREIGN KEY (document_id) REFERENCES documents(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_history ADD FOREIGN KEY (event_id) REFERENCES audit_events(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE idempotency_requests ADD FOREIGN KEY (actor_user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_plans ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_plans ADD FOREIGN KEY (actor_user_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_plans ADD FOREIGN KEY (committed_root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE categories ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_types ADD FOREIGN KEY (category_id, library_id) REFERENCES categories(id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE templates ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE template_versions ADD FOREIGN KEY (template_id, library_id) REFERENCES templates(id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE template_requirements ADD FOREIGN KEY (template_version_id, library_id) REFERENCES template_versions(id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE template_requirements ADD FOREIGN KEY (document_type_id, category_id, library_id) REFERENCES document_types(id, category_id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE cases ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE cases ADD FOREIGN KEY (template_version_id, library_id) REFERENCES template_versions(id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE case_requirements ADD FOREIGN KEY (source_template_requirement_id) REFERENCES template_requirements(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE case_requirements ADD FOREIGN KEY (case_id, library_id) REFERENCES cases(id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE case_requirements ADD FOREIGN KEY (document_type_id, category_id, library_id) REFERENCES document_types(id, category_id, library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE library_configuration_versions ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE library_configuration_versions ADD FOREIGN KEY (created_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE upload_batches ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE upload_batches ADD FOREIGN KEY (created_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE upload_items ADD FOREIGN KEY(batch_id,library_id) REFERENCES upload_batches(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE upload_items ADD FOREIGN KEY(document_id,library_id) REFERENCES documents(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE review_requests ADD FOREIGN KEY (content_version_id) REFERENCES content_versions(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE review_requests ADD FOREIGN KEY (requested_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE review_requests ADD FOREIGN KEY (assigned_reviewer_id) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE review_requests ADD FOREIGN KEY (decided_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE review_requests ADD FOREIGN KEY(document_id,library_id) REFERENCES documents(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY (content_version_id) REFERENCES content_versions(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY (approved_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY (review_id) REFERENCES review_requests(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY(document_id,library_id) REFERENCES documents(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY(physical_file_id,library_id) REFERENCES physical_files(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY(target_root_id,library_id) REFERENCES storage_roots(id,library_id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE materializations ADD FOREIGN KEY(library_id,library_configuration_revision) REFERENCES library_configuration_versions(library_id,revision) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE library_document_sequences ADD FOREIGN KEY (library_id) REFERENCES libraries(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE file_scan_cache ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE scan_progress ADD FOREIGN KEY (scan_id) REFERENCES root_scans(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE scan_errors ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE job_progress ADD FOREIGN KEY (job_id) REFERENCES jobs(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE job_controls ADD FOREIGN KEY (job_id) REFERENCES jobs(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_scan_controls ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_watch_recovery ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_file_skips ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE preview_cache ADD FOREIGN KEY (document_id) REFERENCES documents(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE preview_cache ADD FOREIGN KEY (content_version_id) REFERENCES content_versions(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE preview_cache ADD FOREIGN KEY (requested_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_retired_documents ADD FOREIGN KEY (root_id) REFERENCES storage_roots(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE root_retired_documents ADD FOREIGN KEY (document_id) REFERENCES documents(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE deletion_documents ADD FOREIGN KEY (operation_id) REFERENCES entity_deletions(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE deletion_cleanup ADD FOREIGN KEY (operation_id) REFERENCES entity_deletions(id) DEFERRABLE INITIALLY IMMEDIATE;


ALTER TABLE document_corrections ADD FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_corrections ADD FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_corrections ADD FOREIGN KEY (requested_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_corrections ADD FOREIGN KEY (decided_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_replacements ADD FOREIGN KEY (previous_document_id) REFERENCES documents(id) ON DELETE CASCADE DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_replacements ADD FOREIGN KEY (new_document_id) REFERENCES documents(id) ON DELETE CASCADE DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_replacements ADD FOREIGN KEY (requested_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE document_replacements ADD FOREIGN KEY (authorized_by) REFERENCES users(id) DEFERRABLE INITIALLY IMMEDIATE;

CREATE UNIQUE INDEX sessions_one_open_per_user ON sessions(user_id) WHERE closed_at IS NULL;

CREATE INDEX sessions_history ON sessions(user_id, started_at DESC);

CREATE INDEX authentication_attempts_time ON authentication_attempts(occurred_at DESC);

CREATE INDEX authentication_attempts_rate ON authentication_attempts(observed_ip, occurred_at);

CREATE INDEX audit_time ON audit_events(occurred_at DESC, id);

CREATE INDEX audit_library ON audit_events(library_id, occurred_at DESC);

CREATE INDEX audit_document ON audit_events(document_id, occurred_at DESC);

CREATE INDEX audit_actor ON audit_events(actor_user_id, occurred_at DESC);

CREATE INDEX authentication_attempts_identifier ON authentication_attempts(attempted_identifier, occurred_at);

CREATE INDEX library_memberships ON library_role_assignments(library_id, user_id);

CREATE UNIQUE INDEX storage_roots_unique_path ON storage_roots(comparison_key)
    WHERE status NOT IN ('retired', 'superseded');

CREATE INDEX storage_roots_library ON storage_roots(library_id, status);

CREATE UNIQUE INDEX physical_files_os_identity ON physical_files(os_identity_key) WHERE os_identity_key IS NOT NULL;

CREATE INDEX physical_files_library_availability ON physical_files(library_id, availability);

CREATE UNIQUE INDEX physical_locations_current_path ON physical_file_locations(comparison_key) WHERE retired_at IS NULL;

CREATE INDEX physical_locations_explorer ON physical_file_locations(root_id, relative_path);

CREATE INDEX physical_locations_file ON physical_file_locations(physical_file_id, retired_at);

CREATE INDEX content_versions_hash ON content_versions(sha256) WHERE sha256 IS NOT NULL;

CREATE INDEX jobs_claim ON jobs(status, available_at, created_at);

CREATE UNIQUE INDEX jobs_one_running_file ON jobs(physical_file_id)
    WHERE status = 'running' AND physical_file_id IS NOT NULL;

CREATE UNIQUE INDEX root_scans_one_running ON root_scans(root_id) WHERE status = 'running';

CREATE UNIQUE INDEX jobs_one_running_root ON jobs(target_version) WHERE status='running' AND job_type='scan';

CREATE INDEX root_scans_history ON root_scans(root_id,started_at DESC);

CREATE INDEX documents_case_status ON documents(case_id,document_type_id,approval_status) WHERE deleted_at IS NULL;

CREATE INDEX documents_library_filters ON documents(library_id,category_id,document_type_id) WHERE deleted_at IS NULL;

CREATE INDEX uploads_document ON upload_items(document_id,status);

CREATE UNIQUE INDEX review_one_pending ON review_requests(document_id) WHERE status='pending';

CREATE INDEX review_library_inbox ON review_requests(library_id,status,requested_at,id);

CREATE INDEX license_artifacts_time ON license_artifacts(created_at,id);

CREATE INDEX jobs_processing_target ON jobs(job_type,target_version,status);

CREATE INDEX jobs_file_version ON jobs(physical_file_id,target_version,job_type,created_at DESC);

CREATE INDEX audit_library_actor_time ON audit_events(library_id,actor_user_id,occurred_at DESC,id DESC);

CREATE INDEX audit_library_type_time ON audit_events(library_id,event_type,occurred_at DESC,id DESC);

CREATE INDEX job_attempts_finished ON job_attempts(finished_at,job_id);

CREATE UNIQUE INDEX jobs_one_root_verification ON jobs(target_version)
 WHERE job_type IN ('scan','verify_managed') AND status IN ('queued','running','retry_wait','paused');

CREATE INDEX preview_cache_lru ON preview_cache(status,last_accessed_at,id);

CREATE INDEX diagnostic_events_time ON diagnostic_events(last_occurred_at DESC,id DESC);

CREATE INDEX diagnostic_events_filters ON diagnostic_events(module,severity,status,last_occurred_at DESC);

CREATE INDEX deletion_cleanup_pending ON deletion_cleanup(status,operation_id);

CREATE INDEX deletion_history ON entity_deletions(library_id,created_at DESC,id);

CREATE INDEX eula_installation_version ON eula_acceptances(installation_id,version,text_sha256);

CREATE INDEX entity_deletions_execution ON entity_deletions(execution_status,created_at,id);

CREATE UNIQUE INDEX entity_deletions_active_entity ON entity_deletions(kind,entity_id)
 WHERE execution_status IN ('pending','running');

CREATE INDEX cases_library_exercise ON cases(library_id,exercise) WHERE deleted_at IS NULL;

CREATE INDEX case_requirements_case_mandatory ON case_requirements(case_id,mandatory);

CREATE INDEX documents_dashboard_dates ON documents(created_at,library_id) WHERE deleted_at IS NULL;

CREATE INDEX documents_library_created ON documents(library_id,created_at DESC,id DESC) WHERE deleted_at IS NULL;

CREATE INDEX physical_files_primary_location ON physical_files(primary_location_id) WHERE primary_location_id IS NOT NULL;

CREATE INDEX preview_document ON preview_cache(document_id);

CREATE INDEX physical_files_current_version ON physical_files(current_content_version_id,id) WHERE current_content_version_id IS NOT NULL;

CREATE INDEX documents_approved_version ON documents(approved_content_version_id) WHERE approved_content_version_id IS NOT NULL;

CREATE INDEX extraction_runs_version ON extraction_runs(content_version_id,physical_file_id);

CREATE INDEX notifications_version ON document_notifications(content_version_id) WHERE content_version_id IS NOT NULL;

CREATE INDEX review_requests_version ON review_requests(content_version_id);

CREATE INDEX materializations_version ON materializations(content_version_id);

CREATE INDEX preview_cache_version ON preview_cache(content_version_id);

CREATE UNIQUE INDEX document_correction_pending ON document_corrections(document_id,operation) WHERE status='pending';

CREATE INDEX document_correction_queue ON document_corrections(library_id,status,requested_at,id);

CREATE UNIQUE INDEX document_replacement_active ON document_replacements(previous_document_id) WHERE status<>'cancelled';

CREATE INDEX extraction_pages_ocr_lookup ON extraction_pages(extraction_id)
 WHERE extraction_method IN ('ocr','empty');

CREATE INDEX jobs_library_status_created ON jobs(library_id,status,created_at);

CREATE INDEX scan_errors_recent ON scan_errors(occurred_at DESC,root_id);

CREATE INDEX document_file_skips_recent ON document_file_skips(observed_at DESC,root_id);

CREATE INDEX job_attempts_dashboard_cover ON job_attempts(finished_at,job_id,fencing_token,error_code,started_at)
 WHERE finished_at IS NOT NULL;

CREATE INDEX document_replacements_completed ON document_replacements(previous_document_id)
 WHERE status='completed';

CREATE INDEX documents_explorer_page ON documents(library_id,id)
 WHERE deleted_at IS NULL;

CREATE INDEX jobs_retry_parent ON jobs(retry_of_job_id) WHERE retry_of_job_id IS NOT NULL;

CREATE INDEX indexed_pages_extraction ON indexed_pages(extraction_id);

CREATE INDEX physical_files_indexed_extraction ON physical_files(indexed_extraction_id,id) WHERE indexed_extraction_id IS NOT NULL;

CREATE INDEX scan_file_observations_location ON scan_file_observations(location_id);

INSERT INTO roles(id,name,scope_kind) VALUES ('installation_admin','Administrador de instalación','global');

INSERT INTO roles(id,name,scope_kind) VALUES ('access_auditor','Auditor de accesos','global');

INSERT INTO roles(id,name,scope_kind) VALUES ('library_reader','Lector de biblioteca','library');

INSERT INTO roles(id,name,scope_kind) VALUES ('library_manager','Gestor de biblioteca','library');

INSERT INTO roles(id,name,scope_kind) VALUES ('library_auditor','Auditor de biblioteca','library');

INSERT INTO roles(id,name,scope_kind) VALUES ('library_contributor','Colaborador de biblioteca','library');

INSERT INTO roles(id,name,scope_kind) VALUES ('library_reviewer','Revisor de biblioteca','library');

INSERT INTO roles(id,name,scope_kind) VALUES ('event_auditor','Auditor de eventos','global');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','users.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','users.reset_password');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','sessions.revoke');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','libraries.create');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','permissions.manage_global');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','license.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','system.configure');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','backup.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','restore.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('access_auditor','sessions.read_all');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('access_auditor','authentication_attempts.read');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reader','documents.read');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reader','documents.download');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reader','search.execute');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.read');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.download');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','search.execute');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','storage.view_paths');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','libraries.configure');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','storage.manage_roots');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','indexing.run');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','indexing.retry');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','permissions.manage_library');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.remove_index');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','audit.read_library');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_auditor','audit.read_library');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_auditor','audit.search_details');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.download');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.read');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','search.execute');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reviewer','documents.download');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reviewer','documents.read');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reviewer','search.execute');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reviewer','documents.review');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.upload');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.classify');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.associate');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.cancel_own');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.upload');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.classify');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.associate');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.cancel_own');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.reassign');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','cases.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','requirements.edit');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','catalogs.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','templates.manage');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.submit');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.submit');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.finalize');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reviewer','documents.approve');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_reviewer','documents.reject');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','libraries.delete');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('event_auditor','audit.read_global');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.correct_classification');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.request_correction');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_manager','documents.replace');

INSERT INTO role_permissions(role_id,permission_key) VALUES ('library_contributor','documents.request_correction');

INSERT INTO processing_policy(singleton,paused,maximum_attempts,retry_delay_seconds,timeout_seconds,revision,updated_at) VALUES (1,0,5,2,3600,1,to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));

INSERT INTO preview_settings(singleton,enabled,maximum_cache_mb,maximum_idle_days,maximum_source_mb,automatic_cleanup,converter_path,revision) VALUES (1,1,512,30,64,1,'',1);

INSERT INTO diagnostic_policy(singleton,retention_days,maximum_events,revision) VALUES (1,30,5000,1);

CREATE INDEX indexed_pages_search_gin ON indexed_pages USING gin(search_vector);

CREATE FUNCTION audit_events_immutable_fn() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit is append-only' USING ERRCODE='23514'; END $$;
CREATE TRIGGER audit_events_immutable BEFORE UPDATE OR DELETE ON audit_events FOR EACH ROW EXECUTE FUNCTION audit_events_immutable_fn();

CREATE FUNCTION template_versions_immutable_fn() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'historical configuration is immutable' USING ERRCODE='23514'; END $$;
CREATE TRIGGER template_versions_immutable BEFORE UPDATE ON template_versions FOR EACH ROW EXECUTE FUNCTION template_versions_immutable_fn();

CREATE FUNCTION template_requirements_immutable_fn() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'historical configuration is immutable' USING ERRCODE='23514'; END $$;
CREATE TRIGGER template_requirements_immutable BEFORE UPDATE ON template_requirements FOR EACH ROW EXECUTE FUNCTION template_requirements_immutable_fn();

CREATE FUNCTION library_configuration_versions_immutable_fn() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'historical configuration is immutable' USING ERRCODE='23514'; END $$;
CREATE TRIGGER library_configuration_versions_immutable BEFORE UPDATE ON library_configuration_versions FOR EACH ROW EXECUTE FUNCTION library_configuration_versions_immutable_fn();


CREATE FUNCTION physical_files_source_immutable_fn() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN IF OLD.storage_source <> NEW.storage_source THEN RAISE EXCEPTION 'storage_source is immutable' USING ERRCODE='23514'; END IF; RETURN NEW; END $$;
CREATE TRIGGER physical_files_source_immutable BEFORE UPDATE OF storage_source ON physical_files FOR EACH ROW EXECUTE FUNCTION physical_files_source_immutable_fn();
CREATE FUNCTION documents_classification_fn() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.case_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM cases WHERE id=NEW.case_id AND library_id=NEW.library_id))
 OR (NEW.category_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM categories WHERE id=NEW.category_id AND library_id=NEW.library_id))
 OR (NEW.document_type_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM document_types WHERE id=NEW.document_type_id AND library_id=NEW.library_id AND category_id=NEW.category_id))
 THEN RAISE EXCEPTION 'classification scope mismatch' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER documents_classification BEFORE INSERT OR UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION documents_classification_fn();


CREATE INDEX fk_sessions_0 ON sessions(user_id);

CREATE INDEX fk_authentication_attempts_1 ON authentication_attempts(user_id);

CREATE INDEX fk_role_permissions_2 ON role_permissions(role_id);

CREATE INDEX fk_global_role_assignments_3 ON global_role_assignments(user_id);

CREATE INDEX fk_global_role_assignments_4 ON global_role_assignments(role_id, scope_kind);

CREATE INDEX fk_bootstrap_state_5 ON bootstrap_state(first_admin_id);

CREATE INDEX fk_library_role_assignments_6 ON library_role_assignments(user_id);

CREATE INDEX fk_library_role_assignments_7 ON library_role_assignments(library_id);

CREATE INDEX fk_library_role_assignments_8 ON library_role_assignments(role_id, scope_kind);

CREATE INDEX fk_storage_roots_9 ON storage_roots(library_id);

CREATE INDEX fk_storage_root_configuration_versions_10 ON storage_root_configuration_versions(root_id);

CREATE INDEX fk_storage_root_configuration_versions_11 ON storage_root_configuration_versions(created_by);

CREATE INDEX fk_root_operations_12 ON root_operations(library_id);

CREATE INDEX fk_root_operations_13 ON root_operations(requested_by);

CREATE INDEX fk_logical_folder_views_14 ON logical_folder_views(library_id);

CREATE INDEX fk_logical_folder_views_15 ON logical_folder_views(root_id, library_id);

CREATE INDEX fk_physical_files_16 ON physical_files(library_id);

CREATE INDEX fk_physical_files_17 ON physical_files(primary_location_id, id);

CREATE INDEX fk_physical_files_18 ON physical_files(current_content_version_id, id);

CREATE INDEX fk_physical_files_19 ON physical_files(indexed_extraction_id, id);

CREATE INDEX fk_physical_file_locations_20 ON physical_file_locations(physical_file_id, library_id, storage_source);

CREATE INDEX fk_physical_file_locations_21 ON physical_file_locations(root_id, library_id, storage_source);

CREATE INDEX fk_content_versions_22 ON content_versions(physical_file_id);

CREATE INDEX fk_extraction_runs_23 ON extraction_runs(physical_file_id);

CREATE INDEX fk_extraction_runs_24 ON extraction_runs(content_version_id, physical_file_id);

CREATE INDEX fk_extraction_pages_25 ON extraction_pages(extraction_id);

CREATE INDEX fk_documents_26 ON documents(library_id);

CREATE INDEX fk_documents_27 ON documents(case_id);

CREATE INDEX fk_documents_28 ON documents(category_id);

CREATE INDEX fk_documents_29 ON documents(document_type_id);

CREATE INDEX fk_documents_30 ON documents(created_by);

CREATE INDEX fk_documents_31 ON documents(approved_content_version_id);

CREATE INDEX fk_documents_32 ON documents(approved_by);

CREATE INDEX fk_documents_33 ON documents(physical_file_id,library_id);

CREATE INDEX fk_indexed_pages_34 ON indexed_pages(physical_file_id);

CREATE INDEX fk_indexed_pages_35 ON indexed_pages(extraction_id);

CREATE INDEX fk_jobs_36 ON jobs(library_id);

CREATE INDEX fk_jobs_37 ON jobs(physical_file_id);

CREATE INDEX fk_jobs_38 ON jobs(retry_of_job_id);

CREATE INDEX fk_job_attempts_39 ON job_attempts(job_id);

CREATE INDEX fk_root_scans_40 ON root_scans(root_id);

CREATE INDEX fk_scan_file_observations_41 ON scan_file_observations(scan_id);

CREATE INDEX fk_scan_file_observations_42 ON scan_file_observations(location_id);

CREATE INDEX fk_search_audit_details_43 ON search_audit_details(event_id);

CREATE INDEX fk_document_notifications_44 ON document_notifications(document_id);

CREATE INDEX fk_document_notifications_45 ON document_notifications(content_version_id);

CREATE INDEX fk_document_notifications_46 ON document_notifications(event_id);

CREATE INDEX fk_notification_receipts_47 ON notification_receipts(notification_id);

CREATE INDEX fk_notification_receipts_48 ON notification_receipts(user_id);

CREATE INDEX fk_document_history_49 ON document_history(document_id);

CREATE INDEX fk_document_history_50 ON document_history(event_id);

CREATE INDEX fk_idempotency_requests_51 ON idempotency_requests(actor_user_id);

CREATE INDEX fk_root_plans_52 ON root_plans(library_id);

CREATE INDEX fk_root_plans_53 ON root_plans(actor_user_id);

CREATE INDEX fk_root_plans_54 ON root_plans(committed_root_id);

CREATE INDEX fk_categories_55 ON categories(library_id);

CREATE INDEX fk_document_types_56 ON document_types(category_id, library_id);

CREATE INDEX fk_templates_57 ON templates(library_id);

CREATE INDEX fk_template_versions_58 ON template_versions(template_id, library_id);

CREATE INDEX fk_template_requirements_59 ON template_requirements(template_version_id, library_id);

CREATE INDEX fk_template_requirements_60 ON template_requirements(document_type_id, category_id, library_id);

CREATE INDEX fk_cases_61 ON cases(library_id);

CREATE INDEX fk_cases_62 ON cases(template_version_id, library_id);

CREATE INDEX fk_case_requirements_63 ON case_requirements(source_template_requirement_id);

CREATE INDEX fk_case_requirements_64 ON case_requirements(case_id, library_id);

CREATE INDEX fk_case_requirements_65 ON case_requirements(document_type_id, category_id, library_id);

CREATE INDEX fk_library_configuration_versions_66 ON library_configuration_versions(library_id);

CREATE INDEX fk_library_configuration_versions_67 ON library_configuration_versions(created_by);

CREATE INDEX fk_upload_batches_68 ON upload_batches(library_id);

CREATE INDEX fk_upload_batches_69 ON upload_batches(created_by);

CREATE INDEX fk_upload_items_70 ON upload_items(batch_id,library_id);

CREATE INDEX fk_upload_items_71 ON upload_items(document_id,library_id);

CREATE INDEX fk_review_requests_72 ON review_requests(content_version_id);

CREATE INDEX fk_review_requests_73 ON review_requests(requested_by);

CREATE INDEX fk_review_requests_74 ON review_requests(assigned_reviewer_id);

CREATE INDEX fk_review_requests_75 ON review_requests(decided_by);

CREATE INDEX fk_review_requests_76 ON review_requests(document_id,library_id);

CREATE INDEX fk_materializations_77 ON materializations(content_version_id);

CREATE INDEX fk_materializations_78 ON materializations(approved_by);

CREATE INDEX fk_materializations_79 ON materializations(review_id);

CREATE INDEX fk_materializations_80 ON materializations(document_id,library_id);

CREATE INDEX fk_materializations_81 ON materializations(physical_file_id,library_id);

CREATE INDEX fk_materializations_82 ON materializations(target_root_id,library_id);

CREATE INDEX fk_materializations_83 ON materializations(library_id,library_configuration_revision);

CREATE INDEX fk_library_document_sequences_84 ON library_document_sequences(library_id);

CREATE INDEX fk_file_scan_cache_85 ON file_scan_cache(root_id);

CREATE INDEX fk_scan_progress_86 ON scan_progress(scan_id);

CREATE INDEX fk_scan_errors_87 ON scan_errors(root_id);

CREATE INDEX fk_job_progress_88 ON job_progress(job_id);

CREATE INDEX fk_job_controls_89 ON job_controls(job_id);

CREATE INDEX fk_root_scan_controls_90 ON root_scan_controls(root_id);

CREATE INDEX fk_root_watch_recovery_91 ON root_watch_recovery(root_id);

CREATE INDEX fk_document_file_skips_92 ON document_file_skips(root_id);

CREATE INDEX fk_preview_cache_93 ON preview_cache(document_id);

CREATE INDEX fk_preview_cache_94 ON preview_cache(content_version_id);

CREATE INDEX fk_preview_cache_95 ON preview_cache(requested_by);

CREATE INDEX fk_root_retired_documents_96 ON root_retired_documents(root_id);

CREATE INDEX fk_root_retired_documents_97 ON root_retired_documents(document_id);

CREATE INDEX fk_deletion_documents_98 ON deletion_documents(operation_id);

CREATE INDEX fk_deletion_cleanup_99 ON deletion_cleanup(operation_id);

CREATE INDEX fk_eula_acceptances_100 ON eula_acceptances(user_id);

CREATE INDEX fk_document_corrections_101 ON document_corrections(document_id);

CREATE INDEX fk_document_corrections_102 ON document_corrections(library_id);

CREATE INDEX fk_document_corrections_103 ON document_corrections(requested_by);

CREATE INDEX fk_document_corrections_104 ON document_corrections(decided_by);

CREATE INDEX fk_document_replacements_105 ON document_replacements(previous_document_id);

CREATE INDEX fk_document_replacements_106 ON document_replacements(new_document_id);

CREATE INDEX fk_document_replacements_107 ON document_replacements(requested_by);

CREATE INDEX fk_document_replacements_108 ON document_replacements(authorized_by);

INSERT INTO license_state(singleton) VALUES(1);
