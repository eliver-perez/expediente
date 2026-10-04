import { Accordion } from './Accordion';
import { useEffect, useRef, useState } from 'react';
import { api, formatDate, message, type Page } from './api';
import { Notice } from './components';

export interface Event { id: string; event_type: string; occurred_at: string; actor_kind: string; actor_user_id?: string; actor_name?: string; library_id?: string; library_name?: string; document_id?: string; document_name?: string; details_json?: string; details?: Record<string, unknown> }
interface Options { types: { event_type: string }[]; users: { id: string; name: string }[] }
const nouns: Record<string, string> = { configuration: 'Configuración', system: 'Sistema', installation: 'Instalación', authentication: 'Autenticación', session: 'Sesión', user: 'Usuario', permissions: 'Permisos', password: 'Contraseña', library: 'Biblioteca', storage: 'Almacenamiento', indexing: 'Procesamiento', document: 'Documento', search: 'Búsqueda', case: 'Expediente', catalog: 'Catálogo', license: 'Licencia', upload: 'Carga', review: 'Revisión', materialization: 'Incorporación', template: 'Plantilla' };
const actions: Record<string, string> = { processing_policy_configured: 'Pausa y límites de procesamiento actualizados', files_changed: 'Reglas de archivos actualizadas', network_change_requested: 'Cambio de acceso solicitado', opened: 'Documento abierto', downloaded: 'Descarga', index_removed: 'Retiro del índice', materialization_requested: 'Incorporación solicitada', review_submitted: 'Envío a revisión', self_review_exception: 'Excepción de revisión propia', temporary_cleaned: 'Temporal eliminado', uploaded: 'Documento cargado', root_retired: 'Carpeta retirada', view_created: 'Vista creada', permissions_changed: 'Permisos modificados', saved: 'Datos guardados', requirement_changed: 'Requisito modificado', requirements_initialized: 'Requisitos inicializados', version_created: 'Versión creada', batch_created: 'Lote de carga creado', accepted: 'Licencia aceptada', clock_warning: 'Aviso de reloj', contact_failed: 'Fallo de conexión', deactivated: 'Desactivación', identity_recovered: 'Identidad recuperada', installation_created: 'Instalación registrada', offline_request_created: 'Solicitud sin conexión creada', online_requested: 'Activación solicitada', created: 'Creación', updated: 'Actualización', started: 'Inicio', closed: 'Cierre', success: 'Acceso correcto', failed: 'Fallo', rate_limited: 'Intentos limitados', bootstrapped: 'Administrador inicial creado', changed: 'Cambio', reset: 'Restablecimiento', local_recovery: 'Recuperación local', sessions_revoked: 'Sesiones revocadas', global_changed: 'Cambio de permisos globales', discovered: 'Documento detectado', classification: 'Clasificación actualizada', associate: 'Documento asociado al expediente', reassign: 'Documento reasignado', managed_changed: 'Original administrado modificado', managed_missing: 'Original administrado no disponible', managed_reappeared: 'Original administrado disponible nuevamente', linked_changed: 'Contenido vinculado modificado', linked_missing: 'Original no disponible', linked_reappeared: 'Original disponible nuevamente', scan_failed: 'Error de reconciliación', root_configured: 'Configuración de carpeta', root_added: 'Carpeta agregada', requested: 'Verificación solicitada', attempt_failed: 'Intento fallido', retry_requested: 'Reintento solicitado', errors_retry_requested: 'Reintento de rutas con error', cancel_requested: 'Cancelación solicitada', license_paused: 'Pausa por licencia', approved: 'Aprobación', rejected: 'Rechazo', cancelled: 'Cancelación', executed: 'Consulta realizada' };
function labels(event: Event) { const [category, ...rest] = event.event_type.split('.'); const action = rest.join('.'); return { type: nouns[category] || 'Actividad', action: actions[action] || 'Actividad registrada' }; }
function details(event: Event): Record<string, unknown> { if (event.details) return event.details; try { return JSON.parse(event.details_json || '{}'); } catch { return {}; } }
const documentDescriptions: Record<string, string> = {
  opened: 'Se abrió la vista previa del original.', downloaded: 'Se descargó el archivo original.',
  discovered: 'AIBID detectó el documento durante un recorrido de carpeta.',
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
  if (event.event_type.startsWith('document.')) {
    const explanation = documentDescriptions[event.event_type.slice(9)] || 'Se registró una acción sobre el documento.';
    return reason ? `${explanation} Motivo: ${reason}` : explanation;
  }
  const names = [data.name, data.filename, data.title].filter(value => typeof value === 'string');
  return reason || names.join(' · ') || (event.document_id ? event.document_name || 'Acción registrada sobre un documento' : event.library_name ? `En ${event.library_name}` : 'Acción registrada en la aplicación');
}
function localDate(date: Date) { return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16); }
function defaults() { const end = new Date(); const start = new Date(end); start.setMonth(start.getMonth() - 1); return { from: localDate(start), to: localDate(new Date(end.getTime() + 60000)), event_type: '', actor_user_id: '', q: '' }; }
const fieldNames: Record<string, string> = { reason: 'Motivo', root_id: 'Carpeta registrada', job_id: 'Trabajo', retry_of: 'Trabajo original', error_code: 'Código de diagnóstico', attempt: 'Intento', content_version_id: 'Versión de contenido', document_id: 'Documento', name: 'Nombre', filename: 'Archivo', enabled: 'Habilitado', interval: 'Intervalo en segundos', paths_count: 'Rutas seleccionadas', library_id: 'Biblioteca', mode: 'Modalidad', user_id: 'Usuario', role_ids: 'Perfiles' };
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
    <div className="table-wrap"><table><thead><tr>{[['date', 'Fecha y hora'], ['actor', 'Usuario'], ['event', 'Tipo de evento']].map(([key, text]) => <th key={key} aria-sort={sort === key ? direction === 'asc' ? 'ascending' : 'descending' : 'none'}><button className="text-button" onClick={() => order(key)}>{text} {sort === key ? direction === 'asc' ? '↑' : '↓' : ''}</button></th>)}<th>Acción realizada</th><th>Descripción</th><th>Detalles</th></tr></thead><tbody>{page?.items.map(event => { const label = labels(event); return <tr key={event.id}><td>{formatDate(event.occurred_at)}</td><td>{actorName(event)}</td><td>{label.type}</td><td>{label.action}</td><td>{description(event)}</td><td><button className="text-button" onClick={() => setSelected(event)}>Ver detalles</button></td></tr>; })}</tbody></table></div>
    {busy && <p role="status">Consultando eventos…</p>}{!busy && page?.items.length === 0 && <p className="empty-state">No hay eventos que coincidan con estos filtros.</p>}
    <div className="actions"><button className="secondary" disabled={busy || !previous.length} onClick={() => { setCursor(previous[previous.length - 1]); setPrevious(previous.slice(0, -1)); }}>Anterior</button><span>Página {previous.length + 1}</span><button className="secondary" disabled={busy || !page?.next_cursor} onClick={() => { setPrevious([...previous, cursor]); setCursor(page?.next_cursor || ''); }}>Siguiente</button></div>
    {selected && <AuditDetail event={selected} close={() => setSelected(null)} />}
  </section>;
}
export function AuditDetail({ event, close }: { event: Event; close: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null); useEffect(() => { const element = dialog.current; element?.showModal(); return () => element?.close(); }, []);
  const label = labels(event); const values = details(event);
  return <dialog ref={dialog} className="document-dialog audit-dialog" onCancel={close} aria-labelledby="audit-detail-heading"><div className="section-heading"><h2 id="audit-detail-heading">{label.action}</h2><button className="secondary" autoFocus onClick={close}>Cerrar detalle</button></div><dl className="detail-grid"><dt>Fecha y hora</dt><dd>{formatDate(event.occurred_at)}</dd><dt>Usuario responsable</dt><dd>{actorName(event)}</dd><dt>Tipo de evento</dt><dd>{label.type}</dd><dt>Acción</dt><dd>{label.action}</dd><dt>Descripción</dt><dd>{description(event)}</dd>{event.library_id && <><dt>Biblioteca</dt><dd>{event.library_name || event.library_id}</dd></>}{event.document_id && <><dt>Documento relacionado</dt><dd>{event.document_name || event.document_id}</dd></>}{Object.entries(values).filter(([key, value]) => value !== null && typeof value !== 'object' && !key.endsWith('_id')).map(([key, value]) => <div className="detail-field" key={key}><dt>{fieldNames[key] || key.replaceAll('_', ' ')}</dt><dd>{typeof value === 'boolean' ? value ? 'Sí' : 'No' : String(value)}</dd></div>)}</dl><Accordion title="Información avanzada · registro original"><pre className="audit-json">{JSON.stringify({ ...event, details: values, details_json: undefined }, null, 2)}</pre></Accordion></dialog>;
}

export function actorName(event: Event) { return event.actor_name || (event.actor_kind === 'system' ? 'Sistema' : event.actor_kind === 'anonymous' ? 'Sin sesión' : 'Usuario no disponible'); }
export { labels as eventLabels, description as eventDescription };
