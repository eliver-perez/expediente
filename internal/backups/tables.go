package backups

// This is deliberately a closed list. Adding a database table never silently
// makes its contents portable; in particular license and session state cannot
// enter an archive through sqlite_master discovery.
var tables = []string{
	"roles", "role_permissions", "users", "global_role_assignments", "bootstrap_state",
	"libraries", "library_role_assignments", "storage_roots", "storage_root_configuration_versions", "root_operations", "logical_folder_views",
	"categories", "document_types", "templates", "template_versions", "template_requirements", "cases", "case_requirements", "library_configuration_versions",
	"physical_files", "physical_file_locations", "content_versions", "extraction_runs", "extraction_pages", "documents", "indexed_pages",
	"jobs", "job_attempts", "root_scans", "scan_file_observations", "search_audit_details", "document_notifications", "notification_receipts", "document_history", "root_plans",
	"review_requests", "materializations", "library_document_sequences", "upload_batches", "upload_items",
	"processing_settings", "scan_progress", "scan_errors", "job_progress", "job_controls", "root_scan_controls",
	"document_file_settings", "document_file_skips", "processing_policy", "preview_settings", "advanced_settings", "diagnostic_policy",
	"root_retired_documents", "entity_deletions", "deletion_documents", "deletion_cleanup", "audit_events",
}

var tableSet = func() map[string]bool {
	m := map[string]bool{}
	for _, name := range tables {
		m[name] = true
	}
	return m
}()

// Operations with physical identities, paths, active sessions, transient caches,
// license artifacts and private installation state are intentionally omitted.
