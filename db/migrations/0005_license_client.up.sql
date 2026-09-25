-- H6: client-side licensing only; no commercial server entities.
CREATE TABLE license_installation (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1), installation_id TEXT NOT NULL UNIQUE,
 public_key TEXT NOT NULL, fingerprint_version TEXT NOT NULL CHECK(fingerprint_version='1'),
 initial_fingerprint_hash TEXT NOT NULL, created_at TEXT NOT NULL
) STRICT;
CREATE TABLE license_state (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1), current_jws TEXT NOT NULL DEFAULT '',
 deactivated INTEGER NOT NULL DEFAULT 0 CHECK(deactivated IN (0,1)),
 offline_deactivation_pending INTEGER NOT NULL DEFAULT 0 CHECK(offline_deactivation_pending IN (0,1)),
 last_trusted_at TEXT NOT NULL DEFAULT '', last_observed_at TEXT NOT NULL DEFAULT '',
 last_contact_at TEXT NOT NULL DEFAULT '', last_error_code TEXT NOT NULL DEFAULT '',
 clock_warning INTEGER NOT NULL DEFAULT 0 CHECK(clock_warning IN (0,1))
) STRICT;
INSERT INTO license_state(singleton) VALUES(1);
CREATE TABLE license_revision_floors (
 activation_id TEXT PRIMARY KEY, license_id TEXT NOT NULL, highest_revision INTEGER NOT NULL CHECK(highest_revision>0),
 jws_sha256 TEXT NOT NULL, deactivated INTEGER NOT NULL DEFAULT 0 CHECK(deactivated IN (0,1))
) STRICT;
CREATE TABLE license_artifacts (
 id TEXT PRIMARY KEY, direction TEXT NOT NULL CHECK(direction IN ('request','response')),
 action TEXT NOT NULL, request_id TEXT NOT NULL, contents TEXT NOT NULL, sha256 TEXT NOT NULL,
 actor_user_id TEXT, created_at TEXT NOT NULL, UNIQUE(direction,sha256)
) STRICT;
CREATE TABLE license_operations (
 request_id TEXT PRIMARY KEY, action TEXT NOT NULL CHECK(action IN ('activate','refresh','deactivate')),
 request_digest TEXT NOT NULL, request_json TEXT NOT NULL CHECK(json_valid(request_json)),
 status TEXT NOT NULL CHECK(status IN ('prepared','succeeded','failed')),
 error_code TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
) STRICT;
CREATE INDEX license_artifacts_time ON license_artifacts(created_at,id);
