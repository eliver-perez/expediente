import { useEffect, useRef, useState } from 'react';
import { api, message, type Page } from './api';

export function Notice({ text, kind = 'error' }: { text: string; kind?: 'error' | 'success' }) {
  return text ? <div className={`notice ${kind}`} role={kind === 'error' ? 'alert' : 'status'}>{text}</div> : null;
}

export function usePage<T>(endpoint: string) {
  const [items, setItems] = useState<T[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const previousEndpoint = useRef(endpoint);
  useEffect(() => {
    const controller = new AbortController();
    setBusy(true); setError('');
    if (previousEndpoint.current !== endpoint) { setItems([]); setCursor(null); previousEndpoint.current = endpoint; }
    api<Page<T>>(endpoint, { signal: controller.signal }).then(page => {
      setItems(page.items); setCursor(page.next_cursor);
    }).catch(error => { if (!controller.signal.aborted) setError(message(error)); })
      .finally(() => { if (!controller.signal.aborted) setBusy(false); });
    return () => controller.abort();
  }, [endpoint, revision]);
  async function more() {
    if (!cursor || busy) return;
    setBusy(true); setError('');
    try {
      const page = await api<Page<T>>(`${endpoint}${endpoint.includes('?') ? '&' : '?'}cursor=${encodeURIComponent(cursor)}`);
      setItems(previous => [...previous, ...page.items]); setCursor(page.next_cursor);
    } catch (error) { setError(message(error)); } finally { setBusy(false); }
  }
  return { items, busy, error, cursor, more, reload: () => setRevision(value => value + 1) };
}

export function PageEnd({ busy, cursor, more, empty }: { busy: boolean; cursor: string | null; more: () => void; empty: boolean }) {
  return <div className="page-end">
    {busy ? <p role="status">Cargando…</p> : empty ? <p>No hay registros para mostrar.</p> : null}
    {cursor && <button className="secondary" disabled={busy} onClick={more}>Mostrar más</button>}
  </div>;
}
