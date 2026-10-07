CREATE TABLE advanced_settings (
 scope TEXT PRIMARY KEY,
 configuration_json TEXT NOT NULL CHECK(json_valid(configuration_json)),
 revision INTEGER NOT NULL CHECK(revision>=1),
 updated_at TEXT NOT NULL
) STRICT;
-- Preserve explicit nondefault languages from installations predating inheritance.
INSERT INTO advanced_settings SELECT 'library:'||id,json_object('ocr_languages',ocr_languages),1,updated_at FROM libraries WHERE ocr_languages<>'spa';
