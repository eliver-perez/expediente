-- Avoid a replacement-table scan for every visible document and read pages
-- in library/id order without sorting all documents into a temporary table.
CREATE INDEX document_replacements_completed ON document_replacements(previous_document_id)
 WHERE status='completed';
CREATE INDEX documents_explorer_page ON documents(library_id,id)
 WHERE deleted_at IS NULL;
