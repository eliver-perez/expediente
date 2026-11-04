import { useEffect, useRef, useState } from 'react';
import { api, formatDate, message, type Page } from './api';
import { Notice } from './components';
import { AuditDetailsButton, ModalCloseButton } from './ActionButtons';
import { SearchAuditCriteria, type SearchAuditData } from './SearchAuditCriteria';
import { useModalDialog } from './useModalDialog';

export interface Event { id: string; event_type: string; occurred_at: string; actor_kind: string; actor_user_id?: string; actor_name?: string; observed_ip?: string; library_id?: string; library_name?: string; document_id?: string; document_name?: string; details_json?: string; details?: Record<string, unknown>; search_details_json?: string | null }
interface Options { types: { event_type: string }[]; users: { id: string; name: string }[] }
const nouns: Record<string, string> = { folder:'Autorización de carpeta', configuration: 'Configuración', system: 'Sistema', installation: 'Instalación', authentication: 'Autenticación', session: 'Sesión', user: 'Usuario', permissions: 'Permisos', password: 'Contraseña', library: 'Biblioteca', storage: 'Almacenamiento', indexing: 'Procesamiento', document: 'Documento', search: 'Búsqueda', case: 'Expediente', catalog: 'Catálogo', license: 'Licencia', upload: 'Carga', review: 'Revisión', materialization: 'Incorporación', template: 'Plantilla' };
const actions: Record<string, string> = { destination_changed:'Destino predeterminado actualizado', incorporation_requested:'Incorporación solicitada al Gestor', classification_correction_requested:'Corrección de metadatos propuesta',correction_requested:'Corrección solicitada por el autor',correction_decided:'Corrección resuelta',replacement_requested:'Nueva versión propuesta',version_preserved:'Versión anterior conservada',authorization_requested:'Autorización local solicitada',authorization_checked:'Acceso del servicio comprobado', diagnostics_configured: 'Retención de diagnósticos actualizada', diagnostic_reviewed: 'Estado del diagnóstico actualizado', advanced_changed: 'Configuración avanzada actualizada', reindex_requested: 'Reindexación solicitada', previews_configured: 'Vistas previas configuradas', preview_cache_cleared: 'Caché de vistas previas vaciada', preview_requested: 'Vista previa solicitada', preview_opened: 'Vista previa consultada', processing_policy_configured: 'Pausa y límites de procesamiento actualizados', files_changed: 'Reglas de archivos actualizadas', network_change_requested: 'Cambio de acceso solicitado', opened: 'Documento abierto', downloaded: 'Descarga', index_removed: 'Retiro del índice', materialization_requested: 'Incorporación solicitada', review_submitted: 'Envío a revisión', self_review_exception: 'Excepción de revisión propia', temporary_cleaned: 'Temporal eliminado', uploaded: 'Documento cargado', root_retired: 'Carpeta retirada', view_created: 'Vista creada', permissions_changed: 'Permisos modificados', saved: 'Datos guardados', requirement_changed: 'Requisito modificado', requirements_initialized: 'Requisitos inicializados', version_created: 'Versión creada', batch_created: 'Carga creada', accepted: 'Licencia aceptada', clock_warning: 'Aviso de reloj', contact_failed: 'Fallo de conexión', deactivated: 'Desactivación', identity_recovered: 'Identidad recuperada', installation_created: 'Instalación registrada', offline_request_created: 'Solicitud sin conexión creada', online_requested: 'Activación solicitada', created: 'Creación', updated: 'Actualización', started: 'Inicio', closed: 'Cierre', success: 'Acceso correcto', failed: 'Fallo', rate_limited: 'Intentos limitados', bootstrapped: 'Administrador inicial creado', changed: 'Cambio', reset: 'Restablecimiento', local_recovery: 'Recuperación local', sessions_revoked: 'Sesiones revocadas', global_changed: 'Cambio de permisos globales', discovered: 'Documento detectado', classification: 'Clasificación actualizada', associate: 'Documento asociado al expediente', reassign: 'Documento reasignado', managed_changed: 'Original administrado modificado', managed_missing: 'Original administrado no disponible', managed_reappeared: 'Original administrado disponible nuevamente', linked_changed: 'Contenido vinculado modificado', linked_missing: 'Original no disponible', linked_reappeared: 'Original disponible nuevamente', scan_failed: 'Error de reconciliación', root_configured: 'Configuración de carpeta', root_added: 'Carpeta agregada', requested: 'Verificación solicitada', attempt_failed: 'Intento fallido', retry_requested: 'Reintento solicitado', errors_retry_requested: 'Reintento de rutas con error', cancel_requested: 'Cancelación solicitada', license_paused: 'Pausa por licencia', approved: 'Aprobación', rejected: 'Rechazo', cancelled: 'Cancelación', executed: 'Consulta realizada' };
function labels(event: Event) { const [category, ...rest] = event.event_type.split('.'); const action = rest.join('.'); return { type: nouns[category] || 'Actividad', action: actions[action] || 'Actividad registrada' }; }
function details(event: Event): Record<string, unknown> { if (event.details) return event.details; try { return JSON.parse(event.details_json || '{}'); } catch { return {}; } }
function searchDetails(event: Event): SearchAuditData | null { try { return event.search_details_json ? JSON.parse(event.search_details_json) as SearchAuditData : null; } catch { return null; } }
const documentDescriptions: Record<string, string> = { incorporation_requested:'El autor presentó explícitamente su borrador al Gestor, sin aprobarlo ni publicarlo.', classification_correction_requested:'Se propuso una corrección para revisión; los metadatos vigentes y la aprobación física se conservaron.', correction_requested:'El autor solicitó corregir su documento sin obtener permisos de edición sobre el aprobado.', correction_decided:'Se registró la decisión de una persona autorizada sobre la corrección.', replacement_requested:'Se vinculó una carga nueva como propuesta de sustitución; el original conserva su vigencia hasta su aprobación.',version_preserved:'La nueva versión fue aprobada y se conservó íntegro el archivo físico anterior.',
  reindex_requested: 'Se comprobó el original y se solicitó una nueva extracción con la configuración actual.',
  opened: 'Se abrió la vista previa del original.', downloaded: 'Se descargó el archivo original.',
  discovered: 'AIDEX detectó el documento durante un recorrido de carpeta.',
  linked_changed: 'Cambió el contenido del original vinculado; se registró una nueva versión.',
  linked_missing: 'El original vinculado dejó de estar disponible.', linked_reappeared: 'El original vinculado volvió a estar disponible.',
  managed_changed: 'La verificación detectó cambios en el original administrado.', managed_missing: 'No se encontró el original administrado.', managed_reappeared: 'El original administrado volvió a estar disponible.',
  uploaded: 'Se recibió el documento mediante una carga.', classification: 'Se actualizaron la clasificación o los datos del documento.',
  associate: 'Se incorporó la referencia del documento a un expediente.', reassign: 'Se cambió el expediente asociado al documento.',
  review_submitted: 'El documento se envió a revisión.', self_review_exception: 'Se aplicó una excepción autorizada de revisión propia.',
  approved: 'Se aprobó el documento.', rejected: 'Se rechazó el documento.', cancelled: 'Se canceló la carga del documento.',
  materialization_requested: 'Se solicitó guardar el documento en el destino administrado.',
  temporary_cleaned: 'Se eliminó una carga temporal al finalizar su plazo de conservación.',
  index_removed: 'El documento se retiró de las búsquedas y del listado; su archivo original se conservó.'
};
function description(event: Event) {
  const data = details(event); const reason = typeof data.reason === 'string' ? data.reason : '';
  if (event.event_type === 'search.executed') { const search = searchDetails(event); return search ? `Consulta ${search.query ? `«${search.query}»` : 'por filtros'} · ${data.result_count ?? 0} resultados` : 'Consulta registrada; los criterios originales no están disponibles.'; }
  if (event.event_type.startsWith('document.')) {
    const explanation = documentDescriptions[event.event_type.slice(9)] || 'Se registró una acción sobre el documento.';
    return reason ? `${explanation} Motivo: ${reason}` : explanation;
  }
  const names = [data.name, data.filename, data.title].filter(value => typeof value === 'string');
  return reason || names.join(' · ') || (event.document_id ? event.document_name || 'Acción registrada sobre un documento' : event.library_name ? `En ${event.library_name}` : 'Acción registrada en la aplicación');
}
function localDate(date: Date) { return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16); }
function defaults() { const end = new Date(); const start = new Date(end); start.setMonth(start.getMonth() - 1); return { from: localDate(start), to: localDate(new Date(end.getTime() + 60000)), event_type: '', actor_user_id: '', q: '' }; }
const fieldNames: Record<string, string> = { previous:'Valores anteriores', new:'Valores nuevos', proposed:'Valores propuestos', content_approval_preserved:'Aprobación del contenido conservada', original_preserved:'Original conservado', reason: 'Motivo', error_code: 'Código de diagnóstico', attempt: 'Intento', name: 'Nombre', filename: 'Archivo', title: 'Título', enabled: 'Habilitado', interval: 'Intervalo en segundos', paths_count: 'Rutas seleccionadas', mode: 'Modalidad', role_ids: 'Perfiles', result_count: 'Resultados encontrados', returned_count: 'Resultados mostrados', outcome: 'Resultado', scope: 'Alcance', status: 'Estado', documents_preserved: 'Documentos conservados' };
const hiddenFields = new Set(['result_count', 'returned_count', 'request_id', 'session_id', 'path_sha256', 'operation_id', 'content_version_id', 'document_id', 'user_id', 'library_id', 'root_id', 'job_id', 'retry_of']);
const roleNames: Record<string, string> = { installation_admin: 'Administrador de instalación', access_auditor: 'Auditor de accesos', event_auditor: 'Auditor de eventos', library_reader: 'Lector', library_manager: 'Gestor', library_auditor: 'Auditor de biblioteca', library_contributor: 'Colaborador', library_reviewer: 'Revisor' };
function readableValue(value: unknown, key: string): string | null {
  if (value == null) return null;
  if (typeof value === 'boolean') return value ? 'Sí' : 'No';
  if (Array.isArray(value)) return value.map(item => typeof item === 'string' ? key === 'role_ids' ? roleNames[item] || item : item : String(item)).join(', ') || null;
  if(typeof value==='object'&&['previous','new','proposed'].includes(key)){
    const data=value as Record<string,unknown>;
    const labels=[data.case_identifier&&`Expediente: ${data.case_identifier}`,data.category_name&&`Categoría: ${data.category_name}`,data.document_type_name&&`Tipo: ${data.document_type_name}`,data.title&&`Título: ${data.title}`].filter(Boolean);
    if(data.metadata&&typeof data.metadata==='object')labels.push(...Object.entries(data.metadata).map(([name,text])=>`${name}: ${String(text)}`));
    return labels.join(' · ')||'Sin clasificación adicional';
  }
  if (typeof value === 'object') return Object.entries(value).map(([name, item]) => `${fieldNames[name] || name.replaceAll('_', ' ')}: ${String(item)}`).join(' · ') || null;
  if (typeof value === 'string' && /^[0-9a-f]{8}-[0-9a-f-]{27,}$/i.test(value)) return null;
  return String(value);
}
export function AuditTable({ libraryID }: { libraryID?: string }) {
  const endpoint = libraryID ? `/libraries/${libraryID}/audit-events` : '/audit-events';
  const [filters, setFilters] = useState(defaults); const [applied, setApplied] = useState(filters);
  const [sort, setSort] = useState('date'); const [direction, setDirection] = useState('desc');
  const [cursor, setCursor] = useState(''); const [previous, setPrevious] = useState<string[]>([]);
  const [options, setOptions] = useState<Options>({ types: [], users: [] }); const [page, setPage] = useState<Page<Event> | null>(null);
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [selected, setSelected] = useState<Event | null>(null);
  useEffect(() => { const controller = new AbortController(); api<Options>(libraryID ? `/libraries/${libraryID}/audit-options` : '/audit-options', { signal: controller.signal }).then(setOptions).catch(cause => { if (!controller.signal.aborted) setError(message(cause)); }); return () => controller.abort(); }, [libraryID]);
  useEffect(() => {
    const controller = new AbortController(); setBusy(true); setError('');
    const parameters = new URLSearchParams({ ...applied, from: new Date(applied.from).toISOString(), to: new Date(applied.to).toISOString(), sort, direction, cursor, limit: '25' });
    api<Page<Event>>(`${endpoint}?${parameters}`, { signal: controller.signal }).then(setPage).catch(cause => { if (!controller.signal.aborted) setError(message(cause)); }).finally(() => { if (!controller.signal.aborted) setBusy(false); });
    return () => controller.abort();
  }, [endpoint, applied, sort, direction, cursor]);
  function firstPage() { setCursor(''); setPrevious([]); }
  function order(column: string) { setDirection(sort === column && direction === 'desc' ? 'asc' : 'desc'); setSort(column); firstPage(); }
  return <section className="card"><h2>{libraryID ? 'Eventos de biblioteca' : 'Visor de eventos'}</h2><p className="muted">Acciones de personas y del sistema. Fechas expresadas en tu zona horaria.</p>
    <form className="filters audit-filters" onSubmit={event => { event.preventDefault(); if (filters.from >= filters.to) { setError('La fecha inicial debe ser anterior a la final.'); return; } setApplied({ ...filters }); firstPage(); }}>
      <label>Desde<input type="datetime-local" required value={filters.from} onChange={event => setFilters({ ...filters, from: event.target.value })} /></label><label>Hasta<input type="datetime-local" required value={filters.to} onChange={event => setFilters({ ...filters, to: event.target.value })} /></label>
      <label>Tipo de evento<select value={filters.event_type} onChange={event => setFilters({ ...filters, event_type: event.target.value })}><option value="">Mostrar todo</option>{options.types.map(type => { const label = labels({ event_type: type.event_type } as Event); return <option key={type.event_type} value={type.event_type}>{label.type} · {label.action}</option>; })}</select></label>
      <label>Usuario<select value={filters.actor_user_id} onChange={event => setFilters({ ...filters, actor_user_id: event.target.value })}><option value="">Mostrar todo</option>{options.users.map(user => <option key={user.id} value={user.id}>{user.name}</option>)}</select></label>
      <label>Buscar<input value={filters.q} maxLength={200} placeholder="Persona, evento o detalle" onChange={event => setFilters({ ...filters, q: event.target.value })} /></label><div className="actions"><button disabled={busy}>Aplicar filtros</button><button type="button" className="secondary" onClick={() => { const value = defaults(); setFilters(value); setApplied(value); setSort('date'); setDirection('desc'); firstPage(); }}>Restablecer</button></div>
    </form><Notice text={error} />
    <div className="table-wrap"><table><thead><tr>{[['date', 'Fecha y hora'], ['actor', 'Usuario'], ['event', 'Tipo de evento']].map(([key, text]) => <th key={key} aria-sort={sort === key ? direction === 'asc' ? 'ascending' : 'descending' : 'none'}><button className="text-button" onClick={() => order(key)}>{text} {sort === key ? direction === 'asc' ? '↑' : '↓' : ''}</button></th>)}<th>Acción realizada</th><th>Descripción</th><th>Detalles</th></tr></thead><tbody>{page?.items.map(event => { const label = labels(event); return <tr key={event.id}><td>{formatDate(event.occurred_at)}</td><td>{actorName(event)}</td><td>{label.type}</td><td>{label.action}</td><td>{description(event)}</td><td><AuditDetailsButton onClick={() => setSelected(event)} /></td></tr>; })}</tbody></table></div>
    {busy && <p role="status">Consultando eventos…</p>}{!busy && page?.items.length === 0 && <p className="empty-state">No hay eventos que coincidan con estos filtros.</p>}
    <div className="actions"><button className="secondary" disabled={busy || !previous.length} onClick={() => { setCursor(previous[previous.length - 1]); setPrevious(previous.slice(0, -1)); }}>Anterior</button><span>Página {previous.length + 1}</span><button className="secondary" disabled={busy || !page?.next_cursor} onClick={() => { setPrevious([...previous, cursor]); setCursor(page?.next_cursor || ''); }}>Siguiente</button></div>
    {selected && <AuditDetail event={selected} close={() => setSelected(null)} />}
  </section>;
}
export function AuditDetail({ event, close }: { event: Event; close: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  useModalDialog(dialog, close);
  const label = labels(event); const values = details(event); const search = searchDetails(event);
  return <dialog ref={dialog} className="document-dialog audit-dialog" aria-labelledby="audit-detail-heading"><div className="section-heading"><h2 id="audit-detail-heading">{label.action}</h2><ModalCloseButton aria-label="Cerrar detalle del evento" autoFocus onClick={close} /></div><dl className="detail-grid"><dt>Fecha y hora</dt><dd>{formatDate(event.occurred_at)}</dd><dt>Usuario responsable</dt><dd>{actorName(event)}</dd><dt>Tipo de evento</dt><dd>{label.type}</dd><dt>Acción</dt><dd>{label.action}</dd><dt>Descripción</dt><dd>{description(event)}</dd>{event.observed_ip && <><dt>IP observada</dt><dd>{event.observed_ip}</dd></>}{event.library_id && <><dt>Biblioteca</dt><dd>{event.library_name || 'Biblioteca no disponible'}</dd></>}{event.document_id && <><dt>Documento relacionado</dt><dd>{event.document_name || 'Documento no disponible'}</dd></>}{event.event_type === 'search.executed' && !search && <><dt>Resultados encontrados</dt><dd>{String(values.result_count ?? 'No disponible')}</dd><dt>Resultados mostrados</dt><dd>{String(values.returned_count ?? 'No disponible')}</dd></>}{event.event_type !== 'search.executed' && Object.entries(values).filter(([key]) => !hiddenFields.has(key) && !key.endsWith('_id')).map(([key, value]) => { const display = readableValue(value, key); return display === null ? null : <div className="detail-field" key={key}><dt>{fieldNames[key] || key.replaceAll('_', ' ')}</dt><dd>{display}</dd></div>; })}</dl>{search && <SearchAuditCriteria data={search} found={Number(values.result_count ?? 0)} returned={typeof values.returned_count === 'number' ? values.returned_count : undefined} />}</dialog>;
}

export function actorName(event: Event) { return event.actor_name || (event.actor_kind === 'system' ? 'Sistema' : event.actor_kind === 'anonymous' ? 'Sin sesión' : 'Usuario no disponible'); }
export { labels as eventLabels, description as eventDescription };
