-- A confirmed deletion becomes a durable background operation. Existing
-- lifecycle rows remain unchanged and preserve their audit/cleanup evidence.
ALTER TABLE entity_deletions ADD COLUMN execution_status TEXT NOT NULL DEFAULT 'unconfirmed'
 CHECK(execution_status IN ('unconfirmed','pending','running','completed','failed'));
ALTER TABLE entity_deletions ADD COLUMN execution_error TEXT NOT NULL DEFAULT '';
UPDATE entity_deletions SET execution_status='completed' WHERE state<>'planned';
CREATE INDEX entity_deletions_execution ON entity_deletions(execution_status,created_at,id);
CREATE UNIQUE INDEX entity_deletions_active_entity ON entity_deletions(kind,entity_id)
 WHERE execution_status IN ('pending','running');
CREATE INDEX cases_library_exercise ON cases(library_id,exercise) WHERE deleted_at IS NULL;
CREATE INDEX case_requirements_case_mandatory ON case_requirements(case_id,mandatory);
