export interface SearchAuditData {
  query: string;
  type: string;
  libraries: string[];
  filters: Record<string, unknown>;
  category?: string | null;
  document_type?: string | null;
  case_name?: string | null;
  root?: string | null;
  view?: string | null;
}

const scopeNames: Record<string, string> = { general: 'Nombre, expediente y contenido', name: 'Nombre de archivo', identifier: 'Identificador de expediente', ocr: 'Texto nativo y OCR' };
const availabilityNames: Record<string, string> = { available: 'Disponible', missing: 'No disponible', unknown: 'Raíz sin acceso', staged: 'Temporal privado' };
const sourceNames: Record<string, string> = { linked: 'Vinculado', managed: 'Administrado' };
const approvalNames: Record<string, string> = { draft: 'Borrador', pending_review: 'Pendiente de revisión', approved: 'Aprobado', rejected: 'Rechazado', needs_review: 'Requiere revisión', materializing: 'Incorporándose' };
function label(value: unknown, names?: Record<string, string>) { return typeof value === 'string' && value ? names?.[value] || value : ''; }

export function SearchAuditCriteria({ data, found, returned }: { data: SearchAuditData; found?: number; returned?: number }) {
  const filters = data.filters || {};
  const fields: [string, string][] = [
    ['Texto buscado', data.query || '(sin texto; búsqueda por filtros)'],
    ['Bibliotecas consultadas', data.libraries.length ? data.libraries.join(', ') : 'Ninguna biblioteca disponible'],
    ['Ámbito de búsqueda', label(data.type, scopeNames)],
    ['Formato', label(filters.format)?.toUpperCase()],
    ['Disponibilidad', label(filters.availability, availabilityNames)],
    ['Origen', label(filters.storage_source, sourceNames)],
    ['Estado documental', label(filters.approval_status, approvalNames)],
    ['Categoría documental', filters.category_id ? data.category || 'Categoría no disponible' : ''],
    ['Tipo documental', filters.document_type_id ? data.document_type || 'Tipo no disponible' : ''],
    ['Ejercicio', label(filters.exercise)],
    ['Expediente', filters.case_id ? data.case_name || 'Expediente no disponible' : ''],
    ['Sin expediente', filters.unassigned ? 'Sí' : ''],
    ['Carpeta raíz', filters.root_id ? data.root || 'Carpeta no disponible' : ''],
    ['Vista lógica', filters.view_id ? data.view || 'Vista no disponible' : ''],
    ['Carpeta y subcarpetas', label(filters.prefix)],
    ['Solo hijos directos', filters.direct_children ? 'Sí' : ''],
  ];
  if (found !== undefined) fields.push(['Resultados encontrados', String(found)]);
  if (returned !== undefined) fields.push(['Resultados mostrados', String(returned)]);
  return <dl className="detail-grid search-audit-fields">{fields.filter(([, value]) => value).map(([name, value]) => <div className="detail-field" key={name}><dt>{name}</dt><dd>{value}</dd></div>)}</dl>;
}
