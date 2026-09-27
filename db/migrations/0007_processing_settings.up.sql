CREATE TABLE processing_settings (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 mode TEXT NOT NULL CHECK(mode IN ('auto','manual')),
 native_workers INTEGER NOT NULL CHECK(native_workers BETWEEN 1 AND 8),
 ocr_workers INTEGER NOT NULL CHECK(ocr_workers BETWEEN 1 AND 4),
 total_workers INTEGER NOT NULL CHECK(total_workers BETWEEN 1 AND 8),
 revision INTEGER NOT NULL DEFAULT 1,
 updated_at TEXT NOT NULL,
 CHECK(native_workers<=total_workers AND ocr_workers<=total_workers)
) STRICT;
CREATE INDEX job_attempts_finished ON job_attempts(finished_at,job_id);
