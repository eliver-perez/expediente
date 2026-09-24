-- H2: identity, global authorization and append-only audit.
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

CREATE TABLE bootstrap_state (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    first_admin_id TEXT NOT NULL REFERENCES users(id),
    completed_at TEXT NOT NULL
) STRICT;
CREATE TABLE authentication_throttles (
    scope TEXT NOT NULL CHECK (scope IN ('ip', 'identifier')),
    key_digest TEXT NOT NULL,
    attempt_count INTEGER NOT NULL CHECK (attempt_count >= 0),
    window_started_at TEXT NOT NULL,
    next_allowed_at TEXT NOT NULL,
    PRIMARY KEY (scope, key_digest)
) STRICT;
CREATE INDEX authentication_attempts_identifier ON authentication_attempts(attempted_identifier, occurred_at);

INSERT INTO roles(id,name,scope_kind) VALUES ('installation_admin','Administrador de instalación','global');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','users.manage');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','users.reset_password');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','sessions.revoke');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','libraries.create');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','permissions.manage_global');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','license.manage');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','system.configure');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','backup.manage');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','restore.manage');

INSERT INTO roles(id,name,scope_kind) VALUES ('access_auditor','Auditor de accesos','global');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('access_auditor','sessions.read_all');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('access_auditor','authentication_attempts.read');
INSERT INTO role_permissions(role_id,permission_key) VALUES ('access_auditor','audit.read_global');
