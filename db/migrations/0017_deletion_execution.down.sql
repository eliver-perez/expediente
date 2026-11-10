DROP INDEX cases_library_exercise;
DROP INDEX case_requirements_case_mandatory;
DROP INDEX entity_deletions_execution;
DROP INDEX entity_deletions_active_entity;
ALTER TABLE entity_deletions DROP COLUMN execution_error;
ALTER TABLE entity_deletions DROP COLUMN execution_status;
