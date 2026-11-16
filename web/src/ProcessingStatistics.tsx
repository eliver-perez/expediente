import { memo, useEffect, useState } from 'react';
import { api, APIError, formatDate, getCSRF, message } from './api';

const labels = {
  total: 'Archivos registrados', linked: 'Vinculados', managed: 'Administrados',
  pending: 'Pendientes de contenido', stored_only: 'Conservados sin indexación',
  processed: 'Procesados', errors: 'Con error de extracción', native: 'Solo texto directo', ocr: 'Con OCR',
};
type Counts = Record<keyof typeof labels, number>;
interface Snapshot { stats: Counts; at: string }
interface Response { stats_available: boolean; stats?: Counts; stats_updated_at?: string; stats_error?: string }

// Memory only, bounded and scoped to the authenticated session and library.
// Returning to Processing can immediately display its last complete snapshot.
const snapshots = new Map<string, Snapshot>();
let session = '';
window.addEventListener('session-ended', () => { snapshots.clear(); session = ''; });
function cached(libraryID: string) {
  if (session !== getCSRF()) { snapshots.clear(); session = getCSRF(); }
  return snapshots.get(libraryID) ?? null;
}
function complete(value: Response): value is Response & { stats: Counts; stats_updated_at: string } {
  return value.stats_available && !!value.stats && !!value.stats_updated_at &&
    Number.isFinite(Date.parse(value.stats_updated_at)) &&
    Object.keys(labels).every(key => Number.isSafeInteger(value.stats![key as keyof Counts]) && value.stats![key as keyof Counts] >= 0);
}
const Metrics = memo(function Metrics({ stats }: { stats: Counts }) {
  return <div className="metrics processing-metrics" aria-label="Estadísticas de procesamiento">
    {Object.entries(labels).map(([key, label]) => <div className="metric" key={key}><strong>{stats[key as keyof Counts]}</strong><span>{label}</span></div>)}
  </div>;
});

export function ProcessingStatistics({ libraryID }: { libraryID: string }) {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(() => cached(libraryID));
  const [updating, setUpdating] = useState(true);
  const [problem, setProblem] = useState('');
  const [clock, setClock] = useState(Date.now());
  useEffect(() => {
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    let request: AbortController;
    const sessionAtStart = getCSRF();
    setSnapshot(cached(libraryID)); setProblem('');
    const ageTimer = setInterval(() => setClock(Date.now()), 5000);
    async function load() {
      request = new AbortController();
      // Shorter than the general API timeout; activity polling is independent.
      const timeout = setTimeout(() => request.abort(), 8000);
      setUpdating(true);
      try {
        const value = await api<Response>(`/libraries/${libraryID}/processing/stats`, { signal: request.signal });
        if (stopped || sessionAtStart !== getCSRF()) return;
        if (!complete(value)) throw new Error(value.stats_error || 'El servidor todavía no dispone de estadísticas completas. Se reintentará automáticamente.');
        const previous = snapshots.get(libraryID);
        const unchanged = previous && Object.keys(labels).every(key => previous.stats[key as keyof Counts] === value.stats[key as keyof Counts]);
        const next = { stats: unchanged ? previous.stats : value.stats, at: value.stats_updated_at };
        snapshots.delete(libraryID); snapshots.set(libraryID, next);
        if (snapshots.size > 32) snapshots.delete(snapshots.keys().next().value!);
        setSnapshot(next); setProblem(''); setClock(Date.now());
      } catch (cause) {
        if (stopped || sessionAtStart !== getCSRF()) return;
        if (cause instanceof APIError && [401, 403, 404].includes(cause.status)) {
          snapshots.delete(libraryID); setSnapshot(null);
        }
        setProblem(request.signal.aborted ? 'El servidor superó el tiempo de espera. Se reintentará automáticamente.' : message(cause));
      } finally {
        clearTimeout(timeout);
        if (!stopped) { setUpdating(false); timer = setTimeout(() => void load(), 10000); }
      }
    }
    void load();
    return () => { stopped = true; request?.abort(); clearTimeout(timer); clearInterval(ageTimer); };
  }, [libraryID]);
  const stale = !!snapshot && (!!problem || clock - Date.parse(snapshot.at) > 60000);
  return <>
    {snapshot && <Metrics stats={snapshot.stats} />}
    <p className="muted" role="status">
      {snapshot ? <>
        {stale ? 'Datos desactualizados. ' : ''}
        Última obtención: <time dateTime={snapshot.at}>{formatDate(snapshot.at)}</time>.
        {updating ? ' Actualizando estadísticas…' : ''}
        {problem ? ` ${problem}` : stale ? ' No se han obtenido estadísticas recientes; se reintentará automáticamente.' : ''}
      </> : problem ? `No hay estadísticas válidas todavía. ${problem}` : 'Cargando estadísticas…'}
    </p>
  </>;
}
