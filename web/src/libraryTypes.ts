import type { ProcessingRun } from './ExtractionDetails';
import type { LibrarySettings } from './organizationTypes';
export interface Library { id: string; name: string; mode: string; revision: number; ocr_languages: string; permissions: string[]; settings: LibrarySettings }
export interface Root { last_scan_result: string; last_scan_completed_at: string; last_scan_changes: number; id: string; library_id: string; server_path?: string; status: string; watch_mode: string; revision: number; reconcile_interval_seconds: number; last_error_code: string; last_scan_at: string; storage_source: string }
export interface FolderView { id: string; name: string; root_id: string; relative_prefix: string }
export interface RootPlan { id: string; relation: string; expected_configuration_revision: number; related_root_ids: string[]; server_path: string; storage_source: string }
export interface Document { format: string; detected_mime: string; extension: string; extension_mismatch: boolean; index_block_reason: string; size_bytes?: number | null; created_at?: string; modified_at?: string; library_name?: string; integrity_status: string; id: string; library_id: string; title: string; original_filename: string; availability: string; approval_status: string; extraction_freshness: string; revision: number; root_id: string; relative_path: string; original_path?: string; sha256: string; page_count: number; unit_count: number; first_text_unit?: number; processing?: ProcessingRun; can_preview_original: boolean; can_download: boolean; matches: { page_number: number; unit_kind: string; context_label: string; segments: { text: string; highlighted: boolean }[] }[]; storage_source: string; case_id: string; case_identifier: string; category_id: string; document_type_id: string; created_by: string; metadata: Record<string, string>; can_classify: boolean; can_associate: boolean; can_reassign: boolean; can_cancel: boolean; category_name: string; document_type_name: string }
export interface SearchInput { query: string; search_type: string; library_ids: string[]; filters: { root_id?: string; view_id?: string; prefix?: string; availability?: string; case_id?: string; category_id?: string; document_type_id?: string; exercise?: string; storage_source?: string; approval_status?: string; unassigned?: boolean }; limit?: number; cursor?: string }
export interface SearchResult { groups: {library_id: string; library_name: string; result_count: number}[]; items: Document[]; result_count: number; next_cursor?: string; consulted_library_ids: string[] }
export interface Job { library_name: string; filename: string; relative_path: string; root_path: string; root_id: string; operation: string; completed_units: number; total_units: number | null; started_at: string; finished_at: string; updated_at: string; cancelling: boolean; id: string; job_type: string; status: string; attempt_count: number; maximum_attempts: number; available_at: string; error_class: string; processing_state: string; last_error_code: string; created_at: string }
export const availabilityLabel: Record<string, string> = { available: 'Disponible', missing: 'No disponible', unknown: 'Raíz sin acceso', staged: 'Temporal privado' };
export const jobLabel: Record<string, string> = { queued: 'En cola', running: 'Procesando', retry_wait: 'Esperando reintento', succeeded: 'Completada', failed: 'Requiere atención', paused: 'Pausada', cancelled: 'Cancelada' };
export const diagnosticLabel: Record<string, string> = {
  FILE_TYPE_BLOCKED: 'Archivo omitido por seguridad: ejecutable, script, paquete o archivo comprimido.',
  FILE_TYPE_UNKNOWN: 'Formato o codificación no reconocidos.',
  FILE_TYPE_MISMATCH: 'La extensión no coincide con el contenido detectado.',
  INVALID_OFFICE_DOCUMENT: 'Estructura Office no válida.',
  DOCUMENT_COMPLEXITY_LIMIT: 'El documento supera los límites de complejidad.',
  FILE_FORMAT_DISABLED: 'Este formato no está permitido por la biblioteca.',
  FILE_NOT_INDEXABLE: 'La biblioteca rechaza archivos que no pueden indexarse.',
  FILE_INDEX_DISABLED: 'Extracción omitida por las reglas de archivos vigentes.',
 PATH_ACCESS_DENIED: 'La cuenta del servicio no tiene permiso de lectura en esta ruta.', PATH_NOT_FOUND: 'La ruta dejó de estar disponible durante el recorrido.', SCAN_PARTIAL: 'Recorrido incompleto: consulta las rutas con error en Procesamiento.', USER_CANCELLED: 'Cancelado por el usuario.',
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
  PROCESSING_INTERNAL_ERROR: 'Se produjo un error interno en este trabajo. Los demás documentos continúan procesándose.', OCR_FAILED: 'No se pudo reconocer el texto con OCR.', TEXT_LIMIT: 'El texto extraído supera el límite por documento.', EXTRACTION_FAILED: 'No se pudo extraer el documento. Revisa el formato, cifrado e idioma OCR.', EXTRACTOR_UNAVAILABLE: 'Falta una herramienta PDF/OCR.',
  PROCESS_TIMEOUT: 'La extracción superó el tiempo permitido.', PAGE_LIMIT: 'El PDF supera el límite de páginas.',
  INTERNAL_LINK_SKIPPED: 'Se omitieron enlaces internos.', WATCH_LIMIT: 'Vigilancia nativa limitada; se utiliza reconciliación.',
  WATCH_EVENTS_LOST: 'Hay eventos pendientes de recuperar. El aviso se retirará tras una reconciliación completa sin errores.', FILE_AUTHORIZATION_CONFLICT: 'El archivo pertenece a otra biblioteca.',
  FILE_IDENTITY_AMBIGUOUS: 'La identidad del archivo requiere revisión.', VERSION_SUPERSEDED: 'Existe una versión más reciente.',
  ROOT_DISABLED: 'La raíz está pausada o retirada.', ROOT_PLAN_STALE: 'Cambió la configuración de la raíz.', PROCESS_RESTARTED: 'Tarea recuperada tras reiniciar el servicio.'
};

export const indexReasonLabel: Record<string, string> = {
  content_rejected: 'Contenido externo no admitido: requiere revisión',
  format_not_indexed: 'Conservado sin indexación de contenido',
  index_size_limit: 'Conservado: supera el tamaño indexable',
  extractor_pending: 'Conservado: extracción aún no disponible',
  unknown_extension: 'Texto con extensión desconocida: sin indexación',
};

export const processingStateLabel: Record<string,string> = {pending:'Pendiente',analysing:'Analizando archivo',validating:'Validando tipo',extracting:'Extrayendo texto',ocr:'Procesando OCR',waiting_ocr:'Esperando turno de OCR',indexing:'Indexando',metadata:'Generando metadatos',completed:'Completado',completed_with_warnings:'Completado con advertencias',error:'Error',unsupported:'No soportado',retrying:'Esperando reintento',paused:'Pausado',cancelled:'Cancelado'};
export const processingErrorLabel: Record<string,string> = {timeout:'Tiempo agotado',invalid_document:'Documento inválido o dañado',unsupported:'Formato o extractor no disponible',limit:'Límite excedido',extractor:'Error del extractor',system:'Error del sistema o del almacenamiento',interrupted:'Interrumpido'};
