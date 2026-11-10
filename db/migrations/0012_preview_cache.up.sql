CREATE TABLE preview_settings (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
 maximum_cache_mb INTEGER NOT NULL DEFAULT 512 CHECK(maximum_cache_mb BETWEEN 1 AND 20480),
 maximum_idle_days INTEGER NOT NULL DEFAULT 30 CHECK(maximum_idle_days BETWEEN 1 AND 365),
 maximum_source_mb INTEGER NOT NULL DEFAULT 64 CHECK(maximum_source_mb BETWEEN 1 AND 512),
 automatic_cleanup INTEGER NOT NULL DEFAULT 1 CHECK(automatic_cleanup IN (0,1)),
 converter_path TEXT NOT NULL DEFAULT '',
 revision INTEGER NOT NULL DEFAULT 1
) STRICT;
INSERT INTO preview_settings(singleton) VALUES(1);
-- Cache entries never become documents, locations, versions or extraction runs.
CREATE TABLE preview_cache (
 id TEXT PRIMARY KEY,
 document_id TEXT NOT NULL REFERENCES documents(id),
 content_version_id TEXT NOT NULL REFERENCES content_versions(id),
 source_hash TEXT NOT NULL,
 generator_version TEXT NOT NULL,
 requested_by TEXT NOT NULL REFERENCES users(id),
 status TEXT NOT NULL CHECK(status IN ('queued','generating','ready','error')),
 media_type TEXT NOT NULL DEFAULT '',
 size_bytes INTEGER NOT NULL DEFAULT 0 CHECK(size_bytes>=0),
 error_code TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 generated_at TEXT NOT NULL DEFAULT '',
 last_accessed_at TEXT NOT NULL,
 UNIQUE(document_id,source_hash,generator_version)
) STRICT;
CREATE INDEX preview_cache_lru ON preview_cache(status,last_accessed_at,id);
