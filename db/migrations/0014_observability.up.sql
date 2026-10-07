-- Operational diagnostics have their own retention; audit_events are untouched.
CREATE TABLE diagnostic_policy (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 retention_days INTEGER NOT NULL DEFAULT 30 CHECK(retention_days BETWEEN 1 AND 365),
 maximum_events INTEGER NOT NULL DEFAULT 5000 CHECK(maximum_events BETWEEN 100 AND 50000),
 revision INTEGER NOT NULL DEFAULT 1
) STRICT;
INSERT INTO diagnostic_policy(singleton) VALUES(1);
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
 context_json TEXT NOT NULL CHECK(json_valid(context_json) AND length(context_json)<=2048),
 occurrences INTEGER NOT NULL DEFAULT 1 CHECK(occurrences>0),
 revision INTEGER NOT NULL DEFAULT 1
) STRICT;
CREATE INDEX diagnostic_events_time ON diagnostic_events(last_occurred_at DESC,id DESC);
CREATE INDEX diagnostic_events_filters ON diagnostic_events(module,severity,status,last_occurred_at DESC);
