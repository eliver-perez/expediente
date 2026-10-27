import { useState } from 'react';
import { formatDate } from './api';
import { Notice, usePolling } from './components';
import { diagnosticLabel, jobLabel, processingStateLabel, type Job } from './libraryTypes';

export function PageControls({ page, total, size, busy = false, changePage, changeSize }: { page: number; total: number; size: number; busy?: boolean; changePage: (n: number) => void; changeSize: (n: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / size));
  return <div className="compact-pagination"><label>Registros por página<select value={size} disabled={busy} onChange={e => changeSize(Number(e.target.value))}>{[10, 25, 50, 100].map(n => <option key={n} value={n}>{n}</option>)}</select></label><span>{total.toLocaleString('es-MX')} registros · Página {page} de {pages}</span><div className="actions"><button className="secondary compact" disabled={busy || page <= 1} onClick={() => changePage(page - 1)}>Anterior</button><button className="secondary compact" disabled={busy || page >= pages} onClick={() => changePage(page + 1)}>Siguiente</button></div></div>;
}
export function FullPath({ value }: { value: string }) {
  if (!value) return <>—</>;
  return <details className="compact-path"><summary title={value}><span>{value}</span></summary><p className="path-text">{value}</p></details>;
}
interface TablePage<T> { items: T[]; total: number; page: number; page_size: number }
interface Skip { root_id: string; relative_path: string; error_code: string; occurred_at: string }
const kinds: Record<string, string> = { scan: 'Reconciliar carpeta', extract: 'Extraer texto y OCR', verify_managed: 'Verificar integridad', materialize: 'Guardar documento definitivo' };
const omissionLabels = Object.entries(diagnosticLabel).filter(([key]) => (key.startsWith('WATCH_') && key.endsWith('_IGNORED')) || ['WATCH_OFFICE_TEMPORARY', 'FILE_TYPE_BLOCKED', 'FILE_TYPE_UNKNOWN', 'FILE_TYPE_MISMATCH', 'FILE_FORMAT_DISABLED', 'FILE_NOT_INDEXABLE', 'INVALID_OFFICE_DOCUMENT', 'DOCUMENT_COMPLEXITY_LIMIT'].includes(key));
export function ProcessingTable({ libraryID, skips = false, retry, action, busy, revision }: { libraryID: string; skips?: boolean; retry?: boolean; action?: (path: string, body?: object) => Promise<void>; busy?: boolean; revision: number }) {
  const [filters, setFilters] = useState({ q: '', state: '', kind: '', from: '', until: '', sort: 'date', direction: 'desc', page: 1, page_size: 10 });
  const change = (key: string, value: string | number) => setFilters(previous => ({ ...previous, [key]: value, page: key === 'page' ? Number(value) : 1 }));
  const query = new URLSearchParams(Object.entries(filters).map(([k, v]) => [k, String(v)])); query.set('refresh', String(revision));
  const result = usePolling<TablePage<Job | Skip>>(`/libraries/${libraryID}/${skips ? 'file-skips' : 'job-history'}?${query}`, 5000);
  const data = result.data;
  return <section className={`card compact-section ${skips ? 'file-skips' : 'job-history'}`}><h2>{skips ? 'Archivos omitidos por las reglas' : 'Cola e historial de tareas'}</h2>
    {skips && <p className="muted">Omisiones conservadas del recorrido más reciente de cada carpeta. Este registro se actualiza al verificar; no es un historial permanente. Los originales permanecen intactos.</p>}
    <Notice text={result.error} />
    <form className="filters compact-filters" onSubmit={e => { e.preventDefault(); change('q', String(new FormData(e.currentTarget).get('query') || '')); }}>
      <label>Buscar {skips ? 'archivo o ruta' : 'archivo, carpeta o diagnóstico'}<input name="query" maxLength={240} type="search" /></label><button className="secondary compact">Buscar</button>
      <label>{skips ? 'Motivo de omisión' : 'Estado de tarea'}<select value={filters.state} onChange={e => change('state', e.target.value)}><option value="">Todos</option>{(skips ? omissionLabels : Object.entries(jobLabel)).map(([key, label]) => <option value={key} key={key}>{label}</option>)}</select></label>
      {!skips && <><label>Tipo de tarea<select value={filters.kind} onChange={e => change('kind', e.target.value)}><option value="">Todos</option>{Object.entries(kinds).map(([key, label]) => <option value={key} key={key}>{label}</option>)}</select></label><label>Desde (UTC)<input type="date" value={filters.from} onChange={e => change('from', e.target.value)} /></label><label>Hasta (UTC)<input type="date" value={filters.until} onChange={e => change('until', e.target.value)} /></label></>}
      <label>Ordenar por<select value={filters.sort} onChange={e => change('sort', e.target.value)}><option value="date">Fecha</option><option value={skips ? 'path' : 'name'}>{skips ? 'Ruta' : 'Archivo'}</option><option value={skips ? 'reason' : 'state'}>{skips ? 'Motivo' : 'Estado'}</option>{!skips && <option value="kind">Tipo de tarea</option>}</select></label><label>Dirección<select value={filters.direction} onChange={e => change('direction', e.target.value)}><option value="desc">Descendente</option><option value="asc">Ascendente</option></select></label>
    </form>
    {!data && !result.error && <p role="status">Consultando registros…</p>}
    {data && <><div className="table-wrap"><table className="compact-table" aria-label={skips ? 'Omisiones de archivos' : 'Historial de tareas'}><thead><tr>{(skips ? ['Ruta relativa', 'Motivo', 'Fecha'] : ['Archivo o carpeta', 'Tarea', 'Estado', 'Intentos', 'Diagnóstico y detalles', 'Acciones']).map(label => <th key={label}>{label}</th>)}</tr></thead><tbody>{data.items.map(item => {
      if ('error_code' in item) return <tr key={item.root_id + item.relative_path}><td><FullPath value={item.relative_path} /></td><td>{diagnosticLabel[item.error_code] || item.error_code}</td><td>{formatDate(item.occurred_at)}</td></tr>;
      const job = item; return <tr key={job.id}><td><FullPath value={job.filename || job.root_path || 'Tarea de biblioteca'} /><small>{formatDate(job.created_at)}</small></td><td>{kinds[job.job_type] || job.job_type}</td><td>{processingStateLabel[job.processing_state] || jobLabel[job.status]}</td><td className="numeric">{job.attempt_count} / {job.maximum_attempts}</td><td><details><summary>{diagnosticLabel[job.last_error_code] || job.last_error_code || 'Ver detalles'}</summary><p className="path-text">{job.root_path} {job.relative_path}</p><p>Referencia: {job.id}</p><p>Inicio: {formatDate(job.started_at || null)} · Fin: {formatDate(job.finished_at || null)}</p>{job.status === 'retry_wait' && <p>Próximo intento: {formatDate(job.available_at)}</p>}</details></td><td><div className="actions">{retry && ['failed', 'cancelled'].includes(job.status) && <button className="text-button" disabled={busy} onClick={() => void action?.(`/jobs/${job.id}/retry`, { reason: 'Reintento solicitado desde Procesamiento' })}>Reintentar</button>}{['queued', 'retry_wait', 'paused'].includes(job.status) && ['scan', 'extract'].includes(job.job_type) && <button className="text-button" disabled={busy} onClick={() => void action?.(`/jobs/${job.id}/cancel`)}>Cancelar definitivamente</button>}</div></td></tr>;
    })}</tbody></table></div>{!data.items.length && <p>No hay {skips ? 'omisiones' : 'tareas'} que coincidan con los filtros.</p>}
      <PageControls page={data.page} total={data.total} size={filters.page_size} changePage={page => change('page', page)} changeSize={size => change('page_size', size)} /></>}
  </section>;
}
