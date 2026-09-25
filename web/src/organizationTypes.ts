export interface LibrarySettings {
  review_managed: boolean; review_linked: boolean; retention_days: number;
  identifier_label: string; exercise_enabled: boolean; cases_enabled: boolean;
  structure_pattern: string; filename_pattern: string; filename_prefix: string; default_managed_root_id: string;
}
export interface CatalogEntry { id: string; name: string; category_id: string; allows_multiple: boolean; requires_descriptive_title: boolean; archived: boolean; revision: number }
export interface RequirementInput { category_id: string; document_type_id: string; required_count: number; mandatory: boolean }
export interface TemplateRequirement extends RequirementInput { id: string; allows_multiple: boolean }
export interface TemplateVersion { id: string; revision: number; requirements: TemplateRequirement[] }
export interface Template { id: string; name: string; versions: TemplateVersion[] }
export interface CaseRecord { id: string; library_id: string; identifier: string; exercise: string; template_version_id: string; requirements_initialized: boolean; revision: number; created_at: string }
export interface CaseRequirement extends TemplateRequirement { revision: number; category_name: string; document_type_name: string; counts: Record<string, number>; available_count: number; missing_count: number }
export interface Requirements { items: CaseRequirement[]; progress_percent: number | null; initialized: boolean }
export interface UploadItem { id: string; document_id: string; original_filename: string; status: string; error_code: string; size_bytes: number }
export interface UploadBatch { id: string; library_id: string; created_by: string; created_at: string; items: UploadItem[] }
export const modeLabel: Record<string, string> = { linked: 'Vinculada', managed: 'Administrada', hybrid: 'Híbrida' };
export const approvalLabel: Record<string, string> = { draft: 'Borrador', pending_review: 'En revisión', approved: 'Aprobado', rejected: 'Rechazado', cancelled: 'Cancelado', materializing: 'Guardando definitivo', needs_review: 'Requiere revisión' };
