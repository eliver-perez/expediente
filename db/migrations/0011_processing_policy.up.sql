CREATE TABLE processing_policy (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 paused INTEGER NOT NULL DEFAULT 0 CHECK(paused IN (0,1)),
 maximum_attempts INTEGER NOT NULL DEFAULT 5 CHECK(maximum_attempts BETWEEN 1 AND 5),
 retry_delay_seconds INTEGER NOT NULL DEFAULT 2 CHECK(retry_delay_seconds BETWEEN 1 AND 3600),
 timeout_seconds INTEGER NOT NULL DEFAULT 3600 CHECK(timeout_seconds BETWEEN 10 AND 3600),
 revision INTEGER NOT NULL DEFAULT 1,
 updated_at TEXT NOT NULL
) STRICT;
INSERT INTO processing_policy(singleton,updated_at) VALUES(1,strftime('%Y-%m-%dT%H:%M:%fZ','now'));
ALTER TABLE jobs ADD COLUMN retry_delay_seconds INTEGER NOT NULL DEFAULT 2 CHECK(retry_delay_seconds BETWEEN 1 AND 3600);
ALTER TABLE jobs ADD COLUMN timeout_seconds INTEGER NOT NULL DEFAULT 3600 CHECK(timeout_seconds BETWEEN 10 AND 3600);
