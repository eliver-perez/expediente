DROP TABLE document_replacements;
DROP TABLE document_corrections;
ALTER TABLE upload_items DROP COLUMN ready_for_incorporation;
DELETE FROM role_permissions WHERE permission_key IN ('documents.correct_classification','documents.request_correction','documents.replace');
