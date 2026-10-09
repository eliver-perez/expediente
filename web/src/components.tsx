import { useEffect, useRef, useState } from 'react';
import { api, message, type Page } from './api';

export function Notice({ text, kind = 'error' }: { text: string; kind?: 'error' | 'success' }) {
  return text ? <div className={`notice ${kind}`} role={kind === 'error' ? 'alert' : 'status'}>{text}</div> : null;
}

export function usePage<T>(endpoint: string, enabled = true) {
  const [items, setItems] = useState<T[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const previousEndpoint = useRef(endpoint);
  const activeRequest = useRef<AbortController | null>(null);
  useEffect(() => {
    if (!enabled) { setBusy(false); return; }
    const controller = new AbortController();
    activeRequest.current?.abort(); activeRequest.current = controller;
    setBusy(true); setError('');
    if (previousEndpoint.current !== endpoint) { setItems([]); setCursor(null); previousEndpoint.current = endpoint; }
    api<Page<T>>(endpoint, { signal: controller.signal }).then(page => {
      if (controller.signal.aborted) return;
      setItems(page.items); setCursor(page.next_cursor);
    }).catch(error => { if (!controller.signal.aborted) setError(message(error)); })
      .finally(() => { if (!controller.signal.aborted) setBusy(false); });
    return () => { activeRequest.current?.abort(); activeRequest.current = null; };
  }, [endpoint, revision, enabled]);
  async function more() {
    if (!cursor || busy || activeRequest.current?.signal.aborted) return;
    // More belongs to the same scope as the first page, including reloads.
    const controller = activeRequest.current;
    if (!controller) return;
    setBusy(true); setError('');
    try {
      const page = await api<Page<T>>(`${endpoint}${endpoint.includes('?') ? '&' : '?'}cursor=${encodeURIComponent(cursor)}`, { signal: controller.signal });
      if (controller.signal.aborted) return;
      setItems(previous => [...previous, ...page.items]); setCursor(page.next_cursor);
    } catch (error) { if (!controller.signal.aborted) setError(message(error)); }
    finally { if (!controller.signal.aborted) setBusy(false); }
  }
  return { items, busy, error, cursor, more, reload: () => setRevision(value => value + 1) };
}

export function PageEnd({ busy, cursor, more, empty }: { busy: boolean; cursor: string | null; more: () => void; empty: boolean }) {
  return <div className="page-end">
    {busy ? <p role="status">Cargando…</p> : empty ? <p>No hay registros para mostrar.</p> : null}
    {cursor && <button className="secondary" disabled={busy} onClick={more}>Mostrar más</button>}
  </div>;
}

// Fetch again only after the previous request settles; abort on navigation.
export function usePolling<T>(endpoint: string, interval = 2500) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    const controller = new AbortController(); let timer: ReturnType<typeof setTimeout>;
    setData(null);
    async function load() {
      try { const value = await api<T>(endpoint, { signal: controller.signal }); if (!controller.signal.aborted) { setData(value); setError(''); } }
      catch (cause) { if (!controller.signal.aborted) setError(message(cause)); }
      finally { if (!controller.signal.aborted) timer = setTimeout(() => void load(), interval); }
    }
    void load(); return () => { controller.abort(); clearTimeout(timer); };
  }, [endpoint, interval, revision]);
  return { data, error, reload: () => setRevision(value => value + 1) };
}
