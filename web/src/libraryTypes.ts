import type { LibrarySettings } from './organizationTypes';
export interface Library { id: string; name: string; mode: string; revision: number; ocr_languages: string; permissions: string[]; settings: LibrarySettings }
export interface Root { id: string; library_id: string; server_path?: string; status: string; watch_mode: string; revision: number; reconcile_interval_seconds: number; last_error_code: string; last_scan_at: string; storage_source: string }
export interface FolderView { id: string; name: string; root_id: string; relative_prefix: string }
export interface RootPlan { id: string; relation: string; expected_configuration_revision: number; related_root_ids: string[]; server_path: string; storage_source: string }
export interface Document { integrity_status: string; id: string; library_id: string; title: string; original_filename: string; availability: string; approval_status: string; extraction_freshness: string; revision: number; root_id: string; relative_path: string; original_path?: string; sha256: string; page_count: number; can_preview_original: boolean; can_download: boolean; matches: { page_number: number; segments: { text: string; highlighted: boolean }[] }[]; storage_source: string; case_id: string; case_identifier: string; category_id: string; document_type_id: string; created_by: string; metadata: Record<string, string>; can_classify: boolean; can_associate: boolean; can_reassign: boolean; can_cancel: boolean; category_name: string; document_type_name: string }
export interface SearchInput { query: string; search_type: string; library_ids: string[]; filters: { root_id?: string; view_id?: string; prefix?: string; availability?: string; case_id?: string; category_id?: string; document_type_id?: string; exercise?: string; storage_source?: string; approval_status?: string; unassigned?: boolean }; limit?: number; cursor?: string }
export interface SearchResult { items: Document[]; result_count: number; next_cursor?: string; consulted_library_ids: string[] }
export interface Job { id: string; job_type: string; status: string; attempt_count: number; last_error_code: string; created_at: string }
export const availabilityLabel: Record<string, string> = { available: 'Disponible', missing: 'No disponible', unknown: 'Raíz sin acceso', staged: 'Temporal privado' };
export const jobLabel: Record<string, string> = { queued: 'En cola', running: 'Procesando', retry_wait: 'Esperando reintento', succeeded: 'Completada', failed: 'Requiere atención', paused: 'Pausada', cancelled: 'Cancelada' };
export const diagnosticLabel: Record<string, string> = {
  LICENSE_FEATURE: 'Tarea pausada: se requiere un módulo de licencia.', LICENSE_READ_ONLY: 'Tarea pausada: la licencia está en solo lectura.', LICENSE_RECOVERY_REQUIRED: 'Tarea pausada: activa o recupera la licencia.',
  STORAGE_COLLISION: 'Ya existe otro archivo en el destino. No se sobrescribió.',
  STORAGE_UNAVAILABLE: 'No se puede escribir o comprobar el destino.', STORAGE_SPACE: 'El destino no tiene espacio suficiente.',
  STORAGE_INTEGRITY_FAILED: 'La copia no coincide con la huella esperada. Se conserva el temporal.',
  STORAGE_PUBLICATION_FAILED: 'No se pudo publicar sin sobrescribir. Revisa el destino y sus permisos.',
  STORAGE_SYNC_UNSUPPORTED: 'El destino no permite confirmar la escritura en disco.',
  STORAGE_PATH_INVALID: 'El destino contiene un enlace o una ruta no admitida.',
  INVALID_PDF: 'Se omitió un archivo con extensión PDF cuyo contenido no es un PDF.',
  ROOT_UNAVAILABLE: 'No se puede acceder a la raíz o cambió el disco montado.', SOURCE_UNAVAILABLE: 'No se pudo leer una carpeta o archivo.',
  FILE_UNSTABLE: 'El archivo aún está cambiando. Se volverá a intentar.', FILE_SIZE_LIMIT: 'El archivo supera el tamaño permitido.',
  EXTRACTION_FAILED: 'No se pudo extraer el PDF. Revisa el formato, cifrado e idioma OCR.', EXTRACTOR_UNAVAILABLE: 'Falta una herramienta PDF/OCR.',
  PROCESS_TIMEOUT: 'La extracción superó el tiempo permitido.', PAGE_LIMIT: 'El PDF supera el límite de páginas.',
  INTERNAL_LINK_SKIPPED: 'Se omitieron enlaces internos.', WATCH_LIMIT: 'Vigilancia nativa limitada; se utiliza reconciliación.',
  WATCH_EVENTS_LOST: 'Se perdieron eventos. La reconciliación recupera los cambios.', FILE_AUTHORIZATION_CONFLICT: 'El archivo pertenece a otra biblioteca.',
  FILE_IDENTITY_AMBIGUOUS: 'La identidad del archivo requiere revisión.', VERSION_SUPERSEDED: 'Existe una versión más reciente.',
  ROOT_DISABLED: 'La raíz está pausada o retirada.', ROOT_PLAN_STALE: 'Cambió la configuración de la raíz.', PROCESS_RESTARTED: 'Tarea recuperada tras reiniciar el servicio.'
};
